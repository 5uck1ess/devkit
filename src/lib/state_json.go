package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EnforceMode is a typed enum for workflow enforcement mode. Bare string
// was previously checked only in engine.validate(), meaning a stray
// assignment from any package writing to SessionState could silently
// fall through to the "soft" branch of guard.go's switch. The named
// type concentrates the valid-value set in IsValid() and lets writers
// at every layer signal intent at the type level. The YAML and JSON
// wire format is unchanged — gopkg.in/yaml.v3 and encoding/json both
// handle string-aliased types transparently. Defined here (not in
// engine) because engine already imports lib; inverting that would
// create a cycle. engine re-exports this type as a type alias for
// ergonomic call sites.
type EnforceMode string

const (
	EnforceInherit EnforceMode = ""     // step-level only — inherits workflow
	EnforceHard    EnforceMode = "hard" // default — guard blocks tools mid-step
	EnforceSoft    EnforceMode = "soft" // allow + nudge; Stop-hook still blocks
)

// IsValid reports whether m is a concrete enforcement mode. The empty
// value is valid on a step override (inherit) but not on a resolved
// SessionState.StepEnforce — callers in that context should reject "".
func (m EnforceMode) IsValid() bool {
	return m == EnforceHard || m == EnforceSoft
}

// SessionState is the hot-path state file read by hooks on every tool call.
type SessionState struct {
	ID           string      `json:"id"`
	Workflow     string      `json:"workflow"`
	Input        string      `json:"input"`
	CurrentStep  string      `json:"current_step"`
	CurrentIndex int         `json:"current_index"`
	TotalSteps   int         `json:"total_steps"`
	StepType     string      `json:"step_type"` // "prompt" | "command" | "parallel"
	StepEnforce  EnforceMode `json:"enforce"`
	Branch       bool        `json:"branch"`
	BudgetUSD    float64     `json:"budget_usd"`
	SpentUSD     float64     `json:"spent_usd"`
	StartedAt    time.Time   `json:"started_at"`
	// UpdatedAt is bumped on every WriteSessionJSON. Hooks read this to
	// detect orphaned sessions (engine crash leaves Status=running but
	// no process is advancing) and refuse to enforce against them.
	UpdatedAt time.Time         `json:"updated_at"`
	Outputs   map[string]string `json:"outputs"`
	Status    string            `json:"status"` // "running" | "done" | "failed"
	// Busy is a claim flag held for the duration of an in-flight step
	// advance. Written under the session lock; a concurrent claimant
	// seeing it set must abort instead of racing the current writer.
	Busy          bool `json:"busy,omitempty"`
	LoopIteration int  `json:"loop_iteration,omitempty"` // current loop count for loop steps
	LoopMax       int  `json:"loop_max,omitempty"`       // max iterations for current loop
}

// UnmarshalJSON validates StepEnforce at read time so a stale or
// hand-edited session.json with an invalid/missing enforce value can
// never reach guard.go's switch. Every current writer goes through
// engine.EffectiveEnforce which returns a concrete mode, so this only
// triggers on corrupt or pre-#80 session files — in which case we'd
// rather fail loudly than silently fall through to "soft". Uses a
// type alias to avoid infinite recursion.
func (s *SessionState) UnmarshalJSON(data []byte) error {
	type alias SessionState
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if !a.StepEnforce.IsValid() {
		return fmt.Errorf("session state has invalid enforce %q — must be \"hard\" or \"soft\"", a.StepEnforce)
	}
	*s = SessionState(a)
	return nil
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
// file for the duration of fn. Cross-process (flock); the lock file
// persists and is released on fd close.
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
	// Acquire platform-specific exclusive advisory lock. Implementations
	// live in state_lock_unix.go / state_lock_windows.go behind build
	// tags so the package compiles on all GOOS targets the release
	// Makefile ships (linux, darwin, windows).
	if err := lockFile(f); err != nil {
		return fmt.Errorf("acquire session lock: %w", err)
	}
	defer unlockFile(f)
	return fn()
}

// WriteSessionJSON atomically writes session state under the session lock.
// Side effect: mutates state.UpdatedAt to time.Now() before serialising.
// Callers holding the struct for later comparison or retry must account
// for this — take a copy first if the original timestamp matters.
func WriteSessionJSON(dataDir string, state *SessionState) error {
	return withSessionLock(dataDir, func() error {
		return writeSessionJSONLocked(dataDir, state)
	})
}

func writeSessionJSONLocked(dataDir string, state *SessionState) error {
	// Bump on every write so hooks can spot orphaned sessions via
	// staleness — see SessionState.UpdatedAt.
	state.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	path := SessionJSONPath(dataDir)
	// CreateTemp uses O_EXCL so the tmp name is collision-free even if
	// a caller ever writes without holding the session lock.
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

// UpdateSessionJSON runs fn under the session lock with the current
// state. fn returning a non-nil state saves it; fn returning nil means
// "no change" and the on-disk state is left untouched. This is a true
// read-modify-write primitive — use it for any state mutation that
// must be atomic against concurrent readers or writers.
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
