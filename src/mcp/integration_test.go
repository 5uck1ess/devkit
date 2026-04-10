package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/5uck1ess/devkit/lib"
	mcpmcp "github.com/mark3labs/mcp-go/mcp"
)

// setupTestServer creates a Server backed by temp dirs with a workflow and optional principles file.
func setupTestServer(t *testing.T, workflowYAML string, principlesYAML string) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	wfDir := filepath.Join(dir, "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "test.yml"), []byte(workflowYAML), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	if principlesYAML != "" {
		skillsDir := filepath.Join(dir, "skills")
		if err := os.MkdirAll(skillsDir, 0o755); err != nil {
			t.Fatalf("mkdir skills: %v", err)
		}
		if err := os.WriteFile(filepath.Join(skillsDir, "_principles.yml"), []byte(principlesYAML), 0o644); err != nil {
			t.Fatalf("write principles: %v", err)
		}
	}

	srv, err := NewServer(dir, dir, wfDir)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return srv, dir
}

// callToolHandler invokes a tool handler with the given arguments and returns the text content.
func callToolHandler(t *testing.T, handler func(context.Context, mcpmcp.CallToolRequest) (*mcpmcp.CallToolResult, error), args map[string]interface{}) (string, bool) {
	t.Helper()
	req := mcpmcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil || len(result.Content) == 0 {
		t.Fatal("handler returned empty result")
	}
	tc, ok := result.Content[0].(mcpmcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", result.Content[0])
	}
	return tc.Text, result.IsError
}

func TestIntegrationFullLifecycle(t *testing.T) {
	workflowYAML := `name: test
description: Full lifecycle test
steps:
  - id: plan
    prompt: "Plan the work for {{input}}."
  - id: check
    command: "echo hello"
    expect: success
  - id: report
    prompt: "Write a report."
`
	srv, dataDir := setupTestServer(t, workflowYAML, "")

	// Step 1: Start workflow
	_, startHandler := srv.startTool()
	out, isErr := callToolHandler(t, startHandler, map[string]interface{}{
		"workflow": "test",
		"input":   "test input",
	})
	if isErr {
		t.Fatalf("start returned error: %s", out)
	}
	if !strings.Contains(out, "STEP 1/3") {
		t.Errorf("expected STEP 1/3, got:\n%s", out)
	}
	if !strings.Contains(out, "plan") {
		t.Errorf("expected step id 'plan', got:\n%s", out)
	}

	// Verify session.json was created
	state, err := lib.ReadSessionJSON(dataDir)
	if err != nil || state == nil {
		t.Fatalf("session.json not created: %v", err)
	}
	sessionID := state.ID
	if state.Workflow != "test" {
		t.Errorf("expected workflow 'test', got %q", state.Workflow)
	}

	// Step 2: Advance past prompt step → should get command step
	_, advHandler := srv.advanceTool()
	out, isErr = callToolHandler(t, advHandler, map[string]interface{}{
		"session": sessionID,
		"output":  "plan output here",
	})
	if isErr {
		t.Fatalf("advance 1 returned error: %s", out)
	}
	if !strings.Contains(out, "STEP 2/3") {
		t.Errorf("expected STEP 2/3, got:\n%s", out)
	}
	if !strings.Contains(out, "check") {
		t.Errorf("expected step id 'check', got:\n%s", out)
	}

	// Step 3: Advance on command step → auto-executes "echo hello", moves to report
	out, isErr = callToolHandler(t, advHandler, map[string]interface{}{
		"session": sessionID,
	})
	if isErr {
		t.Fatalf("advance 2 returned error: %s", out)
	}
	if !strings.Contains(out, "STEP 3/3") {
		t.Errorf("expected STEP 3/3, got:\n%s", out)
	}
	if !strings.Contains(out, "report") {
		t.Errorf("expected step id 'report', got:\n%s", out)
	}

	// Verify command output was captured
	state, _ = lib.ReadSessionJSON(dataDir)
	if !strings.Contains(state.Outputs["check"], "hello") {
		t.Errorf("expected 'hello' in check output, got %q", state.Outputs["check"])
	}

	// Step 4: Advance past final prompt step → workflow complete
	out, isErr = callToolHandler(t, advHandler, map[string]interface{}{
		"session": sessionID,
		"output":  "report done",
	})
	if isErr {
		t.Fatalf("advance 3 returned error: %s", out)
	}
	if !strings.Contains(out, "WORKFLOW COMPLETE") {
		t.Errorf("expected WORKFLOW COMPLETE, got:\n%s", out)
	}
	if !strings.Contains(out, sessionID) {
		t.Errorf("expected session ID in completion message, got:\n%s", out)
	}

	// session.json should be cleared
	cleared, _ := lib.ReadSessionJSON(dataDir)
	if cleared != nil {
		t.Error("expected session.json cleared after completion")
	}
}

func TestIntegrationLoopWithGate(t *testing.T) {
	workflowYAML := `name: test
description: Loop gate test
steps:
  - id: fix
    prompt: "Fix the issue."
  - id: verify
    prompt: "Verify the fix."
    loop:
      max: 5
      gate: "true"
  - id: done
    prompt: "Wrap up."
`
	srv, dataDir := setupTestServer(t, workflowYAML, "")

	// Start workflow
	_, startHandler := srv.startTool()
	out, isErr := callToolHandler(t, startHandler, map[string]interface{}{
		"workflow": "test",
		"input":   "bug fix",
	})
	if isErr {
		t.Fatalf("start returned error: %s", out)
	}

	state, _ := lib.ReadSessionJSON(dataDir)
	sessionID := state.ID

	// Advance past "fix" step → should land on "verify" (loop step)
	_, advHandler := srv.advanceTool()
	out, isErr = callToolHandler(t, advHandler, map[string]interface{}{
		"session": sessionID,
		"output":  "fixed the bug",
	})
	if isErr {
		t.Fatalf("advance to verify returned error: %s", out)
	}
	if !strings.Contains(out, "verify") {
		t.Errorf("expected to land on 'verify', got:\n%s", out)
	}

	// Advance on loop step → gate "true" exits 0, should jump to "done"
	out, isErr = callToolHandler(t, advHandler, map[string]interface{}{
		"session": sessionID,
		"output":  "verification output",
	})
	if isErr {
		t.Fatalf("advance through gate returned error: %s", out)
	}
	if !strings.Contains(out, "done") {
		t.Errorf("expected to advance to 'done' after gate pass, got:\n%s", out)
	}
	if strings.Contains(out, "LOOP ITERATION") {
		t.Errorf("should not see LOOP ITERATION when gate passes, got:\n%s", out)
	}
}

func TestIntegrationPrincipleInjection(t *testing.T) {
	workflowYAML := `name: test
description: Principle injection test
principles: [dry, yagni]
steps:
  - id: step1
    prompt: "do something"
`
	principlesYAML := `dry:
  - Don't abstract until 3rd duplication
yagni:
  - Build what's needed now
`
	srv, _ := setupTestServer(t, workflowYAML, principlesYAML)

	// Start workflow
	_, startHandler := srv.startTool()
	out, isErr := callToolHandler(t, startHandler, map[string]interface{}{
		"workflow": "test",
		"input":   "test input",
	})
	if isErr {
		t.Fatalf("start returned error: %s", out)
	}

	if !strings.Contains(out, "PRINCIPLES:") {
		t.Errorf("expected PRINCIPLES: header, got:\n%s", out)
	}
	if !strings.Contains(out, "dry") {
		t.Errorf("expected 'dry' principle, got:\n%s", out)
	}
	if !strings.Contains(out, "Don't abstract until 3rd duplication") {
		t.Errorf("expected dry rule text, got:\n%s", out)
	}
	if !strings.Contains(out, "yagni") {
		t.Errorf("expected 'yagni' principle, got:\n%s", out)
	}
	if !strings.Contains(out, "Build what's needed now") {
		t.Errorf("expected yagni rule text, got:\n%s", out)
	}
}

func TestIntegrationExpectFailure(t *testing.T) {
	workflowYAML := `name: test
description: Expect failure test
steps:
  - id: should-fail
    command: "exit 1"
    expect: failure
  - id: done
    prompt: "Summarise."
`
	srv, dataDir := setupTestServer(t, workflowYAML, "")

	// Start workflow → lands on command step
	_, startHandler := srv.startTool()
	out, isErr := callToolHandler(t, startHandler, map[string]interface{}{
		"workflow": "test",
		"input":   "test input",
	})
	if isErr {
		t.Fatalf("start returned error: %s", out)
	}

	state, _ := lib.ReadSessionJSON(dataDir)
	sessionID := state.ID

	// Advance on command step: "exit 1" with expect: failure → should succeed
	_, advHandler := srv.advanceTool()
	out, isErr = callToolHandler(t, advHandler, map[string]interface{}{
		"session": sessionID,
	})
	if isErr {
		t.Fatalf("advance returned error (expected success for expected failure): %s", out)
	}
	if !strings.Contains(out, "STEP 2/2") {
		t.Errorf("expected STEP 2/2 after expected failure passed, got:\n%s", out)
	}
	if !strings.Contains(out, "done") {
		t.Errorf("expected step id 'done', got:\n%s", out)
	}
}
