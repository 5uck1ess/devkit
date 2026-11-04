package mcp

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/5uck1ess/devkit/lib"
	mcpmcp "github.com/mark3labs/mcp-go/mcp"
)

// flowHarness drives devkit_start / devkit_advance against one workflow file.
type flowHarness struct {
	t       *testing.T
	srv     *Server
	dataDir string
	session string
}

func newFlow(t *testing.T, yml, input string) (*flowHarness, string) {
	t.Helper()
	wfDir, dataDir := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(wfDir, "wf.yml"), yml)
	h := &flowHarness{t: t, srv: newTestServer(t, dataDir, wfDir), dataDir: dataDir}
	_, start := h.srv.startTool()
	req := mcpmcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"workflow": "wf", "input": input}
	res, err := start(context.Background(), req)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	text := resultText(t, res)
	if res.IsError {
		t.Fatalf("start tool error: %s", text)
	}
	st, err := lib.ReadSessionJSON(dataDir)
	if err != nil || st == nil {
		t.Fatalf("read session: %v", err)
	}
	h.session = st.ID
	return h, text
}

// advance calls devkit_advance with extra args; returns text and IsError.
func (h *flowHarness) advance(args map[string]interface{}) (string, bool) {
	h.t.Helper()
	_, adv := h.srv.advanceTool()
	req := mcpmcp.CallToolRequest{}
	a := map[string]interface{}{"session": h.session}
	for k, v := range args {
		a[k] = v
	}
	req.Params.Arguments = a
	res, err := adv(context.Background(), req)
	if err != nil {
		h.t.Fatalf("advance: %v", err)
	}
	return resultText(h.t, res), res.IsError
}

func (h *flowHarness) state() *lib.SessionState {
	h.t.Helper()
	st, err := lib.ReadSessionJSON(h.dataDir)
	if err != nil {
		h.t.Fatalf("read session: %v", err)
	}
	return st
}

func resultText(t *testing.T, res *mcpmcp.CallToolResult) string {
	t.Helper()
	if res == nil || len(res.Content) == 0 {
		t.Fatal("empty result")
	}
	tc, ok := res.Content[0].(mcpmcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type %T", res.Content[0])
	}
	return tc.Text
}

// tri-review layout: parallel children come BEFORE their dispatcher.
const triLayout = `name: tri
description: children before dispatcher
steps:
  - id: gather
    prompt: Gather {{input}}
  - id: review-a
    model: smart
    prompt: Review A of {{gather}}
  - id: review-b
    model: fast
    prompt: Review B of {{gather}}
  - id: dispatch
    parallel: [review-a, review-b]
  - id: consolidate
    prompt: "Merge: {{review-a}} || {{review-b}}"
`

func TestAdvanceSkipsParallelChildren(t *testing.T) {
	h, _ := newFlow(t, triLayout, "diff")
	text, isErr := h.advance(map[string]interface{}{"output": "THE-DIFF"})
	if isErr {
		t.Fatalf("advance error: %s", text)
	}
	if st := h.state(); st.CurrentStep != "dispatch" {
		t.Fatalf("after gather, current step = %q, want dispatch (children must be skipped)", st.CurrentStep)
	}
	// The dispatcher response must carry each child's interpolated prompt.
	for _, want := range []string{"review-a", "Review A of THE-DIFF", "review-b", "Review B of THE-DIFF", "outputs"} {
		if !strings.Contains(text, want) {
			t.Errorf("dispatch response missing %q:\n%s", want, text)
		}
	}
}

func TestAdvanceParallelStoresPerChildOutputs(t *testing.T) {
	h, _ := newFlow(t, triLayout, "diff")
	h.advance(map[string]interface{}{"output": "THE-DIFF"})
	text, isErr := h.advance(map[string]interface{}{
		"outputs": map[string]interface{}{"review-a": "A-VERDICT", "review-b": "B-VERDICT"},
	})
	if isErr {
		t.Fatalf("advance error: %s", text)
	}
	if st := h.state(); st.CurrentStep != "consolidate" {
		t.Fatalf("current step = %q, want consolidate", st.CurrentStep)
	}
	if !strings.Contains(text, "Merge: A-VERDICT || B-VERDICT") {
		t.Errorf("consolidate prompt did not receive child outputs:\n%s", text)
	}
}

func TestAdvanceParallelRequiresEveryChildOutput(t *testing.T) {
	h, _ := newFlow(t, triLayout, "diff")
	h.advance(map[string]interface{}{"output": "THE-DIFF"})

	text, isErr := h.advance(map[string]interface{}{"output": "one consolidated blob"})
	if !isErr || !strings.Contains(text, "review-a") {
		t.Fatalf("expected error naming missing child outputs, got isErr=%v:\n%s", isErr, text)
	}
	text, isErr = h.advance(map[string]interface{}{"outputs": map[string]interface{}{"review-a": "A"}})
	if !isErr || !strings.Contains(text, "review-b") {
		t.Fatalf("expected error naming review-b, got isErr=%v:\n%s", isErr, text)
	}
	if st := h.state(); st.CurrentStep != "dispatch" || st.Busy {
		t.Fatalf("rejected advance must leave session on dispatch and unclaimed, got step=%q busy=%v", st.CurrentStep, st.Busy)
	}
}

func TestStartSkipsLeadingParallelChildren(t *testing.T) {
	_, text := newFlow(t, `name: lead
description: children first
steps:
  - id: a
    prompt: A {{input}}
  - id: b
    prompt: B {{input}}
  - id: fan
    parallel: [a, b]
  - id: after
    prompt: After
`, "x")
	if !strings.Contains(text, "STEP 3/4: fan") {
		t.Fatalf("start should land on the dispatcher, got:\n%s", text)
	}
}

const gatedLoop = `name: gated
description: gate + until
steps:
  - id: fix
    prompt: Fix it, then say DONE.
    loop:
      max: 3
      until: DONE
      gate: "%s"
  - id: next
    prompt: Next step.
`

func TestLoopFailedGateIgnoresUntilSentinel(t *testing.T) {
	h, _ := newFlow(t, strings.Replace(gatedLoop, "%s", "echo GATE-SAYS-RED; exit 1", 1), "x")
	text, isErr := h.advance(map[string]interface{}{"output": "DONE"})
	if isErr {
		t.Fatalf("unexpected error: %s", text)
	}
	if st := h.state(); st.CurrentStep != "fix" {
		t.Fatalf("failing gate + DONE must stay in loop, now at %q", st.CurrentStep)
	}
	if !strings.Contains(text, "GATE-SAYS-RED") || !strings.Contains(text, "exit 1") {
		t.Errorf("retry response should show gate output and exit code:\n%s", text)
	}
}

func TestLoopPassingGateStillNeedsUntil(t *testing.T) {
	h, _ := newFlow(t, strings.Replace(gatedLoop, "%s", "true", 1), "x")
	h.advance(map[string]interface{}{"output": "made some progress"})
	if st := h.state(); st.CurrentStep != "fix" {
		t.Fatalf("gate pass without DONE must keep looping, now at %q", st.CurrentStep)
	}
	text, _ := h.advance(map[string]interface{}{"output": "DONE"})
	if st := h.state(); st.CurrentStep != "next" {
		t.Fatalf("gate pass + DONE should advance, now at %q:\n%s", st.CurrentStep, text)
	}
}

func TestLoopExhaustedWithFailingGateHalts(t *testing.T) {
	h, _ := newFlow(t, strings.Replace(strings.Replace(gatedLoop, "%s", "false", 1), "max: 3", "max: 1", 1), "x")
	text, isErr := h.advance(map[string]interface{}{"output": "DONE"})
	if !isErr || !strings.Contains(strings.ToLower(text), "exhausted") {
		t.Fatalf("expected halt error mentioning exhausted, got isErr=%v:\n%s", isErr, text)
	}
	if st := h.state(); st != nil {
		t.Fatalf("halted workflow should clear session state, got %+v", st)
	}
}

func TestLoopExhaustedWithoutUntilWarnsAndContinues(t *testing.T) {
	h, _ := newFlow(t, `name: soft
description: bounded rounds
steps:
  - id: rounds
    prompt: Research more; say COMPLETE when done.
    loop:
      max: 1
      until: COMPLETE
  - id: synth
    prompt: Synthesize.
`, "x")
	text, isErr := h.advance(map[string]interface{}{"output": "found two more sources"})
	if isErr {
		t.Fatalf("unexpected error: %s", text)
	}
	if st := h.state(); st.CurrentStep != "synth" {
		t.Fatalf("should continue to synth, at %q", st.CurrentStep)
	}
	if !strings.Contains(text, "LOOP EXHAUSTED") {
		t.Errorf("expected LOOP EXHAUSTED warning:\n%s", text)
	}
}

func TestAdvancePastLoopEvaluatesBranch(t *testing.T) {
	h, _ := newFlow(t, `name: lb
description: loop with branch
steps:
  - id: work
    prompt: Work; say DONE.
    loop:
      max: 3
      until: DONE
    branch:
      - when: SKIP
        goto: last
  - id: middle
    prompt: Middle.
  - id: last
    prompt: Last.
`, "x")
	h.advance(map[string]interface{}{"output": "SKIP\nDONE"})
	if st := h.state(); st.CurrentStep != "last" {
		t.Fatalf("branch on loop step ignored: at %q, want last", st.CurrentStep)
	}
}

const endLayout = `name: ends
description: explicit end
steps:
  - id: triage
    prompt: Classify.
    branch:
      - when: TINY
        goto: quick
  - id: full
    prompt: Full path.
    end: true
  - id: quick
    prompt: Quick path.
`

func TestEndStepCompletesWorkflow(t *testing.T) {
	h, _ := newFlow(t, endLayout, "x")
	h.advance(map[string]interface{}{"output": "BIG"})
	if st := h.state(); st.CurrentStep != "full" {
		t.Fatalf("at %q, want full", st.CurrentStep)
	}
	text, isErr := h.advance(map[string]interface{}{"output": "did it"})
	if isErr || !strings.Contains(text, "WORKFLOW COMPLETE") {
		t.Fatalf("end: true should complete the workflow, got isErr=%v:\n%s", isErr, text)
	}
}

func TestEndNotReachedOnBranchPath(t *testing.T) {
	h, _ := newFlow(t, endLayout, "x")
	h.advance(map[string]interface{}{"output": "TINY"})
	if st := h.state(); st.CurrentStep != "quick" {
		t.Fatalf("at %q, want quick", st.CurrentStep)
	}
}

func TestAdvanceParallelValidatesChildRequire(t *testing.T) {
	h, _ := newFlow(t, `name: req
description: child require
steps:
  - id: a
    prompt: A
    require:
      until: APPROVED
  - id: b
    prompt: B
  - id: fan
    parallel: [a, b]
  - id: after
    prompt: "{{a}}"
`, "x")
	text, isErr := h.advance(map[string]interface{}{
		"outputs": map[string]interface{}{"a": "", "b": "fine"},
	})
	if !isErr || !strings.Contains(text, "a") {
		t.Fatalf("child require must be enforced, got isErr=%v:\n%s", isErr, text)
	}
	if st := h.state(); st.CurrentStep != "fan" || st.Busy {
		t.Fatalf("rejected advance must stay on fan and release claim, got %q busy=%v", st.CurrentStep, st.Busy)
	}
	_, isErr = h.advance(map[string]interface{}{
		"outputs": map[string]interface{}{"a": "APPROVED", "b": "fine"},
	})
	if isErr {
		t.Fatal("valid child outputs rejected")
	}
}

func TestLoopUntilRequiresFreshOutput(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "green")
	h, _ := newFlow(t, strings.Replace(gatedLoop, "%s", "test -f "+marker, 1), "x")
	// Iteration 1: agent says DONE but gate is red → stay.
	h.advance(map[string]interface{}{"output": "DONE"})
	if st := h.state(); st.CurrentStep != "fix" {
		t.Fatalf("at %q, want fix", st.CurrentStep)
	}
	// Gate turns green; agent advances WITHOUT a new output. The stale
	// DONE from iteration 1 must not complete the loop.
	writeFile(t, marker, "ok")
	h.advance(map[string]interface{}{})
	if st := h.state(); st.CurrentStep != "fix" {
		t.Fatalf("stale DONE completed the loop: at %q", st.CurrentStep)
	}
}

func TestAdvanceWithoutOutputDoesNotReuseStaleBranchOutput(t *testing.T) {
	h, _ := newFlow(t, `name: stale
description: revisit
steps:
  - id: decide
    prompt: Decide; say AGAIN to repeat.
    branch:
      - when: AGAIN
        goto: decide
  - id: done
    prompt: Done.
`, "x")
	h.advance(map[string]interface{}{"output": "AGAIN"})
	if st := h.state(); st.CurrentStep != "decide" {
		t.Fatalf("at %q, want decide", st.CurrentStep)
	}
	// Second visit, no output: the first visit's AGAIN must not loop us back.
	h.advance(map[string]interface{}{})
	if st := h.state(); st.CurrentStep != "done" {
		t.Fatalf("stale AGAIN re-used: at %q, want done", st.CurrentStep)
	}
}

func TestParallelDispatchShowsChildRequire(t *testing.T) {
	_, text := newFlow(t, `name: req2
description: child require shown
steps:
  - id: fan
    parallel: [a]
  - id: a
    prompt: A
    require:
      until: APPROVED
`, "x")
	if !strings.Contains(text, "APPROVED") {
		t.Errorf("dispatch response should show child require contract:\n%s", text)
	}
}
