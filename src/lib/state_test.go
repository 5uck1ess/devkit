package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSessionID(t *testing.T) {
	id := NewSessionID()
	if len(id) != 12 {
		t.Errorf("session ID length = %d, want 12", len(id))
	}
	// Should be hex
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("session ID contains non-hex character: %c", c)
		}
	}
}

func TestNewSessionIDUnique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewSessionID()
		if ids[id] {
			t.Fatalf("duplicate session ID: %s", id)
		}
		ids[id] = true
	}
}

func TestEnsureSessionDir(t *testing.T) {
	root := t.TempDir()
	id := "test12345678"

	if err := EnsureSessionDir(root, id); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}

	dir := SessionDir(root, id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("session directory was not created")
	}
}

func TestWriteHandoff(t *testing.T) {
	root := t.TempDir()
	session := &Session{
		ID:            "hand12345678",
		Workflow:      "improve",
		Target:        "src/",
		Objective:     "fix all tests",
		Metric:        "go test ./...",
		MaxIterations: 10,
		BudgetUSD:     5.00,
	}

	steps := []Step{
		{Iteration: 1, Kept: true, MetricExitCode: 0, CostUSD: 0.05, ChangeSummary: "fixed auth"},
		{Iteration: 2, Kept: false, MetricExitCode: 1, CostUSD: 0.03, ChangeSummary: "broke tests"},
	}

	baseline := MetricResult{ExitCode: 1, Output: "3 tests failed"}

	if err := WriteHandoff(root, session, steps, baseline); err != nil {
		t.Fatalf("write handoff: %v", err)
	}

	path := HandoffPath(root, session.ID)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read handoff: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Iteration: 3 of 10") {
		t.Error("handoff should show next iteration as 3")
	}
	if !strings.Contains(content, "fix all tests") {
		t.Error("handoff should contain objective")
	}
	if !strings.Contains(content, "fixed auth") {
		t.Error("handoff should contain iteration history")
	}
	if !strings.Contains(content, "$4.92") {
		t.Errorf("handoff should show remaining budget, got:\n%s", content)
	}
}

func TestHandoffPath(t *testing.T) {
	path := HandoffPath("/repo", "abc123def456")
	expected := filepath.Join("/repo", ".devkit", "sessions", "abc123def456", "handoff.md")
	if path != expected {
		t.Errorf("path = %s, want %s", path, expected)
	}
}

func TestSessionJSON(t *testing.T) {
	dir := t.TempDir()
	state := &SessionState{
		ID:          "abc123",
		Workflow:    "research",
		CurrentStep: "clarify",
		StepType:    "prompt",
		Enforce:     "hard",
		Status:      "running",
		Outputs:     map[string]string{},
	}

	if err := WriteSessionJSON(dir, state); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ReadSessionJSON(dir)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.ID != "abc123" || got.CurrentStep != "clarify" {
		t.Errorf("got %+v", got)
	}

	if err := ClearSessionJSON(dir); err != nil {
		t.Fatalf("clear: %v", err)
	}
	got, err = ReadSessionJSON(dir)
	if err != nil {
		t.Fatalf("read after clear: %v", err)
	}
	if got != nil {
		t.Error("expected nil after clear")
	}
}
