package lib

import (
	"os"
	"path/filepath"
	"strings"
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
		StepEnforce: EnforceHard,
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
		ID:          "abc123",
		Workflow:    "research",
		StepType:    "prompt",
		StepEnforce: EnforceHard,
		Status:      "running",
		Outputs:     map[string]string{},
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

// TestSessionJSONRejectsInvalidEnforce verifies that a stale or
// hand-edited session.json with a missing or bogus enforce value is
// rejected at ReadSessionJSON time by SessionState.UnmarshalJSON. This
// is the type-level replacement for guard.go's old effectiveEnforce
// empty-default: rather than silently coercing to "hard" we fail fast
// so the caller can see the corruption.
func TestSessionJSONRejectsInvalidEnforce(t *testing.T) {
	cases := []struct {
		name      string
		raw       string
		errSubstr string // expected substring; "" means any error is fine
	}{
		// The type-level gate — these all hit UnmarshalJSON's IsValid()
		// check and surface with "invalid enforce" in the error string.
		{"missing enforce field", `{"id":"x","status":"running"}`, "invalid enforce"},
		{"empty enforce", `{"id":"x","status":"running","enforce":""}`, "invalid enforce"},
		{"bogus enforce", `{"id":"x","status":"running","enforce":"medium"}`, "invalid enforce"},
		// Case variants — IsValid is strict, no folding.
		{"uppercase HARD", `{"id":"x","status":"running","enforce":"HARD"}`, "invalid enforce"},
		{"titlecase Hard", `{"id":"x","status":"running","enforce":"Hard"}`, "invalid enforce"},
		// Whitespace variants — IsValid does not trim.
		{"leading space", `{"id":"x","status":"running","enforce":" hard"}`, "invalid enforce"},
		{"trailing space", `{"id":"x","status":"running","enforce":"hard "}`, "invalid enforce"},
		// Non-string — encoding/json surfaces a type mismatch from the
		// underlying string alias before our UnmarshalJSON ever runs.
		// Still must reject, but the error substring differs.
		{"non-string enforce", `{"id":"x","status":"running","enforce":42}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "session.json")
			if err := os.WriteFile(path, []byte(tc.raw), 0o600); err != nil {
				t.Fatalf("write: %v", err)
			}
			_, err := ReadSessionJSON(dir)
			if err == nil {
				t.Fatalf("expected parse error, got nil")
			}
			if tc.errSubstr != "" && !strings.Contains(err.Error(), tc.errSubstr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.errSubstr)
			}
		})
	}
}

// TestEnforceModeIsValid pins the IsValid / IsValidOverride contract
// so a future "be lenient" refactor (case folding, trimming, adding a
// third mode) must be a deliberate change rather than an accident.
func TestEnforceModeIsValid(t *testing.T) {
	cases := []struct {
		mode          EnforceMode
		valid         bool
		validOverride bool
	}{
		{EnforceHard, true, true},
		{EnforceSoft, true, true},
		{EnforceInherit, false, true},
		{"HARD", false, false},
		{"Hard", false, false},
		{" hard", false, false},
		{"hard ", false, false},
		{"medium", false, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.mode), func(t *testing.T) {
			if got := tc.mode.IsValid(); got != tc.valid {
				t.Errorf("IsValid(%q) = %v, want %v", tc.mode, got, tc.valid)
			}
			if got := tc.mode.IsValidOverride(); got != tc.validOverride {
				t.Errorf("IsValidOverride(%q) = %v, want %v", tc.mode, got, tc.validOverride)
			}
		})
	}
}
