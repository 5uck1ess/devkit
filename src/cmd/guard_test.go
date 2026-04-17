package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/5uck1ess/devkit/lib"
)

// IMPORTANT: tests in this file MUST NOT call t.Parallel().
//
// newGuardTestEnv mutates six package-level globals (guardExit,
// guardStdin, guardStdout, guardStderr, guardToolName, guardStopMode)
// plus the CLAUDE_PLUGIN_DATA env var. Parallel subtests would race on
// all of these — data race, t.Setenv rejection, and cross-test buffer
// bleed-through. If you need parallel execution, refactor guard.go to
// pass a *guardContext struct through runPreToolGuard/runStopGuard
// instead of using package-level IO hooks.

// guardTestEnv wires the package-level IO + exit hooks to per-test
// buffers and a captured exit code, then runs the command. It returns
// the captured exit code, stdout, and stderr so table-driven tests can
// assert on all three.
type guardTestEnv struct {
	exit   int
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func newGuardTestEnv(t *testing.T, stdinPayload, toolName string, stopMode bool, dataDir string) *guardTestEnv {
	t.Helper()

	env := &guardTestEnv{exit: -1}

	// Capture exit. Real os.Exit would terminate the test binary.
	prevExit := guardExit
	guardExit = func(code int) {
		env.exit = code
	}
	t.Cleanup(func() { guardExit = prevExit })

	prevStdin := guardStdin
	prevStdout := guardStdout
	prevStderr := guardStderr
	guardStdin = strings.NewReader(stdinPayload)
	guardStdout = &env.stdout
	guardStderr = &env.stderr
	t.Cleanup(func() {
		guardStdin = prevStdin
		guardStdout = prevStdout
		guardStderr = prevStderr
	})

	// Reset flag state so previous tests don't leak into this one.
	prevToolName := guardToolName
	prevStopMode := guardStopMode
	guardToolName = toolName
	guardStopMode = stopMode
	t.Cleanup(func() {
		guardToolName = prevToolName
		guardStopMode = prevStopMode
	})

	// Isolate CLAUDE_PLUGIN_DATA so parallel tests don't collide.
	prevEnv, hadPrev := os.LookupEnv("CLAUDE_PLUGIN_DATA")
	if dataDir == "" {
		os.Unsetenv("CLAUDE_PLUGIN_DATA")
	} else {
		t.Setenv("CLAUDE_PLUGIN_DATA", dataDir)
	}
	t.Cleanup(func() {
		if hadPrev {
			os.Setenv("CLAUDE_PLUGIN_DATA", prevEnv)
		} else {
			os.Unsetenv("CLAUDE_PLUGIN_DATA")
		}
	})

	return env
}

// writeSession writes a session.json into tmpDir using the Go struct so
// the on-disk layout tracks lib.SessionState exactly. Tests that want
// malformed or partial JSON use writeSessionRaw instead.
func writeSession(t *testing.T, dir string, s lib.SessionState) {
	t.Helper()
	// Default UpdatedAt to now so tests aren't all accidentally stale.
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = time.Now()
	}
	data, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	writeSessionRaw(t, dir, data)
}

func writeSessionRaw(t *testing.T, dir string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "session.json"), data, 0o600); err != nil {
		t.Fatalf("write session.json: %v", err)
	}
}

func TestGuardPreToolUse(t *testing.T) {
	// Each row maps 1:1 to a case in hooks/hooks_test.sh so the fixture
	// matrix stays portable across the shell and Go harnesses.
	//
	// wantStderrSubstr is optional and pins the veto-message wording
	// on block rows. This is a deliberate substring match so future
	// message edits have to update the tests — the wording is the
	// only feedback the agent sees when a tool call is rejected, and
	// wording regressions have been historically easy to ship.
	tests := []struct {
		name             string
		dataDir          bool // set CLAUDE_PLUGIN_DATA
		hasSession       bool
		session          lib.SessionState
		stdin            string
		toolFlag         string
		wantExit         int
		wantStderrSubstr string
	}{
		{
			name:     "no CLAUDE_PLUGIN_DATA → allow",
			dataDir:  false,
			stdin:    `{"tool_name":"Bash"}`,
			wantExit: 0,
		},
		{
			name:     "no session file → allow",
			dataDir:  true,
			stdin:    `{"tool_name":"Bash"}`,
			wantExit: 0,
		},
		{
			name:       "status=done → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status:      "done",
				StepType:    "command",
				StepEnforce: lib.EnforceHard,
				CurrentStep: "build",
			},
			stdin:    `{"tool_name":"Bash"}`,
			wantExit: 0,
		},
		{
			name:       "command+hard+Bash → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status:      "running",
				StepType:    "command",
				StepEnforce: lib.EnforceHard,
				CurrentStep: "build",
				Workflow:    "feature",
				TotalSteps:  3,
			},
			stdin:            `{"tool_name":"Bash"}`,
			wantExit:         2,
			wantStderrSubstr: "BLOCKED: Command step",
		},
		{
			name:       "command+hard+Write → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:            `{"tool_name":"Write"}`,
			wantExit:         2,
			wantStderrSubstr: "attempted tool: Write",
		},
		{
			// Subagent dispatch is a prompt-step privilege. Command steps
			// are run by the engine directly, not by the model — dispatch
			// there would mean the model bypassing the engine, defeating
			// determinism. Pin this so a future refactor can't quietly
			// extend the prompt carve-out to command steps.
			name:       "command+hard+Agent → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:            `{"tool_name":"Agent"}`,
			wantExit:         2,
			wantStderrSubstr: "attempted tool: Agent",
		},
		{
			name:       "command+hard+Task → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:            `{"tool_name":"Task"}`,
			wantExit:         2,
			wantStderrSubstr: "attempted tool: Task",
		},
		{
			name:       "command+hard+devkit_advance → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:    `{"tool_name":"devkit_advance"}`,
			wantExit: 0,
		},
		{
			name:       "command+hard+mcp__devkit__advance → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:    `{"tool_name":"mcp__devkit__advance"}`,
			wantExit: 0,
		},
		{
			name:       "command+hard+TodoWrite → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:    `{"tool_name":"TodoWrite"}`,
			wantExit: 0,
		},
		{
			// Mid-workflow skill dispatch must work so a nested
			// "do tri-review" inside an active feature workflow
			// can load the trigger skill — engine still gates the
			// devkit_start it issues, so guard does not need to.
			name:       "command+hard+Skill → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:    `{"tool_name":"Skill"}`,
			wantExit: 0,
		},
		{
			name:       "command+soft → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceSoft, CurrentStep: "build",
			},
			stdin:    `{"tool_name":"Bash"}`,
			wantExit: 0,
		},
		{
			name:       "prompt+hard+Read → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "analyse",
			},
			stdin:    `{"tool_name":"Read"}`,
			wantExit: 0,
		},
		{
			name:       "prompt+hard+Grep → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "analyse",
			},
			stdin:    `{"tool_name":"Grep"}`,
			wantExit: 0,
		},
		{
			name:       "prompt+hard+Bash → block (issue #63 drift hole)",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "analyse",
				Workflow: "tri-review", TotalSteps: 6,
			},
			stdin:            `{"tool_name":"Bash"}`,
			wantExit:         2,
			wantStderrSubstr: "gather evidence with Read/Grep/Glob",
		},
		{
			name:       "prompt+hard+Write → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "analyse",
				Workflow: "tri-review", TotalSteps: 6,
			},
			stdin:            `{"tool_name":"Write"}`,
			wantExit:         2,
			wantStderrSubstr: "tri-review step",
		},
		{
			// Subagent dispatch (Task / Agent) is allowed on prompt+hard
			// so tri-* workflows can hand off to an external-model
			// reviewer without the main Claude model faking the verdict.
			// Write/Edit/Bash remain blocked on prompt+hard so the
			// main agent cannot cheat by writing its own output.
			name:       "prompt+hard+Task → allow (subagent dispatch)",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "review-smart",
			},
			stdin:    `{"tool_name":"Task"}`,
			wantExit: 0,
		},
		{
			name:       "prompt+hard+Agent → allow (subagent dispatch)",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "review-smart",
			},
			stdin:    `{"tool_name":"Agent"}`,
			wantExit: 0,
		},
		{
			name:       "prompt+hard+devkit_advance → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "analyse",
			},
			stdin:    `{"tool_name":"devkit_advance"}`,
			wantExit: 0,
		},
		{
			name:       "prompt+soft+Bash → allow with nudge",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceSoft, CurrentStep: "analyse",
			},
			stdin:    `{"tool_name":"Bash"}`,
			wantExit: 0,
		},
		{
			name:       "parallel+hard+Task → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "parallel", StepEnforce: lib.EnforceHard, CurrentStep: "fanout",
			},
			stdin:    `{"tool_name":"Task"}`,
			wantExit: 0,
		},
		{
			// Session files with a missing/empty enforce field are now
			// rejected at ReadSessionJSON time by SessionState.UnmarshalJSON
			// — the guard never gets a chance to fall through to a
			// silent default. Exits with a "cannot read session state"
			// error rather than the command-step "BLOCKED" message.
			name:       "session missing enforce field → parse-reject",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", CurrentStep: "build",
			},
			stdin:            `{"tool_name":"Bash"}`,
			wantExit:         2,
			wantStderrSubstr: "cannot read session state",
		},
		{
			// Empty step_type is non-command and non-prompt, so the
			// default branch (parallel/unknown) allows through. This
			// protects the agent from being wedged by a schema gap.
			name:       "missing step_type → allow (non-command fallthrough)",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepEnforce: lib.EnforceHard, CurrentStep: "analyse",
			},
			stdin:    `{"tool_name":"Bash"}`,
			wantExit: 0,
		},
		{
			// --tool-name flag short-circuits the stdin read entirely.
			name:       "command+hard+tool-name flag Bash → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:    "",
			toolFlag: "Bash",
			wantExit: 2,
		},
		{
			// Long MCP tool name produced by Claude Code's plugin
			// namespacing — mcp__plugin_<plugin>_<server>__<tool>.
			// Must still match the mcp__*devkit* allowlist glob.
			name:       "command+hard+long MCP devkit tool name → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:    `{"tool_name":"mcp__plugin_devkit_devkit-engine__devkit_advance"}`,
			wantExit: 0,
		},
		{
			name:       "prompt+hard+Glob → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "analyse",
			},
			stdin:    `{"tool_name":"Glob"}`,
			wantExit: 0,
		},
		{
			name:       "prompt+hard+Edit → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "analyse",
			},
			stdin:    `{"tool_name":"Edit"}`,
			wantExit: 2,
		},
		{
			name:       "prompt+hard+WebFetch → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "analyse",
			},
			stdin:    `{"tool_name":"WebFetch"}`,
			wantExit: 2,
		},
		{
			name:       "parallel+hard+Write → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "parallel", StepEnforce: lib.EnforceHard, CurrentStep: "fanout",
			},
			stdin:    `{"tool_name":"Write"}`,
			wantExit: 0,
		},
		{
			// parallel+soft should allow exactly the same as parallel+hard.
			// Explicit row so a future refactor that adds an enforce
			// check to the parallel branch cannot sneak through.
			name:       "parallel+soft+Bash → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "parallel", StepEnforce: lib.EnforceSoft, CurrentStep: "fanout",
			},
			stdin:    `{"tool_name":"Bash"}`,
			wantExit: 0,
		},
		{
			// Fixture parity gap: the deleted hooks/devkit-guard_test.sh
			// asserted prompt+hard+TodoWrite allow but the Go port only
			// tested TodoWrite under command+hard. Pin it explicitly.
			name:       "prompt+hard+TodoWrite → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "analyse",
			},
			stdin:    `{"tool_name":"TodoWrite"}`,
			wantExit: 0,
		},
		{
			// Mid-workflow skill dispatch under prompt+hard so a
			// nested keyword ("do deep research") loads its trigger
			// skill instead of being silently blocked.
			name:       "prompt+hard+Skill → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "analyse",
			},
			stdin:    `{"tool_name":"Skill"}`,
			wantExit: 0,
		},
		{
			// Fixture parity gap: soft enforcement is supposed to let
			// write-tier tools through, not just Bash. Pin the contract.
			name:       "prompt+soft+Write → allow with nudge",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceSoft, CurrentStep: "analyse",
				Workflow: "feature", TotalSteps: 4,
			},
			stdin:            `{"tool_name":"Write"}`,
			wantExit:         0,
			wantStderrSubstr: "call devkit_advance",
		},
		{
			// TotalSteps=0 exercises the no-index branch of stepLabel().
			// Engine always writes TotalSteps > 0 in practice, but this
			// pins the defensive fallback so removing it fails a test.
			name:       "prompt+hard+Bash with TotalSteps=0 → block (no-index label)",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "prompt", StepEnforce: lib.EnforceHard, CurrentStep: "unknown",
				Workflow: "mystery", TotalSteps: 0,
			},
			stdin:            `{"tool_name":"Bash"}`,
			wantExit:         2,
			wantStderrSubstr: "mystery (unknown)",
		},
		{
			// N2 negative: a third-party MCP server whose tool name
			// happens to contain "devkit" as a substring must NOT be
			// treated as a devkit MCP tool. The old shell glob
			// mcp__*devkit* would have allowed this, creating an
			// allowlist bypass. Pin the tightened Go behavior.
			name:       "command+hard+evil MCP tool containing 'devkit' → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:            `{"tool_name":"mcp__plugin_evil_server__devkit_masquerade"}`,
			wantExit:         2,
			wantStderrSubstr: "attempted tool: mcp__plugin_evil_server__devkit_masquerade",
		},
		{
			// mcp__devkit__<tool> short-form namespace (used in local
			// dev and non-plugin MCP hosts) must still be allowed.
			name:       "command+hard+mcp__devkit__advance short-form → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:    `{"tool_name":"mcp__devkit__advance"}`,
			wantExit: 0,
		},
		{
			// Anchoring on plugin+server (not just plugin): a
			// hypothetical future second MCP server under the devkit
			// plugin must NOT inherit command-step permissions
			// automatically. Adding such a server would require a
			// deliberate change to isDevkitMCPTool.
			name:       "command+hard+mcp__plugin_devkit_other_server__tool → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:            `{"tool_name":"mcp__plugin_devkit_other_server__probe"}`,
			wantExit:         2,
			wantStderrSubstr: "attempted tool: mcp__plugin_devkit_other_server__probe",
		},
		{
			// Status comparison is case-sensitive. A future engine
			// that writes "RUNNING" would silently disable enforcement.
			// Pin the current contract so any change is deliberate.
			name:       "Status=RUNNING (uppercase) treated as not-running → allow",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "RUNNING", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:    `{"tool_name":"Bash"}`,
			wantExit: 0,
		},
		{
			// --tool-name flag must win over stdin. Flag is used by
			// wrappers that pre-parse the payload; stdin is the
			// fallback. Pin the precedence so it can't silently flip.
			name:       "command+hard+flag Bash beats stdin devkit_advance → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:            `{"tool_name":"devkit_advance"}`,
			toolFlag:         "Bash",
			wantExit:         2,
			wantStderrSubstr: "attempted tool: Bash",
		},
		{
			// Empty stdin with no flag and a hard command step must
			// block — empty tool name is not in any allowlist. Pin
			// the fail-closed contract against schema drift.
			name:       "command+hard+empty stdin → block (unknown tool)",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:            ``,
			wantExit:         2,
			wantStderrSubstr: "attempted tool: <unknown>",
		},
		{
			// Malformed stdin JSON: readToolNameFromStdin returns an
			// error, the call site logs it, falls through with "",
			// and the hard command step blocks. Pin this so a future
			// refactor can't silently allow on parse failure.
			name:       "command+hard+malformed stdin → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
			},
			stdin:            `{not json`,
			wantExit:         2,
			wantStderrSubstr: "cannot determine tool name",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var dir string
			if tc.dataDir {
				dir = t.TempDir()
				if tc.hasSession {
					writeSession(t, dir, tc.session)
				}
			}
			env := newGuardTestEnv(t, tc.stdin, tc.toolFlag, false, dir)
			runGuard(guardCmd, nil)
			if env.exit != tc.wantExit {
				t.Fatalf("exit=%d want=%d\nstderr=%s", env.exit, tc.wantExit, env.stderr.String())
			}
			if tc.wantStderrSubstr != "" && !strings.Contains(env.stderr.String(), tc.wantStderrSubstr) {
				t.Fatalf("stderr missing substring %q\ngot: %s", tc.wantStderrSubstr, env.stderr.String())
			}
		})
	}
}

// TestGuardStaleTTLGarbageValues pins the fail-safe behaviour of
// staleTTL() when the DEVKIT_SESSION_STALE_TTL_SECONDS env var is
// set to garbage values. A silent fallback to default would be
// acceptable correctness-wise, but a silent fallback WITHOUT a stderr
// warning would hide operator misconfiguration, so we assert both the
// returned value AND the warning text.
func TestGuardStaleTTLGarbageValues(t *testing.T) {
	tests := []struct {
		name           string
		envValue       string
		wantWarn       bool
		wantWarnSubstr string
	}{
		{"non-numeric", "abc", true, "is not an integer"},
		{"negative", "-5", true, "must be positive"},
		{"zero", "0", true, "must be positive"},
		{"trailing space (handled)", "1800 ", false, ""}, // TrimSpace normalises this
		{"empty (falls through)", "", false, ""},         // early return, no warn
		{"overflow", "99999999999999999999", true, "is not an integer"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			prevStderr := guardStderr
			guardStderr = &stderr
			t.Cleanup(func() { guardStderr = prevStderr })
			t.Setenv("DEVKIT_SESSION_STALE_TTL_SECONDS", tc.envValue)

			got := staleTTL()
			// For "trailing space" the value parses to 1800s.
			// For everything else we expect the default.
			if tc.envValue == "1800 " {
				if got != 1800*time.Second {
					t.Fatalf("trimmed value should parse to 1800s, got %s", got)
				}
			} else if got != guardDefaultStaleTTL {
				t.Fatalf("garbage value should fall back to default %s, got %s",
					guardDefaultStaleTTL, got)
			}

			warned := stderr.Len() > 0
			if warned != tc.wantWarn {
				t.Fatalf("warn=%v want=%v (stderr=%q)", warned, tc.wantWarn, stderr.String())
			}
			if tc.wantWarnSubstr != "" && !strings.Contains(stderr.String(), tc.wantWarnSubstr) {
				t.Fatalf("stderr missing %q: %q", tc.wantWarnSubstr, stderr.String())
			}
		})
	}
}

// TestGuardStaleTTLHonourLongerOverride checks the "relax the clock"
// direction of the env override: a 2-hour TTL should treat a 45-minute-
// old session as fresh, where the default 30-minute TTL would mark it
// stale. Pins the semantic so a future refactor that hardcodes the
// default cannot silently kill the override.
func TestGuardStaleTTLHonourLongerOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DEVKIT_SESSION_STALE_TTL_SECONDS", "7200") // 2h
	writeSession(t, dir, lib.SessionState{
		Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
		Workflow: "feature", TotalSteps: 2,
		UpdatedAt: time.Now().Add(-45 * time.Minute),
	})
	env := newGuardTestEnv(t, `{"tool_name":"Bash"}`, "", false, dir)
	runGuard(guardCmd, nil)
	if env.exit != 2 {
		t.Fatalf("45-min-old session under 2h TTL should still be fresh and blocking: exit=%d stderr=%s",
			env.exit, env.stderr.String())
	}
}

// TestGuardStaleZeroTimestampWarn verifies the defensive warning for
// sessions with neither UpdatedAt nor StartedAt — a schema-drift
// scenario where an older engine wrote a session file without
// timestamps. The guard treats these as fresh (fail-safe towards
// enforcement), but must log a warning so the anomaly isn't silent.
func TestGuardStaleZeroTimestampWarn(t *testing.T) {
	dir := t.TempDir()
	// Zero time literal requires bypassing writeSession's default.
	data := []byte(`{"status":"running","step_type":"command","enforce":"hard","current_step":"build","workflow":"legacy","total_steps":1}`)
	writeSessionRaw(t, dir, data)
	env := newGuardTestEnv(t, `{"tool_name":"Bash"}`, "", false, dir)
	runGuard(guardCmd, nil)
	if env.exit != 2 {
		t.Fatalf("zero-timestamp session must still enforce: exit=%d", env.exit)
	}
	if !strings.Contains(env.stderr.String(), "no updated_at/started_at timestamps") {
		t.Fatalf("expected zero-timestamp WARNING, got: %s", env.stderr.String())
	}
}

// TestGuardUnreadableSession exercises the fail-closed path when the
// session file exists but cannot be read due to permissions. This is
// the concrete "broken install silently disarms enforcement" scenario
// from the review — the fail-closed contract must fire here, not the
// fail-open fallback the old code had. Unix-only because chmod 0 is
// meaningless on Windows.
func TestGuardUnreadableSession(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod 0 is a no-op on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file-mode permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	if err := os.WriteFile(path, []byte(`{"status":"running","step_type":"command","enforce":"hard"}`), 0o000); err != nil {
		t.Fatalf("write unreadable session: %v", err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o600) }) // let t.TempDir cleanup succeed

	env := newGuardTestEnv(t, `{"tool_name":"Bash"}`, "", false, dir)
	runGuard(guardCmd, nil)
	if env.exit != 2 {
		t.Fatalf("unreadable session must fail closed: exit=%d stderr=%s",
			env.exit, env.stderr.String())
	}
	if !strings.Contains(env.stderr.String(), "BLOCKED") {
		t.Fatalf("expected BLOCKED diagnostic, got: %s", env.stderr.String())
	}
}

// TestGuardStaleSessionPromptSoft covers the gap identified in code
// review: the stale-session warning must fire under prompt+soft just
// like under command+hard, because the staleness check precedes the
// step-type switch.
func TestGuardStaleSessionPromptSoft(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-2 * time.Hour)
	writeSession(t, dir, lib.SessionState{
		Status: "running", StepType: "prompt", StepEnforce: lib.EnforceSoft, CurrentStep: "analyse",
		Workflow: "tri-review", TotalSteps: 6,
		UpdatedAt: old, StartedAt: old,
	})
	env := newGuardTestEnv(t, `{"tool_name":"Write"}`, "", false, dir)
	runGuard(guardCmd, nil)
	if env.exit != 0 {
		t.Fatalf("stale prompt+soft must allow: exit=%d", env.exit)
	}
	if !strings.Contains(env.stderr.String(), "idle past TTL") {
		t.Fatalf("expected stale warning, got: %s", env.stderr.String())
	}
}

func TestGuardPreToolUseCorruptSession(t *testing.T) {
	dir := t.TempDir()
	writeSessionRaw(t, dir, []byte("{not valid json"))
	env := newGuardTestEnv(t, `{"tool_name":"Bash"}`, "", false, dir)
	runGuard(guardCmd, nil)
	if env.exit != 2 {
		t.Fatalf("corrupt session should fail closed: exit=%d stderr=%s", env.exit, env.stderr.String())
	}
	if !strings.Contains(env.stderr.String(), "BLOCKED") {
		t.Fatalf("expected BLOCKED diagnostic, got: %s", env.stderr.String())
	}
}

// TestGuardPreToolUseMissingEnforceField locks in the #81 fix end-to-end:
// a stale session.json missing the enforce field must fail closed through
// SessionState.UnmarshalJSON's rejection, not silently fall through to the
// pre-PR guard.go `effectiveEnforce` empty-default. Without this test, a
// refactor that swallowed the parse error between ReadSessionJSON and
// runPreToolGuard would silently disarm enforcement on stale sessions.
func TestGuardPreToolUseMissingEnforceField(t *testing.T) {
	dir := t.TempDir()
	writeSessionRaw(t, dir, []byte(`{"id":"x","status":"running","step_type":"command","workflow":"test","current_step":"build"}`))
	env := newGuardTestEnv(t, `{"tool_name":"Bash"}`, "", false, dir)
	runGuard(guardCmd, nil)
	if env.exit != 2 {
		t.Fatalf("missing enforce should fail closed: exit=%d stderr=%s", env.exit, env.stderr.String())
	}
	if !strings.Contains(env.stderr.String(), "BLOCKED") {
		t.Fatalf("expected BLOCKED diagnostic, got: %s", env.stderr.String())
	}
	if !strings.Contains(env.stderr.String(), "cannot read session state") {
		t.Fatalf("expected parse-reject path (cannot read session state), got: %s", env.stderr.String())
	}
}

func TestGuardPreToolUseStaleSession(t *testing.T) {
	dir := t.TempDir()
	// Stale session: UpdatedAt older than TTL. This is the orphan
	// recovery path — without it a crashed engine process would wedge
	// every subsequent tool call until the user manually cleaned up.
	old := time.Now().Add(-2 * time.Hour)
	writeSession(t, dir, lib.SessionState{
		Status:      "running",
		StepType:    "command",
		StepEnforce: lib.EnforceHard,
		CurrentStep: "build",
		Workflow:    "feature",
		UpdatedAt:   old,
		StartedAt:   old,
	})
	env := newGuardTestEnv(t, `{"tool_name":"Bash"}`, "", false, dir)
	runGuard(guardCmd, nil)
	if env.exit != 0 {
		t.Fatalf("stale session should allow: exit=%d", env.exit)
	}
	if !strings.Contains(env.stderr.String(), "idle past TTL") {
		t.Fatalf("expected TTL warning, got: %s", env.stderr.String())
	}
}

func TestGuardPreToolUseEnvTTLOverride(t *testing.T) {
	dir := t.TempDir()
	// Set a 1-second TTL via env so a 2-second-old session is stale.
	// Mirrors the DEVKIT_SESSION_STALE_TTL_SECONDS hook-era knob.
	t.Setenv("DEVKIT_SESSION_STALE_TTL_SECONDS", "1")
	writeSession(t, dir, lib.SessionState{
		Status: "running", StepType: "command", StepEnforce: lib.EnforceHard, CurrentStep: "build",
		UpdatedAt: time.Now().Add(-2 * time.Second),
	})
	env := newGuardTestEnv(t, `{"tool_name":"Bash"}`, "", false, dir)
	runGuard(guardCmd, nil)
	if env.exit != 0 {
		t.Fatalf("stale via env override should allow: exit=%d", env.exit)
	}
}

func TestGuardStopHook(t *testing.T) {
	tests := []struct {
		name         string
		dataDir      bool
		hasSession   bool
		session      lib.SessionState
		rawSession   []byte // overrides session if non-nil
		wantDecision string
		wantReason   string // substring match, "" skips
	}{
		{
			name:         "no CLAUDE_PLUGIN_DATA → approve",
			dataDir:      false,
			wantDecision: "approve",
		},
		{
			name:         "no session file → approve",
			dataDir:      true,
			wantDecision: "approve",
		},
		{
			name:       "running workflow → block",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "running", Workflow: "test", TotalSteps: 5, CurrentIndex: 2,
				StepEnforce: lib.EnforceHard,
			},
			wantDecision: "block",
			wantReason:   "3 steps remaining",
		},
		{
			name:       "done workflow → approve",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "done", Workflow: "test", TotalSteps: 5, CurrentIndex: 4,
				StepEnforce: lib.EnforceHard,
			},
			wantDecision: "approve",
		},
		{
			name:       "failed workflow → approve",
			dataDir:    true,
			hasSession: true,
			session: lib.SessionState{
				Status: "failed", Workflow: "test", TotalSteps: 5, CurrentIndex: 2,
				StepEnforce: lib.EnforceHard,
			},
			wantDecision: "approve",
		},
		{
			name:         "corrupt JSON → block (fail closed)",
			dataDir:      true,
			rawSession:   []byte("not json"),
			wantDecision: "block",
			wantReason:   "unreadable",
		},
		{
			// A stale session.json without an enforce field must flow
			// through SessionState.UnmarshalJSON's rejection and fail
			// closed in the stop hook too — not just the pre-tool guard.
			// Without this test, a refactor that swallowed the parse
			// error anywhere between ReadSessionJSON and runStopGuard
			// would silently disarm enforcement on stale sessions.
			name:         "missing enforce field → block (parse-reject through stop hook)",
			dataDir:      true,
			rawSession:   []byte(`{"id":"x","status":"running","workflow":"test"}`),
			wantDecision: "block",
			wantReason:   "unreadable",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var dir string
			if tc.dataDir {
				dir = t.TempDir()
				switch {
				case tc.rawSession != nil:
					writeSessionRaw(t, dir, tc.rawSession)
				case tc.hasSession:
					writeSession(t, dir, tc.session)
				}
			}
			env := newGuardTestEnv(t, "", "", true, dir)
			runGuard(guardCmd, nil)

			if env.exit != 0 {
				t.Fatalf("stop hook must always exit 0, got %d", env.exit)
			}

			var v stopVerdict
			if err := json.Unmarshal(env.stdout.Bytes(), &v); err != nil {
				t.Fatalf("stop verdict not valid JSON: %q (%v)", env.stdout.String(), err)
			}
			if v.Decision != tc.wantDecision {
				t.Fatalf("decision=%q want=%q", v.Decision, tc.wantDecision)
			}
			if tc.wantReason != "" && !strings.Contains(v.Reason, tc.wantReason) {
				t.Fatalf("reason=%q should contain %q", v.Reason, tc.wantReason)
			}
		})
	}
}

// TestGuardStopHookRepoScope pins the issue-#91 scope restriction:
// the stop-guard nag fires only when the current Claude Code session's
// repo matches the repo that started the workflow. Cross-repo sessions
// approve silently; the block remains active when the user returns to
// the originating repo. The legacy empty-RepoRoot case (pre-#91
// sessions) falls through to the historical block behavior — no silent
// bypass on missing scope signal.
func TestGuardStopHookRepoScope(t *testing.T) {
	tests := []struct {
		name             string
		stateRepoRoot    string
		claudeProjectDir string
		wantDecision     string
	}{
		{
			name:             "matching repo → block",
			stateRepoRoot:    "/tmp/repo-a",
			claudeProjectDir: "/tmp/repo-a",
			wantDecision:     "block",
		},
		{
			name:             "matching repo with trailing slash → block",
			stateRepoRoot:    "/tmp/repo-a",
			claudeProjectDir: "/tmp/repo-a/",
			wantDecision:     "block",
		},
		{
			name:             "different repo → approve",
			stateRepoRoot:    "/tmp/repo-a",
			claudeProjectDir: "/tmp/repo-b",
			wantDecision:     "approve",
		},
		{
			name:             "empty state.RepoRoot (pre-#91) → block",
			stateRepoRoot:    "",
			claudeProjectDir: "/tmp/repo-b",
			wantDecision:     "block",
		},
		{
			name:             "empty CLAUDE_PROJECT_DIR (cannot resolve current) → block",
			stateRepoRoot:    "/tmp/repo-a",
			claudeProjectDir: "",
			wantDecision:     "block",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeSession(t, dir, lib.SessionState{
				Status:      "running",
				Workflow:    "pr-ready",
				StepEnforce: lib.EnforceHard,
				TotalSteps:  5,
				RepoRoot:    tc.stateRepoRoot,
			})
			env := newGuardTestEnv(t, "", "", true, dir)
			// CLAUDE_PROJECT_DIR drives currentRepoRoot() for the
			// stop-guard's repo-match check. Unset forces the
			// pwd-walk fallback, which tempdirs won't satisfy → "".
			if tc.claudeProjectDir != "" {
				t.Setenv("CLAUDE_PROJECT_DIR", tc.claudeProjectDir)
			} else {
				t.Setenv("CLAUDE_PROJECT_DIR", "")
				// Also move pwd into a non-git tempdir so the
				// fallback walk returns "" deterministically.
				prev, _ := os.Getwd()
				nowhere := t.TempDir()
				if err := os.Chdir(nowhere); err != nil {
					t.Fatalf("chdir: %v", err)
				}
				t.Cleanup(func() { os.Chdir(prev) })
			}
			runGuard(guardCmd, nil)
			var v stopVerdict
			if err := json.Unmarshal(env.stdout.Bytes(), &v); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if v.Decision != tc.wantDecision {
				t.Fatalf("decision=%q want=%q (stderr=%q)", v.Decision, tc.wantDecision, env.stderr.String())
			}
		})
	}
}

// TestGuardStopHookRepoScopeWalkUp covers the pwd-walk fallback path in
// currentRepoRoot() — all cases in TestGuardStopHookRepoScope set
// CLAUDE_PROJECT_DIR, so the `.git`-finding loop never executes there.
// Also covers the subdirectory case (session cwd several levels below
// the repo root).
func TestGuardStopHookRepoScopeWalkUp(t *testing.T) {
	// Build a fake repo with a nested subdir: repo/a/b.
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	sub := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}

	tests := []struct {
		name          string
		stateRepoRoot string
		cwd           string
		wantDecision  string
	}{
		{
			name:          "walk-up from repo root resolves match → block",
			stateRepoRoot: repo,
			cwd:           repo,
			wantDecision:  "block",
		},
		{
			name:          "walk-up from nested subdir resolves match → block",
			stateRepoRoot: repo,
			cwd:           sub,
			wantDecision:  "block",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeSession(t, dir, lib.SessionState{
				Status:      "running",
				Workflow:    "pr-ready",
				StepEnforce: lib.EnforceHard,
				TotalSteps:  5,
				RepoRoot:    tc.stateRepoRoot,
			})
			env := newGuardTestEnv(t, "", "", true, dir)
			// Unset CLAUDE_PROJECT_DIR to force the pwd-walk
			// branch of currentRepoRoot().
			t.Setenv("CLAUDE_PROJECT_DIR", "")
			prev, _ := os.Getwd()
			if err := os.Chdir(tc.cwd); err != nil {
				t.Fatalf("chdir: %v", err)
			}
			t.Cleanup(func() { os.Chdir(prev) })
			runGuard(guardCmd, nil)
			var v stopVerdict
			if err := json.Unmarshal(env.stdout.Bytes(), &v); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if v.Decision != tc.wantDecision {
				t.Fatalf("decision=%q want=%q (stderr=%q)", v.Decision, tc.wantDecision, env.stderr.String())
			}
		})
	}
}

// TestGuardStopHookRepoScopeSymlink proves samePath() actually uses
// EvalSymlinks rather than bare Clean(): a symlinked path pointing at
// the originating repo must compare equal and keep the block active.
// Regression guard for macOS /var → /private/var and user-created
// symlinks to worktrees.
func TestGuardStopHookRepoScopeSymlink(t *testing.T) {
	real := t.TempDir()
	if err := os.Mkdir(filepath.Join(real, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	linkParent := t.TempDir()
	link := filepath.Join(linkParent, "linked-repo")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink not supported on this platform: %v", err)
	}

	dir := t.TempDir()
	writeSession(t, dir, lib.SessionState{
		Status:      "running",
		Workflow:    "pr-ready",
		StepEnforce: lib.EnforceHard,
		TotalSteps:  5,
		RepoRoot:    real,
	})
	env := newGuardTestEnv(t, "", "", true, dir)
	t.Setenv("CLAUDE_PROJECT_DIR", link)

	runGuard(guardCmd, nil)
	var v stopVerdict
	if err := json.Unmarshal(env.stdout.Bytes(), &v); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if v.Decision != "block" {
		t.Fatalf("decision=%q want=%q — samePath should resolve the symlink (stderr=%q)", v.Decision, "block", env.stderr.String())
	}
}

// TestGuardStopHookRepoScopeWorktree pins currentRepoRoot()'s support
// for git worktrees, where `.git` is a FILE (gitlink) instead of a
// directory. devkit itself runs from worktrees; a future refactor to
// IsDir() on the walk-up check would silently bypass the block for
// every worktree user without this test.
func TestGuardStopHookRepoScopeWorktree(t *testing.T) {
	worktree := t.TempDir()
	// Worktrees have .git as a regular file containing `gitdir: ...`.
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: /tmp/fake-main/.git/worktrees/wt\n"), 0o644); err != nil {
		t.Fatalf("write .git file: %v", err)
	}

	dir := t.TempDir()
	writeSession(t, dir, lib.SessionState{
		Status:      "running",
		Workflow:    "pr-ready",
		StepEnforce: lib.EnforceHard,
		TotalSteps:  5,
		RepoRoot:    worktree,
	})
	env := newGuardTestEnv(t, "", "", true, dir)
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	prev, _ := os.Getwd()
	if err := os.Chdir(worktree); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(prev) })

	runGuard(guardCmd, nil)
	var v stopVerdict
	if err := json.Unmarshal(env.stdout.Bytes(), &v); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if v.Decision != "block" {
		t.Fatalf("decision=%q want=%q — worktree .git as file must still resolve as repo root (stderr=%q)", v.Decision, "block", env.stderr.String())
	}
}

func TestGuardStopHookStaleSession(t *testing.T) {
	// Stale-during-Stop → approve so a crashed engine doesn't trap the
	// user in an un-stoppable session.
	dir := t.TempDir()
	old := time.Now().Add(-2 * time.Hour)
	writeSession(t, dir, lib.SessionState{
		Status:      "running",
		Workflow:    "test",
		StepEnforce: lib.EnforceHard,
		UpdatedAt:   old,
		StartedAt:   old,
	})
	env := newGuardTestEnv(t, "", "", true, dir)
	runGuard(guardCmd, nil)
	var v stopVerdict
	if err := json.Unmarshal(env.stdout.Bytes(), &v); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if v.Decision != "approve" {
		t.Fatalf("stale stop should approve: %+v", v)
	}
}
