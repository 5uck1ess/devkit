package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SessionState is the hot-path state file read by hooks on every tool call.
type SessionState struct {
	ID            string            `json:"id"`
	Workflow      string            `json:"workflow"`
	Input         string            `json:"input"`
	CurrentStep   string            `json:"current_step"`
	CurrentIndex  int               `json:"current_index"`
	TotalSteps    int               `json:"total_steps"`
	StepType      string            `json:"step_type"` // "prompt" | "command" | "parallel"
	Enforce       string            `json:"enforce"`
	Branch        bool              `json:"branch"`
	BudgetUSD     float64           `json:"budget_usd"`
	SpentUSD      float64           `json:"spent_usd"`
	StartedAt     time.Time         `json:"started_at"`
	Outputs       map[string]string `json:"outputs"`
	Status        string            `json:"status"`                   // "running" | "done" | "failed"
	LoopIteration int               `json:"loop_iteration,omitempty"` // current loop count for loop steps
	LoopMax       int               `json:"loop_max,omitempty"`       // max iterations for current loop
}

// SessionJSONPath returns the path to the hot-state session file.
func SessionJSONPath(dataDir string) string {
	return filepath.Join(dataDir, "session.json")
}

// WriteSessionJSON atomically writes session state to the hot-path JSON file.
func WriteSessionJSON(dataDir string, state *SessionState) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	path := SessionJSONPath(dataDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write session tmp: %w", err)
	}
	return os.Rename(tmp, path)
}

// ReadSessionJSON reads the hot-path session state. Returns nil if no session file exists.
func ReadSessionJSON(dataDir string) (*SessionState, error) {
	path := SessionJSONPath(dataDir)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}
	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	return &state, nil
}

// ClearSessionJSON removes the hot-path session file.
func ClearSessionJSON(dataDir string) error {
	path := SessionJSONPath(dataDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear session: %w", err)
	}
	return nil
}
