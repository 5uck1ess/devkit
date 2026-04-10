package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// SessionState is the hot-path state file read by hooks on every tool call.
type SessionState struct {
	ID           string            `json:"id"`
	Workflow     string            `json:"workflow"`
	Input        string            `json:"input"`
	CurrentStep  string            `json:"current_step"`
	CurrentIndex int               `json:"current_index"`
	TotalSteps   int               `json:"total_steps"`
	StepType     string            `json:"step_type"` // "prompt" | "command" | "parallel"
	Enforce      string            `json:"enforce"`
	Branch       bool              `json:"branch"`
	BudgetUSD    float64           `json:"budget_usd"`
	SpentUSD     float64           `json:"spent_usd"`
	StartedAt    time.Time         `json:"started_at"`
	Outputs      map[string]string `json:"outputs"`
	Status       string            `json:"status"` // "running" | "done" | "failed"
	// Busy is a claim flag set by devkit_advance while it is executing.
	// A second concurrent devkit_advance seeing Busy=true rejects with a
	// "step already in progress" error rather than racing the first
	// writer. Written under the cross-process session.json.lock.
	Busy          bool `json:"busy,omitempty"`
	LoopIteration int  `json:"loop_iteration,omitempty"` // current loop count for loop steps
	LoopMax       int  `json:"loop_max,omitempty"`       // max iterations for current loop
}

// SessionJSONPath returns the path to the hot-state session file.
func SessionJSONPath(dataDir string) string {
	return filepath.Join(dataDir, "session.json")
}

// sessionLockPath is the sibling lock file for cross-process serialization.
func sessionLockPath(dataDir string) string {
	return filepath.Join(dataDir, "session.json.lock")
}

// withSessionLock acquires an exclusive advisory lock on a sibling .lock
// file for the duration of fn. The lock is cross-process (flock) so two
// MCP server instances or a racing hook cannot observe torn
// read-modify-write sequences on session.json. The lock file itself is
// created on first use and never removed; the lock is released by
// closing the descriptor.
func withSessionLock(dataDir string, fn func() error) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	lockPath := sessionLockPath(dataDir)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open session lock: %w", err)
	}
	defer f.Close()
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		return fmt.Errorf("acquire session lock: %w", err)
	}
	defer unix.Flock(int(f.Fd()), unix.LOCK_UN)
	return fn()
}

// WriteSessionJSON atomically writes session state to the hot-path JSON file.
// Serialized across processes via a sibling .lock file to prevent two
// concurrent devkit_advance calls from clobbering each other's updates.
// Uses a per-writer temp file name so temp collisions between racing
// writers are impossible.
func WriteSessionJSON(dataDir string, state *SessionState) error {
	return withSessionLock(dataDir, func() error {
		return writeSessionJSONLocked(dataDir, state)
	})
}

func writeSessionJSONLocked(dataDir string, state *SessionState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	path := SessionJSONPath(dataDir)
	// Per-writer temp name so two concurrent writers within the same
	// process (or across processes between flock grants) never target
	// the same temp path. CreateTemp uses O_EXCL so it is collision-free.
	tmp, err := os.CreateTemp(dataDir, "session.json.tmp-*")
	if err != nil {
		return fmt.Errorf("create session tmp: %w", err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we bail before rename.
	committed := false
	defer func() {
		if !committed {
			os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write session tmp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod session tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close session tmp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename session: %w", err)
	}
	committed = true
	return nil
}

// ReadSessionJSON reads the hot-path session state. Returns nil if no session file exists.
// Takes a shared lock so a concurrent writer cannot be observed mid-rename.
func ReadSessionJSON(dataDir string) (*SessionState, error) {
	var state *SessionState
	err := withSessionLock(dataDir, func() error {
		path := SessionJSONPath(dataDir)
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read session: %w", err)
		}
		var s SessionState
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("parse session: %w", err)
		}
		state = &s
		return nil
	})
	return state, err
}

// ClearSessionJSON removes the hot-path session file.
func ClearSessionJSON(dataDir string) error {
	return withSessionLock(dataDir, func() error {
		path := SessionJSONPath(dataDir)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("clear session: %w", err)
		}
		return nil
	})
}

// UpdateSessionJSON runs fn under an exclusive lock with the current
// state, then writes whatever fn returns (unless fn returns nil, which
// means "no change"). This gives callers a true read-modify-write
// primitive so multiple devkit_advance calls never race on the same
// session index.
func UpdateSessionJSON(dataDir string, fn func(*SessionState) (*SessionState, error)) (*SessionState, error) {
	var result *SessionState
	err := withSessionLock(dataDir, func() error {
		path := SessionJSONPath(dataDir)
		data, err := os.ReadFile(path)
		var cur *SessionState
		if err == nil {
			var s SessionState
			if jerr := json.Unmarshal(data, &s); jerr != nil {
				return fmt.Errorf("parse session: %w", jerr)
			}
			cur = &s
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("read session: %w", err)
		}
		next, fnErr := fn(cur)
		if fnErr != nil {
			return fnErr
		}
		if next == nil {
			result = cur
			return nil
		}
		if werr := writeSessionJSONLocked(dataDir, next); werr != nil {
			return werr
		}
		result = next
		return nil
	})
	return result, err
}
