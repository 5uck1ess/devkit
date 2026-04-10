package mcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/5uck1ess/devkit/engine"
	"github.com/5uck1ess/devkit/lib"
	mcpmcp "github.com/mark3labs/mcp-go/mcp"
	mcpgo "github.com/mark3labs/mcp-go/server"
)

// commandTimeout is the maximum duration for workflow command execution.
const commandTimeout = 5 * time.Minute

// gateTimeout is shorter than commandTimeout because loop gates should
// be fast checks (lint, test, build) — a gate that takes longer than
// this is almost certainly wedged and should fail the loop fast rather
// than eat the full command budget.
const gateTimeout = 60 * time.Second

func (s *Server) listTool() (mcpmcp.Tool, mcpgo.ToolHandlerFunc) {
	tool := mcpmcp.NewTool("devkit_list",
		mcpmcp.WithDescription("List available workflows"),
	)
	return tool, func(ctx context.Context, req mcpmcp.CallToolRequest) (*mcpmcp.CallToolResult, error) {
		entries, err := os.ReadDir(s.workflowDir)
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("no workflows directory: %v", err)), nil
		}
		var lines []string
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
				continue
			}
			wfName := strings.TrimSuffix(strings.TrimSuffix(name, ".yml"), ".yaml")
			path := filepath.Join(s.workflowDir, name)
			wf, err := engine.ParseFile(path)
			if err != nil {
				lines = append(lines, fmt.Sprintf("- %s (parse error)", wfName))
				continue
			}
			lines = append(lines, fmt.Sprintf("- **%s**: %s", wfName, wf.Description))
		}
		return mcpmcp.NewToolResultText(strings.Join(lines, "\n")), nil
	}
}

func (s *Server) statusTool() (mcpmcp.Tool, mcpgo.ToolHandlerFunc) {
	tool := mcpmcp.NewTool("devkit_status",
		mcpmcp.WithDescription("Check workflow progress"),
		mcpmcp.WithString("session", mcpmcp.Description("Session ID (optional)")),
	)
	return tool, func(ctx context.Context, req mcpmcp.CallToolRequest) (*mcpmcp.CallToolResult, error) {
		state, err := lib.ReadSessionJSON(s.dataDir)
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("read state: %v", err)), nil
		}
		if state == nil {
			return mcpmcp.NewToolResultText("No active workflow session."), nil
		}
		msg := fmt.Sprintf("Workflow: %s\nSession: %s\nStep: %s (%d/%d)\nEnforce: %s\nStatus: %s",
			state.Workflow, state.ID, state.CurrentStep,
			state.CurrentIndex+1, state.TotalSteps,
			state.Enforce, state.Status)
		return mcpmcp.NewToolResultText(msg), nil
	}
}

func (s *Server) startTool() (mcpmcp.Tool, mcpgo.ToolHandlerFunc) {
	tool := mcpmcp.NewTool("devkit_start",
		mcpmcp.WithDescription("Start a workflow"),
		mcpmcp.WithString("workflow", mcpmcp.Required(), mcpmcp.Description("Workflow name")),
		mcpmcp.WithString("input", mcpmcp.Required(), mcpmcp.Description("Workflow input/description")),
	)
	return tool, func(ctx context.Context, req mcpmcp.CallToolRequest) (*mcpmcp.CallToolResult, error) {
		// Check no active session — propagate read errors
		existing, err := lib.ReadSessionJSON(s.dataDir)
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("read session state: %v", err)), nil
		}
		if existing != nil && existing.Status == "running" {
			return mcpmcp.NewToolResultError(fmt.Sprintf("workflow %s already running (session %s). Call devkit_advance to continue or devkit_status to check.", existing.Workflow, existing.ID)), nil
		}

		wfName, err := req.RequireString("workflow")
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("missing argument: %v", err)), nil
		}
		input, err := req.RequireString("input")
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("missing argument: %v", err)), nil
		}

		// Reject workflow names with path separators or dot-dot traversal sequences.
		if strings.ContainsAny(wfName, `/\`) || strings.Contains(wfName, "..") {
			return mcpmcp.NewToolResultError(fmt.Sprintf("invalid workflow name %q: must not contain path separators", wfName)), nil
		}

		// Find and parse workflow — resolve and verify the path stays inside workflowDir.
		wfPath, err := s.resolveWorkflowPath(wfName)
		if err != nil {
			return mcpmcp.NewToolResultError(err.Error()), nil
		}
		wf, err := engine.ParseFile(wfPath)
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("parse workflow %q: %v", wfName, err)), nil
		}

		if len(wf.Steps) == 0 {
			return mcpmcp.NewToolResultError(fmt.Sprintf("workflow %q has no steps", wfName)), nil
		}

		// Create session — store the validated filename for safe re-parsing in advance
		sessionID := lib.NewSessionID()
		firstStep := wf.Steps[0]

		state := &lib.SessionState{
			ID:           sessionID,
			Workflow:     wfName, // store filename, not wf.Name, to prevent traversal in advance
			Input:        input,
			CurrentStep:  firstStep.ID,
			CurrentIndex: 0,
			TotalSteps:   len(wf.Steps),
			StepType:     stepType(firstStep),
			Enforce:      wf.Enforce,
			Branch:       wf.BranchMode,
			Status:       "running",
			StartedAt:    time.Now(),
			Outputs:      map[string]string{},
		}
		if err := lib.WriteSessionJSON(s.dataDir, state); err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("write state: %v", err)), nil
		}

		// SQLite record
		if s.db != nil {
			if err := s.db.CreateSession(&lib.Session{
				ID:       sessionID,
				Workflow: wf.Name,
				Prompt:   input,
				Status:   "running",
			}); err != nil {
				fmt.Fprintf(os.Stderr, "warning: db create session: %v\n", err)
			}
		}

		// Git branch if configured. Must be a hard error — if a
		// workflow declares branch: true and we silently fall through,
		// completeWorkflow will later commit onto the user's current
		// branch (often main). Roll back session state and DB record
		// before returning.
		if wf.BranchMode {
			if s.git == nil {
				_ = lib.ClearSessionJSON(s.dataDir)
				if s.db != nil {
					_ = s.db.UpdateSessionStatus(sessionID, "failed")
				}
				return mcpmcp.NewToolResultError(fmt.Sprintf("workflow %q requires branch mode but git is not available", wfName)), nil
			}
			branchName := fmt.Sprintf("%s/%s", wf.Name, sessionID)
			if err := s.git.CreateBranch(branchName); err != nil {
				_ = lib.ClearSessionJSON(s.dataDir)
				if s.db != nil {
					_ = s.db.UpdateSessionStatus(sessionID, "failed")
				}
				return mcpmcp.NewToolResultError(fmt.Sprintf("create branch %q: %v (workflow declares branch mode and cannot proceed on the current branch)", branchName, err)), nil
			}
		}

		// Build response with first step + principles
		response := s.formatStepResponse(wf, state, &firstStep, input)
		return mcpmcp.NewToolResultText(response), nil
	}
}

// resolveWorkflowPath finds and validates a workflow file path.
// Returns an error if the name resolves outside workflowDir.
func (s *Server) resolveWorkflowPath(name string) (string, error) {
	absWorkflowDir, err := filepath.Abs(s.workflowDir)
	if err != nil {
		return "", fmt.Errorf("resolve workflow dir: %w", err)
	}

	for _, ext := range []string{".yml", ".yaml"} {
		candidate := filepath.Join(s.workflowDir, name+ext)
		absCandidate, err := filepath.Abs(candidate)
		if err != nil {
			return "", fmt.Errorf("resolve workflow path: %w", err)
		}
		if !strings.HasPrefix(absCandidate, absWorkflowDir+string(filepath.Separator)) {
			return "", fmt.Errorf("invalid workflow name %q: resolves outside workflow directory", name)
		}
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("workflow %q not found", name)
}

func stepType(step engine.WfStep) string {
	if step.Command != "" {
		return "command"
	}
	if len(step.Parallel) > 0 {
		return "parallel"
	}
	return "prompt"
}

func (s *Server) formatStepResponse(wf *engine.Workflow, state *lib.SessionState, step *engine.WfStep, input string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "=== STEP %d/%d: %s ===\n", state.CurrentIndex+1, state.TotalSteps, step.ID)

	if step.Command != "" {
		fmt.Fprintf(&b, "TYPE: command (engine will execute automatically on devkit_advance)\n")
		fmt.Fprintf(&b, "COMMAND: %s\n", step.Command)
		fmt.Fprintf(&b, "ENV: DEVKIT_INPUT=%q", input)
		for id, out := range state.Outputs {
			fmt.Fprintf(&b, ", DEVKIT_OUT_%s=<%d bytes>", sanitizeEnvKey(id), len(out))
		}
		fmt.Fprintln(&b)
		if step.Expect != "" {
			fmt.Fprintf(&b, "EXPECT: %s\n", step.Expect)
		}
	} else if len(step.Parallel) > 0 {
		fmt.Fprintf(&b, "TYPE: parallel dispatch\n")
		fmt.Fprintf(&b, "DISPATCH: %s\n", strings.Join(step.Parallel, ", "))
		fmt.Fprintf(&b, "Use the Agent tool and plugins to run these in parallel, then call devkit_advance.\n")
	} else {
		prompt := engine.Interpolate(step.Prompt, input, state.Outputs)
		fmt.Fprintf(&b, "PROMPT: %s\n", prompt)
	}

	// Inject principles
	principles := step.Principles
	if len(principles) == 0 {
		principles = wf.Principles
	}
	if len(principles) > 0 {
		fmt.Fprintf(&b, "\nPRINCIPLES:\n")
		for _, p := range principles {
			if rules, ok := s.principles[p]; ok {
				fmt.Fprintf(&b, "[%s] %s\n", p, strings.Join(rules, "; "))
			}
		}
	}

	if step.Loop != nil {
		fmt.Fprintf(&b, "\nLOOP: max %d iterations", step.Loop.Max)
		if step.Loop.Gate != "" {
			fmt.Fprintf(&b, ", gate: %s", step.Loop.Gate)
		}
		if step.Loop.Until != "" {
			fmt.Fprintf(&b, ", until: %s", step.Loop.Until)
		}
		fmt.Fprintln(&b)
		if rules, ok := s.principles["scratchpad"]; ok {
			fmt.Fprintf(&b, "[scratchpad] %s\n", strings.Join(rules, "; "))
		}
		if rules, ok := s.principles["stuck"]; ok {
			fmt.Fprintf(&b, "[stuck] %s\n", strings.Join(rules, "; "))
		}
	}

	fmt.Fprintf(&b, "\nCall devkit_advance when this step is complete.\n")
	return b.String()
}

func (s *Server) advanceTool() (mcpmcp.Tool, mcpgo.ToolHandlerFunc) {
	tool := mcpmcp.NewTool("devkit_advance",
		mcpmcp.WithDescription("Complete current step and get next"),
		mcpmcp.WithString("session", mcpmcp.Required(), mcpmcp.Description("Session ID")),
		mcpmcp.WithString("output", mcpmcp.Description("Summary of step output (for prompt steps)")),
	)
	return tool, func(ctx context.Context, req mcpmcp.CallToolRequest) (*mcpmcp.CallToolResult, error) {
		sessionID, err := req.RequireString("session")
		if err != nil {
			return mcpmcp.NewToolResultError("missing session argument"), nil
		}

		// Claim the advance slot atomically under the session lock. If
		// another advance is already in progress we reject — letting
		// both proceed would race on the current step index and could
		// execute the same command step twice or skip one.
		state, err := lib.UpdateSessionJSON(s.dataDir, func(cur *lib.SessionState) (*lib.SessionState, error) {
			if cur == nil {
				return nil, fmt.Errorf("no active session")
			}
			if cur.ID != sessionID {
				return nil, fmt.Errorf("session mismatch: active is %s", cur.ID)
			}
			if cur.Busy {
				return nil, fmt.Errorf("step %s already in progress (another devkit_advance call holds the claim)", cur.CurrentStep)
			}
			cur.Busy = true
			return cur, nil
		})
		if err != nil {
			return mcpmcp.NewToolResultError(err.Error()), nil
		}

		// Ensure the claim is released no matter how we exit. On the
		// success path we explicitly clear Busy before the final write;
		// this deferred clear is the safety net for panics, errors, or
		// early returns in the handlers below.
		claimReleased := false
		releaseClaim := func() {
			if claimReleased {
				return
			}
			claimReleased = true
			_, _ = lib.UpdateSessionJSON(s.dataDir, func(cur *lib.SessionState) (*lib.SessionState, error) {
				if cur == nil || cur.ID != sessionID {
					return nil, nil
				}
				cur.Busy = false
				return cur, nil
			})
		}
		defer releaseClaim()

		// Re-parse workflow using validated filename stored in state.Workflow
		wfPath, err := s.resolveWorkflowPath(state.Workflow)
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("resolve workflow: %v", err)), nil
		}
		wf, err := engine.ParseFile(wfPath)
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("parse workflow: %v", err)), nil
		}

		// Bounds check against corrupted state
		if state.CurrentIndex < 0 || state.CurrentIndex >= len(wf.Steps) {
			return mcpmcp.NewToolResultError(fmt.Sprintf("invalid step index %d (workflow has %d steps)", state.CurrentIndex, len(wf.Steps))), nil
		}

		currentStep := wf.Steps[state.CurrentIndex]

		// Handle command steps — engine executes them. Command strings
		// are run literally (no {{...}} expansion); values are passed
		// through env vars DEVKIT_INPUT and DEVKIT_OUT_<step_id> to
		// avoid shell injection via LLM-chosen input or contaminated
		// prior-step output.
		if currentStep.Command != "" {
			output, exitCode, cmdErr := s.runCommand(ctx, currentStep.Command, state)
			if cmdErr != nil {
				return mcpmcp.NewToolResultError(fmt.Sprintf("command failed: %v", cmdErr)), nil
			}

			// Check expect
			if currentStep.Expect == "failure" && exitCode == 0 {
				return mcpmcp.NewToolResultError(fmt.Sprintf("step %s: expected failure but got exit 0", currentStep.ID)), nil
			}
			if currentStep.Expect == "success" && exitCode != 0 {
				return mcpmcp.NewToolResultError(fmt.Sprintf("step %s: expected success but got exit %d\n%s", currentStep.ID, exitCode, output)), nil
			}

			state.Outputs[currentStep.ID] = output
		} else {
			// Prompt/parallel step — record output from Claude
			args := req.GetArguments()
			if outputArg, ok := args["output"]; ok && outputArg != nil {
				if outputStr, ok := outputArg.(string); ok {
					state.Outputs[currentStep.ID] = outputStr
				}
			}
		}

		// Handle loop steps. handleLoopAdvance writes state (clearing
		// Busy) on all its return paths, so mark the claim released
		// here so the deferred release does not double-write.
		if currentStep.Loop != nil {
			claimReleased = true
			return s.handleLoopAdvance(ctx, wf, state, &currentStep, req)
		}

		// Advance to next step
		nextIndex := state.CurrentIndex + 1

		// Check branch conditions
		if len(currentStep.Branch) > 0 {
			if output, ok := state.Outputs[currentStep.ID]; ok {
				target := engine.EvalBranch(output, currentStep.Branch)
				if target != "" {
					for i, step := range wf.Steps {
						if step.ID == target {
							nextIndex = i
							break
						}
					}
				}
			}
		}

		if nextIndex >= len(wf.Steps) {
			claimReleased = true // completeWorkflow clears the whole state
			return s.completeWorkflow(state)
		}

		// Write next step state. Clear the claim as part of this same
		// write so hooks observing session.json mid-transition never
		// see Busy=true with a stale step index.
		nextStep := wf.Steps[nextIndex]
		state.CurrentStep = nextStep.ID
		state.CurrentIndex = nextIndex
		state.StepType = stepType(nextStep)
		state.Busy = false
		if err := lib.WriteSessionJSON(s.dataDir, state); err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("write state: %v", err)), nil
		}
		claimReleased = true

		response := s.formatStepResponse(wf, state, &nextStep, state.Input)
		return mcpmcp.NewToolResultText(response), nil
	}
}

// completeWorkflow marks a session as done, updates DB, and clears hot state.
// Any warnings (DB update failure, commit failure, state clear failure)
// are collected and surfaced in the user-visible response so silent
// post-completion failures are observable.
func (s *Server) completeWorkflow(state *lib.SessionState) (*mcpmcp.CallToolResult, error) {
	state.Status = "done"
	if err := lib.WriteSessionJSON(s.dataDir, state); err != nil {
		return mcpmcp.NewToolResultError(fmt.Sprintf("write final state: %v", err)), nil
	}

	var warnings []string
	if s.db != nil {
		if err := s.db.UpdateSessionStatus(state.ID, "done"); err != nil {
			warnings = append(warnings, fmt.Sprintf("db session status update failed: %v", err))
		}
	}

	if state.Branch && s.git != nil {
		if err := s.git.CommitAll(fmt.Sprintf("%s(%s): complete", state.Workflow, state.ID)); err != nil {
			// Don't swallow — the user's branch work may not be
			// persisted. Report prominently.
			warnings = append(warnings, fmt.Sprintf("final git commit failed: %v (your working tree may have uncommitted changes)", err))
		}
	}

	if err := lib.ClearSessionJSON(s.dataDir); err != nil {
		warnings = append(warnings, fmt.Sprintf("clear session state failed: %v (the hot state file at %s may be stale)", err, s.dataDir))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "=== WORKFLOW COMPLETE ===\nSession: %s\nSteps completed: %d", state.ID, state.TotalSteps)
	if len(warnings) > 0 {
		fmt.Fprintf(&b, "\n\n=== WARNINGS (non-fatal) ===")
		for _, w := range warnings {
			fmt.Fprintf(&b, "\n- %s", w)
			fmt.Fprintf(os.Stderr, "devkit completeWorkflow: %s\n", w)
		}
	}
	return mcpmcp.NewToolResultText(b.String()), nil
}

// runCommand executes a workflow command string under sh -c, passing the
// session's Input and prior step Outputs as environment variables rather
// than interpolating them into the shell string. This eliminates shell
// injection via LLM-chosen input or contaminated prior-step output — the
// command text is always the literal YAML value.
func (s *Server) runCommand(ctx context.Context, command string, state *lib.SessionState) (string, int, error) {
	return s.runCommandWithTimeout(ctx, command, state, commandTimeout)
}

// runCommandWithTimeout is the general form — gate commands call this
// with gateTimeout so a stuck gate does not eat the full command budget.
func (s *Server) runCommandWithTimeout(ctx context.Context, command string, state *lib.SessionState, timeout time.Duration) (string, int, error) {
	// Nest a fresh deadline on top of the parent. Parent cancellation
	// still propagates (e.g. MCP request abort), but the effective
	// deadline is now min(parent deadline, now+timeout).
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = s.repoRoot
	cmd.Env = append(os.Environ(), commandEnv(state)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			// Surface whatever the command produced on its combined
			// stream so the user can see the real cause (missing
			// binary, permission denied, timeout), not just the Go
			// error wrapper.
			if ctx.Err() == context.DeadlineExceeded {
				return out.String(), 124, fmt.Errorf("command timed out after %s: %w", commandTimeout, err)
			}
			return out.String(), 1, fmt.Errorf("command execution failed: %w", err)
		}
	}
	return out.String(), exitCode, nil
}

// commandEnv returns the DEVKIT_INPUT and DEVKIT_OUT_<id> env vars that
// command steps can read via $DEVKIT_INPUT / $DEVKIT_OUT_<id>.
func commandEnv(state *lib.SessionState) []string {
	env := []string{"DEVKIT_INPUT=" + state.Input}
	for id, out := range state.Outputs {
		env = append(env, "DEVKIT_OUT_"+sanitizeEnvKey(id)+"="+out)
	}
	return env
}

// sanitizeEnvKey maps a workflow step ID to a valid env var suffix.
// POSIX allows [A-Za-z_][A-Za-z0-9_]*; step IDs may contain hyphens.
func sanitizeEnvKey(id string) string {
	b := make([]byte, 0, len(id))
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z':
			b = append(b, c-32) // upper
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	return string(b)
}

func (s *Server) handleLoopAdvance(ctx context.Context, wf *engine.Workflow, state *lib.SessionState, step *engine.WfStep, req mcpmcp.CallToolRequest) (*mcpmcp.CallToolResult, error) {
	// Initialize loop tracking on first call
	if state.LoopMax == 0 {
		state.LoopMax = step.Loop.Max
		if state.LoopMax == 0 {
			state.LoopMax = 10 // default max
		}
	}
	state.LoopIteration++

	// Check gate command if present. Gate strings are also literal —
	// values come via env vars DEVKIT_INPUT / DEVKIT_OUT_<step_id>.
	// Gates get a shorter independent timeout so a wedged gate cannot
	// eat the full command budget.
	if step.Loop.Gate != "" {
		gateOut, exitCode, err := s.runCommandWithTimeout(ctx, step.Loop.Gate, state, gateTimeout)
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("gate command failed: %v\n%s", err, gateOut)), nil
		}
		if exitCode == 0 {
			// Gate passed — advance past loop
			return s.advancePastLoop(wf, state)
		}
		// Gate failed — continue loop
	}

	// Check "until" condition. Line-anchored match (see engine.MatchUntil)
	// so sentinels like "DONE" do not match prose mentions.
	if step.Loop.Until != "" {
		if output, ok := state.Outputs[step.ID]; ok {
			if engine.MatchUntil(output, step.Loop.Until) {
				return s.advancePastLoop(wf, state)
			}
		}
	}

	// Check max iterations
	if state.LoopIteration >= state.LoopMax {
		return s.advancePastLoop(wf, state)
	}

	// Continue loop — return same step for another iteration.
	// Clear the advance claim as part of this write (see advanceTool).
	state.Busy = false
	if err := lib.WriteSessionJSON(s.dataDir, state); err != nil {
		return mcpmcp.NewToolResultError(fmt.Sprintf("write loop state: %v", err)), nil
	}
	response := fmt.Sprintf("=== LOOP ITERATION %d/%d: %s ===\n", state.LoopIteration, state.LoopMax, step.ID)
	response += s.formatStepResponse(wf, state, step, state.Input)
	return mcpmcp.NewToolResultText(response), nil
}

func (s *Server) advancePastLoop(wf *engine.Workflow, state *lib.SessionState) (*mcpmcp.CallToolResult, error) {
	nextIndex := state.CurrentIndex + 1
	// Reset loop state
	state.LoopIteration = 0
	state.LoopMax = 0

	if nextIndex >= len(wf.Steps) {
		return s.completeWorkflow(state)
	}

	nextStep := wf.Steps[nextIndex]
	state.CurrentStep = nextStep.ID
	state.CurrentIndex = nextIndex
	state.StepType = stepType(nextStep)
	state.Busy = false
	if err := lib.WriteSessionJSON(s.dataDir, state); err != nil {
		return mcpmcp.NewToolResultError(fmt.Sprintf("write state: %v", err)), nil
	}

	response := s.formatStepResponse(wf, state, &nextStep, state.Input)
	return mcpmcp.NewToolResultText(response), nil
}
