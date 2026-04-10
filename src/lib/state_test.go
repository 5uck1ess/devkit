package lib

import (
	"os"
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
