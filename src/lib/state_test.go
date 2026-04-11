package lib

import (
	"os"
	"testing"
	"time"
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

// TestSessionJSONUpdatedAtBumps guards the staleness signal hooks rely
// on: every Write/Update call must move UpdatedAt forward. A regression
// here would wedge hooks into permanent "fresh session" mode and let
// orphaned sessions block tool calls forever.
func TestSessionJSONUpdatedAtBumps(t *testing.T) {
	dir := t.TempDir()
	state := &SessionState{
		ID:       "abc123",
		Workflow: "research",
		StepType: "prompt",
		Enforce:  "hard",
		Status:   "running",
		Outputs:  map[string]string{},
	}

	if err := WriteSessionJSON(dir, state); err != nil {
		t.Fatalf("write: %v", err)
	}
	first, err := ReadSessionJSON(dir)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if first.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt should be set after WriteSessionJSON")
	}

	// Sleep a hair past the timestamp resolution so the second bump is
	// observable even on systems with 1ms time granularity.
	time.Sleep(2 * time.Millisecond)

	if _, err := UpdateSessionJSON(dir, func(cur *SessionState) (*SessionState, error) {
		cur.CurrentStep = "next"
		return cur, nil
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	second, err := ReadSessionJSON(dir)
	if err != nil {
		t.Fatalf("read after update: %v", err)
	}
	if !second.UpdatedAt.After(first.UpdatedAt) {
		t.Errorf("UpdatedAt did not advance: first=%v second=%v", first.UpdatedAt, second.UpdatedAt)
	}
}
