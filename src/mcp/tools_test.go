package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/5uck1ess/devkit/lib"
	mcpmcp "github.com/mark3labs/mcp-go/mcp"
)

// newTestServer builds a minimal Server with the given dirs — no DB needed for tool tests.
func newTestServer(t *testing.T, dataDir, workflowDir string) *Server {
	t.Helper()
	return &Server{
		dataDir:     dataDir,
		workflowDir: workflowDir,
		repoRoot:    t.TempDir(),
		db:          nil,
		git:         nil,
		principles:  map[string][]string{},
	}
}

func callTool(t *testing.T, handler func(context.Context, mcpmcp.CallToolRequest) (*mcpmcp.CallToolResult, error)) string {
	t.Helper()
	result, err := handler(context.Background(), mcpmcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil || len(result.Content) == 0 {
		t.Fatal("handler returned empty result")
	}
	// Extract text from first content item.
	if tc, ok := result.Content[0].(mcpmcp.TextContent); ok {
		return tc.Text
	}
	t.Fatalf("unexpected content type: %T", result.Content[0])
	return ""
}

func TestList(t *testing.T) {
	dir := t.TempDir()

	// Write two valid workflow YAMLs and one non-YAML file.
	writeFile(t, filepath.Join(dir, "alpha.yml"), `name: alpha
description: Alpha workflow
steps:
  - id: step-one
    prompt: do something
`)
	writeFile(t, filepath.Join(dir, "beta.yaml"), `name: beta
description: Beta workflow
steps:
  - id: step-one
    prompt: do something
`)
	writeFile(t, filepath.Join(dir, "readme.txt"), "ignore me")

	srv := newTestServer(t, t.TempDir(), dir)
	_, handler := srv.listTool()
	out := callTool(t, handler)

	if !strings.Contains(out, "alpha") {
		t.Errorf("expected 'alpha' in output, got: %s", out)
	}
	if !strings.Contains(out, "Alpha workflow") {
		t.Errorf("expected description 'Alpha workflow' in output, got: %s", out)
	}
	if !strings.Contains(out, "beta") {
		t.Errorf("expected 'beta' in output, got: %s", out)
	}
	if strings.Contains(out, "readme") {
		t.Errorf("non-YAML file should not appear in output, got: %s", out)
	}
}

func TestListParseError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "broken.yml"), "{{{{ not valid yaml ~~~~")

	srv := newTestServer(t, t.TempDir(), dir)
	_, handler := srv.listTool()
	out := callTool(t, handler)

	if !strings.Contains(out, "broken") {
		t.Errorf("expected 'broken' in output, got: %s", out)
	}
	if !strings.Contains(out, "parse error") {
		t.Errorf("expected 'parse error' in output, got: %s", out)
	}
}

func TestListMissingDir(t *testing.T) {
	srv := newTestServer(t, t.TempDir(), "/nonexistent/path/workflows")
	_, handler := srv.listTool()
	result, err := handler(context.Background(), mcpmcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for missing workflow dir")
	}
}

func TestStatusNoSession(t *testing.T) {
	dataDir := t.TempDir()
	srv := newTestServer(t, dataDir, t.TempDir())
	_, handler := srv.statusTool()
	out := callTool(t, handler)

	if !strings.Contains(out, "No active workflow session") {
		t.Errorf("expected no-session message, got: %s", out)
	}
}

func TestStatusWithSession(t *testing.T) {
	dataDir := t.TempDir()

	state := &lib.SessionState{
		ID:           "sess-123",
		Workflow:     "my-workflow",
		CurrentStep:  "step-review",
		CurrentIndex: 2,
		TotalSteps:   5,
		Enforce:      "hard",
		Status:       "running",
		StartedAt:    time.Now(),
		Outputs:      map[string]string{},
	}
	if err := lib.WriteSessionJSON(dataDir, state); err != nil {
		t.Fatalf("write session: %v", err)
	}

	srv := newTestServer(t, dataDir, t.TempDir())
	_, handler := srv.statusTool()
	out := callTool(t, handler)

	checks := []struct {
		field string
		value string
	}{
		{"workflow name", "my-workflow"},
		{"session ID", "sess-123"},
		{"step name", "step-review"},
		{"step progress", "3/5"},
		{"enforce", "hard"},
		{"status", "running"},
	}
	for _, c := range checks {
		if !strings.Contains(out, c.value) {
			t.Errorf("expected %s (%q) in output, got:\n%s", c.field, c.value, out)
		}
	}
}

func TestStart(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "review.yml"), `name: review
description: Code review workflow
steps:
  - id: analyse
    prompt: Analyse {{input}} and identify issues.
  - id: report
    prompt: Write a report based on the analysis.
`)

	srv := newTestServer(t, dataDir, wfDir)
	_, handler := srv.startTool()

	req := mcpmcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"workflow": "review",
		"input":    "main.go",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil || len(result.Content) == 0 {
		t.Fatal("handler returned empty result")
	}
	if result.IsError {
		tc, _ := result.Content[0].(mcpmcp.TextContent)
		t.Fatalf("handler returned tool error: %s", tc.Text)
	}

	tc, ok := result.Content[0].(mcpmcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", result.Content[0])
	}
	out := tc.Text

	// Response should mention step 1
	if !strings.Contains(out, "STEP 1/2") {
		t.Errorf("expected 'STEP 1/2' in response, got:\n%s", out)
	}
	if !strings.Contains(out, "analyse") {
		t.Errorf("expected step id 'analyse' in response, got:\n%s", out)
	}
	if !strings.Contains(out, "main.go") {
		t.Errorf("expected interpolated input 'main.go' in response, got:\n%s", out)
	}
	if !strings.Contains(out, "devkit_advance") {
		t.Errorf("expected 'devkit_advance' call-to-action in response, got:\n%s", out)
	}

	// session.json should exist
	state, err := lib.ReadSessionJSON(dataDir)
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	if state == nil {
		t.Fatal("session.json was not created")
	}
	if state.Workflow != "review" {
		t.Errorf("expected workflow 'review', got %q", state.Workflow)
	}
	if state.CurrentStep != "analyse" {
		t.Errorf("expected current_step 'analyse', got %q", state.CurrentStep)
	}
	if state.CurrentIndex != 0 {
		t.Errorf("expected current_index 0, got %d", state.CurrentIndex)
	}
	if state.TotalSteps != 2 {
		t.Errorf("expected total_steps 2, got %d", state.TotalSteps)
	}
	if state.Status != "running" {
		t.Errorf("expected status 'running', got %q", state.Status)
	}
	if state.Input != "main.go" {
		t.Errorf("expected input 'main.go', got %q", state.Input)
	}
	if state.StepType != "prompt" {
		t.Errorf("expected step_type 'prompt', got %q", state.StepType)
	}
}

func TestStartAlreadyRunning(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "review.yml"), `name: review
description: Code review workflow
steps:
  - id: analyse
    prompt: Analyse {{input}} and identify issues.
`)

	// Pre-seed a running session
	existing := &lib.SessionState{
		ID:        "abc123",
		Workflow:  "review",
		Status:    "running",
		StartedAt: time.Now(),
		Outputs:   map[string]string{},
	}
	if err := lib.WriteSessionJSON(dataDir, existing); err != nil {
		t.Fatalf("write session: %v", err)
	}

	srv := newTestServer(t, dataDir, wfDir)
	_, handler := srv.startTool()

	req := mcpmcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"workflow": "review",
		"input":    "main.go",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when session already running")
	}
	tc, ok := result.Content[0].(mcpmcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", result.Content[0])
	}
	if !strings.Contains(tc.Text, "already running") {
		t.Errorf("expected 'already running' in error, got: %s", tc.Text)
	}
}

func TestAdvancePromptSteps(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "three-step.yml"), `name: three-step
description: Three prompt steps
steps:
  - id: plan
    prompt: Plan the work for {{input}}.
  - id: implement
    prompt: Implement the plan.
  - id: verify
    prompt: Verify everything works.
`)

	srv := newTestServer(t, dataDir, wfDir)

	// Start workflow to seed session.json
	_, startHandler := srv.startTool()
	startReq := mcpmcp.CallToolRequest{}
	startReq.Params.Arguments = map[string]interface{}{
		"workflow": "three-step",
		"input":    "widget feature",
	}
	startResult, err := startHandler(context.Background(), startReq)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if startResult.IsError {
		t.Fatalf("start returned error")
	}

	// Read session to get ID
	state, err := lib.ReadSessionJSON(dataDir)
	if err != nil || state == nil {
		t.Fatalf("read session after start: %v", err)
	}
	sessionID := state.ID

	_, advHandler := srv.advanceTool()

	// Advance 1: plan -> implement
	advReq := mcpmcp.CallToolRequest{}
	advReq.Params.Arguments = map[string]interface{}{
		"session": sessionID,
		"output":  "plan output here",
	}
	result, err := advHandler(context.Background(), advReq)
	if err != nil {
		t.Fatalf("advance 1: %v", err)
	}
	if result.IsError {
		tc, _ := result.Content[0].(mcpmcp.TextContent)
		t.Fatalf("advance 1 error: %s", tc.Text)
	}
	tc, _ := result.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc.Text, "STEP 2/3") {
		t.Errorf("advance 1: expected STEP 2/3, got:\n%s", tc.Text)
	}
	if !strings.Contains(tc.Text, "implement") {
		t.Errorf("advance 1: expected step id 'implement', got:\n%s", tc.Text)
	}

	// Verify output was captured
	state, _ = lib.ReadSessionJSON(dataDir)
	if state.Outputs["plan"] != "plan output here" {
		t.Errorf("expected plan output captured, got %q", state.Outputs["plan"])
	}

	// Advance 2: implement -> verify
	advReq2 := mcpmcp.CallToolRequest{}
	advReq2.Params.Arguments = map[string]interface{}{
		"session": sessionID,
		"output":  "implementation done",
	}
	result2, err := advHandler(context.Background(), advReq2)
	if err != nil {
		t.Fatalf("advance 2: %v", err)
	}
	if result2.IsError {
		tc2, _ := result2.Content[0].(mcpmcp.TextContent)
		t.Fatalf("advance 2 error: %s", tc2.Text)
	}
	tc2, _ := result2.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc2.Text, "STEP 3/3") {
		t.Errorf("advance 2: expected STEP 3/3, got:\n%s", tc2.Text)
	}

	// Advance 3: verify -> complete
	advReq3 := mcpmcp.CallToolRequest{}
	advReq3.Params.Arguments = map[string]interface{}{
		"session": sessionID,
		"output":  "all verified",
	}
	result3, err := advHandler(context.Background(), advReq3)
	if err != nil {
		t.Fatalf("advance 3: %v", err)
	}
	if result3.IsError {
		tc3, _ := result3.Content[0].(mcpmcp.TextContent)
		t.Fatalf("advance 3 error: %s", tc3.Text)
	}
	tc3, _ := result3.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc3.Text, "WORKFLOW COMPLETE") {
		t.Errorf("advance 3: expected WORKFLOW COMPLETE, got:\n%s", tc3.Text)
	}
	if !strings.Contains(tc3.Text, sessionID) {
		t.Errorf("advance 3: expected session ID in output, got:\n%s", tc3.Text)
	}

	// session.json should be cleared
	cleared, _ := lib.ReadSessionJSON(dataDir)
	if cleared != nil {
		t.Errorf("expected session.json cleared after completion, but state still exists")
	}
}

func TestAdvanceCommandStep(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "cmd-wf.yml"), `name: cmd-wf
description: Command workflow
steps:
  - id: greet
    command: echo hello
    expect: success
  - id: done
    prompt: Summarise.
`)

	srv := newTestServer(t, dataDir, wfDir)

	// Start
	_, startHandler := srv.startTool()
	startReq := mcpmcp.CallToolRequest{}
	startReq.Params.Arguments = map[string]interface{}{
		"workflow": "cmd-wf",
		"input":    "test",
	}
	startResult, err := startHandler(context.Background(), startReq)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if startResult.IsError {
		tc, _ := startResult.Content[0].(mcpmcp.TextContent)
		t.Fatalf("start error: %s", tc.Text)
	}

	state, _ := lib.ReadSessionJSON(dataDir)
	sessionID := state.ID

	// Advance: should execute "echo hello" and move to next step
	_, advHandler := srv.advanceTool()
	advReq := mcpmcp.CallToolRequest{}
	advReq.Params.Arguments = map[string]interface{}{
		"session": sessionID,
	}
	result, err := advHandler(context.Background(), advReq)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if result.IsError {
		tc, _ := result.Content[0].(mcpmcp.TextContent)
		t.Fatalf("advance error: %s", tc.Text)
	}
	tc, _ := result.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc.Text, "STEP 2/2") {
		t.Errorf("expected STEP 2/2, got:\n%s", tc.Text)
	}

	// Verify command output was captured
	state, _ = lib.ReadSessionJSON(dataDir)
	if !strings.Contains(state.Outputs["greet"], "hello") {
		t.Errorf("expected 'hello' in command output, got %q", state.Outputs["greet"])
	}
}

func TestAdvanceSessionMismatch(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "simple.yml"), `name: simple
description: Simple workflow
steps:
  - id: one
    prompt: Do something.
`)

	srv := newTestServer(t, dataDir, wfDir)

	// Start workflow
	_, startHandler := srv.startTool()
	startReq := mcpmcp.CallToolRequest{}
	startReq.Params.Arguments = map[string]interface{}{
		"workflow": "simple",
		"input":    "test",
	}
	startHandler(context.Background(), startReq)

	// Advance with wrong session ID
	_, advHandler := srv.advanceTool()
	advReq := mcpmcp.CallToolRequest{}
	advReq.Params.Arguments = map[string]interface{}{
		"session": "wrong-session-id",
	}
	result, err := advHandler(context.Background(), advReq)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for session mismatch")
	}
	tc, _ := result.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc.Text, "session mismatch") {
		t.Errorf("expected 'session mismatch' in error, got: %s", tc.Text)
	}
}

func TestLoopMaxIterations(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "loop-wf.yml"), `name: loop-wf
description: Loop workflow
steps:
  - id: iterate
    prompt: Do iteration work.
    loop:
      max: 3
  - id: finish
    prompt: Finish up.
`)

	srv := newTestServer(t, dataDir, wfDir)

	// Start workflow
	_, startHandler := srv.startTool()
	startReq := mcpmcp.CallToolRequest{}
	startReq.Params.Arguments = map[string]interface{}{
		"workflow": "loop-wf",
		"input":    "test input",
	}
	startResult, err := startHandler(context.Background(), startReq)
	if err != nil || startResult.IsError {
		t.Fatalf("start failed")
	}

	state, _ := lib.ReadSessionJSON(dataDir)
	sessionID := state.ID

	_, advHandler := srv.advanceTool()

	advReq := func() mcpmcp.CallToolRequest {
		r := mcpmcp.CallToolRequest{}
		r.Params.Arguments = map[string]interface{}{
			"session": sessionID,
			"output":  "did some work",
		}
		return r
	}

	// Advance 1: still on iterate (iteration 1/3)
	result, err := advHandler(context.Background(), advReq())
	if err != nil {
		t.Fatalf("advance 1: %v", err)
	}
	if result.IsError {
		tc, _ := result.Content[0].(mcpmcp.TextContent)
		t.Fatalf("advance 1 error: %s", tc.Text)
	}
	tc, _ := result.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc.Text, "LOOP ITERATION 1/3") {
		t.Errorf("advance 1: expected LOOP ITERATION 1/3, got:\n%s", tc.Text)
	}
	if !strings.Contains(tc.Text, "iterate") {
		t.Errorf("advance 1: expected step 'iterate' still active, got:\n%s", tc.Text)
	}

	// Advance 2: still on iterate (iteration 2/3)
	result2, err := advHandler(context.Background(), advReq())
	if err != nil {
		t.Fatalf("advance 2: %v", err)
	}
	tc2, _ := result2.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc2.Text, "LOOP ITERATION 2/3") {
		t.Errorf("advance 2: expected LOOP ITERATION 2/3, got:\n%s", tc2.Text)
	}

	// Advance 3: max reached — should advance to finish
	result3, err := advHandler(context.Background(), advReq())
	if err != nil {
		t.Fatalf("advance 3: %v", err)
	}
	if result3.IsError {
		tc3, _ := result3.Content[0].(mcpmcp.TextContent)
		t.Fatalf("advance 3 error: %s", tc3.Text)
	}
	tc3, _ := result3.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc3.Text, "finish") {
		t.Errorf("advance 3: expected step 'finish', got:\n%s", tc3.Text)
	}
	if strings.Contains(tc3.Text, "LOOP ITERATION") {
		t.Errorf("advance 3: should have left loop, got:\n%s", tc3.Text)
	}

	// Loop state should be reset
	state, _ = lib.ReadSessionJSON(dataDir)
	if state.LoopIteration != 0 {
		t.Errorf("expected LoopIteration reset to 0, got %d", state.LoopIteration)
	}
	if state.LoopMax != 0 {
		t.Errorf("expected LoopMax reset to 0, got %d", state.LoopMax)
	}
}

func TestLoopGatePass(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "gate-pass.yml"), `name: gate-pass
description: Gate pass workflow
steps:
  - id: check
    prompt: Do the check.
    loop:
      max: 5
      gate: "true"
  - id: next
    prompt: Next step.
`)

	srv := newTestServer(t, dataDir, wfDir)

	_, startHandler := srv.startTool()
	startReq := mcpmcp.CallToolRequest{}
	startReq.Params.Arguments = map[string]interface{}{
		"workflow": "gate-pass",
		"input":    "test",
	}
	startResult, err := startHandler(context.Background(), startReq)
	if err != nil || startResult.IsError {
		t.Fatalf("start failed")
	}

	state, _ := lib.ReadSessionJSON(dataDir)
	sessionID := state.ID

	_, advHandler := srv.advanceTool()
	advReq := mcpmcp.CallToolRequest{}
	advReq.Params.Arguments = map[string]interface{}{
		"session": sessionID,
		"output":  "check output",
	}

	// First advance: gate "true" exits 0 — should pass and advance to next
	result, err := advHandler(context.Background(), advReq)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if result.IsError {
		tc, _ := result.Content[0].(mcpmcp.TextContent)
		t.Fatalf("advance error: %s", tc.Text)
	}
	tc, _ := result.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc.Text, "next") {
		t.Errorf("expected 'next' step after gate pass, got:\n%s", tc.Text)
	}
	if strings.Contains(tc.Text, "LOOP ITERATION") {
		t.Errorf("should not see LOOP ITERATION header when gate passes, got:\n%s", tc.Text)
	}
}

func TestLoopGateFail(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "gate-fail.yml"), `name: gate-fail
description: Gate fail workflow
steps:
  - id: retry
    prompt: Try again.
    loop:
      max: 5
      gate: "false"
  - id: done
    prompt: Done.
`)

	srv := newTestServer(t, dataDir, wfDir)

	_, startHandler := srv.startTool()
	startReq := mcpmcp.CallToolRequest{}
	startReq.Params.Arguments = map[string]interface{}{
		"workflow": "gate-fail",
		"input":    "test",
	}
	startResult, err := startHandler(context.Background(), startReq)
	if err != nil || startResult.IsError {
		t.Fatalf("start failed")
	}

	state, _ := lib.ReadSessionJSON(dataDir)
	sessionID := state.ID

	_, advHandler := srv.advanceTool()
	advReq := mcpmcp.CallToolRequest{}
	advReq.Params.Arguments = map[string]interface{}{
		"session": sessionID,
		"output":  "attempt output",
	}

	// Advance: gate "false" exits 1 — should stay on loop step
	result, err := advHandler(context.Background(), advReq)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if result.IsError {
		tc, _ := result.Content[0].(mcpmcp.TextContent)
		t.Fatalf("advance error: %s", tc.Text)
	}
	tc, _ := result.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc.Text, "LOOP ITERATION 1/5") {
		t.Errorf("expected LOOP ITERATION 1/5 (still looping), got:\n%s", tc.Text)
	}
	if !strings.Contains(tc.Text, "retry") {
		t.Errorf("expected step 'retry' still active, got:\n%s", tc.Text)
	}
	if strings.Contains(tc.Text, "done") && !strings.Contains(tc.Text, "retry") {
		t.Errorf("should not have advanced to 'done', got:\n%s", tc.Text)
	}
}

// --- Security and edge-case tests ---

func TestStartPathTraversal(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()
	srv := newTestServer(t, dataDir, wfDir)
	_, handler := srv.startTool()

	cases := []struct {
		name string
		wf   string
	}{
		{"dot-dot", "../etc/passwd"},
		{"slash", "foo/bar"},
		{"backslash", `foo\bar`},
		{"dot-dot-no-slash", "..secret"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := mcpmcp.CallToolRequest{}
			req.Params.Arguments = map[string]interface{}{
				"workflow": tc.wf,
				"input":    "test",
			}
			result, err := handler(context.Background(), req)
			if err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if !result.IsError {
				t.Error("expected tool error for path traversal attempt")
			}
		})
	}
}

func TestStartWorkflowNotFound(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()
	srv := newTestServer(t, dataDir, wfDir)
	_, handler := srv.startTool()

	req := mcpmcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"workflow": "nonexistent",
		"input":    "test",
	}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected tool error for nonexistent workflow")
	}
}

func TestAdvanceNoSession(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()
	srv := newTestServer(t, dataDir, wfDir)
	_, handler := srv.advanceTool()

	req := mcpmcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"session": "nonexistent",
	}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected tool error for no active session")
	}
	tc, _ := result.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc.Text, "no active session") {
		t.Errorf("expected 'no active session' error, got: %s", tc.Text)
	}
}

func TestAdvanceExpectSuccessWithFailure(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "test.yml"), `name: test
steps:
  - id: check
    command: "exit 1"
    expect: success
  - id: done
    prompt: done
`)

	srv := newTestServer(t, dataDir, wfDir)

	// Start the workflow
	_, startHandler := srv.startTool()
	startReq := mcpmcp.CallToolRequest{}
	startReq.Params.Arguments = map[string]interface{}{
		"workflow": "test",
		"input":    "test",
	}
	startResult, _ := startHandler(context.Background(), startReq)
	if startResult.IsError {
		t.Fatalf("start failed: %v", startResult.Content)
	}

	// Read session to get ID
	state, _ := lib.ReadSessionJSON(dataDir)

	// Advance — command exits 1 but expect is "success", should error
	_, advHandler := srv.advanceTool()
	advReq := mcpmcp.CallToolRequest{}
	advReq.Params.Arguments = map[string]interface{}{
		"session": state.ID,
	}
	result, err := advHandler(context.Background(), advReq)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected tool error for expect:success with exit 1")
	}
	tc, _ := result.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc.Text, "expected success") {
		t.Errorf("expected 'expected success' in error, got: %s", tc.Text)
	}
}

func TestAdvanceExpectFailureWithSuccess(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "test.yml"), `name: test
steps:
  - id: check
    command: "exit 0"
    expect: failure
  - id: done
    prompt: done
`)

	srv := newTestServer(t, dataDir, wfDir)

	_, startHandler := srv.startTool()
	startReq := mcpmcp.CallToolRequest{}
	startReq.Params.Arguments = map[string]interface{}{
		"workflow": "test",
		"input":    "test",
	}
	startHandler(context.Background(), startReq)

	state, _ := lib.ReadSessionJSON(dataDir)

	_, advHandler := srv.advanceTool()
	advReq := mcpmcp.CallToolRequest{}
	advReq.Params.Arguments = map[string]interface{}{
		"session": state.ID,
	}
	result, err := advHandler(context.Background(), advReq)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected tool error for expect:failure with exit 0")
	}
	tc, _ := result.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc.Text, "expected failure") {
		t.Errorf("expected 'expected failure' in error, got: %s", tc.Text)
	}
}

func TestLoopUntilCondition(t *testing.T) {
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "test.yml"), `name: test
steps:
  - id: fix
    prompt: "Fix the issue"
    loop:
      max: 10
      until: "all clear"
  - id: done
    prompt: done
`)

	srv := newTestServer(t, dataDir, wfDir)

	// Start
	_, startHandler := srv.startTool()
	startReq := mcpmcp.CallToolRequest{}
	startReq.Params.Arguments = map[string]interface{}{
		"workflow": "test",
		"input":    "test",
	}
	startHandler(context.Background(), startReq)
	state, _ := lib.ReadSessionJSON(dataDir)

	_, advHandler := srv.advanceTool()

	// First advance — output does NOT contain "all clear", should stay in loop
	advReq := mcpmcp.CallToolRequest{}
	advReq.Params.Arguments = map[string]interface{}{
		"session": state.ID,
		"output":  "still broken",
	}
	result, _ := advHandler(context.Background(), advReq)
	tc, _ := result.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc.Text, "LOOP ITERATION") {
		t.Errorf("expected loop iteration, got:\n%s", tc.Text)
	}

	// Second advance — output has "all clear" as its own trimmed line,
	// which matches the line-anchored until sentinel.
	advReq2 := mcpmcp.CallToolRequest{}
	advReq2.Params.Arguments = map[string]interface{}{
		"session": state.ID,
		"output":  "fixed the issue\nall clear\n",
	}
	result2, _ := advHandler(context.Background(), advReq2)
	tc2, _ := result2.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc2.Text, "done") {
		t.Errorf("expected to advance past loop to 'done', got:\n%s", tc2.Text)
	}
}

func TestLoopUntilRejectsSubstring(t *testing.T) {
	// Regression: an until sentinel must NOT match when it appears only
	// inside another word. Prior behavior used strings.Contains which
	// made "fail" match "no failures found". Word-boundary match now.
	wfDir := t.TempDir()
	dataDir := t.TempDir()

	writeFile(t, filepath.Join(wfDir, "test.yml"), `name: test
steps:
  - id: fix
    prompt: "Fix the issue"
    loop:
      max: 3
      until: "FAIL"
  - id: end
    prompt: end
`)

	srv := newTestServer(t, dataDir, wfDir)

	_, startHandler := srv.startTool()
	startReq := mcpmcp.CallToolRequest{}
	startReq.Params.Arguments = map[string]interface{}{"workflow": "test", "input": "x"}
	startHandler(context.Background(), startReq)
	state, _ := lib.ReadSessionJSON(dataDir)

	_, advHandler := srv.advanceTool()
	advReq := mcpmcp.CallToolRequest{}
	advReq.Params.Arguments = map[string]interface{}{
		"session": state.ID,
		"output":  "no failures found; classification succeeded",
	}
	result, _ := advHandler(context.Background(), advReq)
	tc, _ := result.Content[0].(mcpmcp.TextContent)
	if !strings.Contains(tc.Text, "LOOP ITERATION") {
		t.Errorf("expected to remain in loop (FAIL must not match inside 'failures'), got:\n%s", tc.Text)
	}
}

// writeFile is a test helper that creates a file with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile %s: %v", path, err)
	}
}
