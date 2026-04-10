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
		// Check no active session
		existing, _ := lib.ReadSessionJSON(s.dataDir)
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

		// Reject workflow names that contain path separators or traversal sequences.
		// The name must be a plain filename component — no slashes or dots that
		// would escape the workflow directory.
		if strings.ContainsAny(wfName, `/\`) || strings.Contains(wfName, "..") {
			return mcpmcp.NewToolResultError(fmt.Sprintf("invalid workflow name %q: must not contain path separators", wfName)), nil
		}

		// Find and parse workflow — resolve and verify the path stays inside workflowDir.
		wfPath := filepath.Join(s.workflowDir, wfName+".yml")
		if _, err := os.Stat(wfPath); os.IsNotExist(err) {
			wfPath = filepath.Join(s.workflowDir, wfName+".yaml")
		}
		// Guard: resolved path must be inside workflowDir (defense-in-depth).
		absWorkflowDir, _ := filepath.Abs(s.workflowDir)
		absWfPath, _ := filepath.Abs(wfPath)
		if !strings.HasPrefix(absWfPath, absWorkflowDir+string(filepath.Separator)) {
			return mcpmcp.NewToolResultError(fmt.Sprintf("invalid workflow name %q: resolves outside workflow directory", wfName)), nil
		}
		wf, err := engine.ParseFile(wfPath)
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("parse workflow %q: %v", wfName, err)), nil
		}

		// Create session
		sessionID := lib.NewSessionID()
		firstStep := wf.Steps[0]

		state := &lib.SessionState{
			ID:           sessionID,
			Workflow:     wf.Name,
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
			dbSession := &lib.Session{
				ID:       sessionID,
				Workflow: wf.Name,
				Prompt:   input,
				Status:   "running",
			}
			s.db.CreateSession(dbSession)
		}

		// Git branch if configured
		if wf.BranchMode && s.git != nil {
			branchName := fmt.Sprintf("%s/%s", wf.Name, sessionID)
			if err := s.git.CreateBranch(branchName); err != nil {
				fmt.Fprintf(os.Stderr, "warning: branch creation failed: %v\n", err)
			}
		}

		// Build response with first step + principles
		response := s.formatStepResponse(wf, state, &firstStep, input)
		return mcpmcp.NewToolResultText(response), nil
	}
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
		cmd := engine.Interpolate(step.Command, input, state.Outputs)
		fmt.Fprintf(&b, "TYPE: command (engine will execute automatically on devkit_advance)\n")
		fmt.Fprintf(&b, "COMMAND: %s\n", cmd)
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

		state, err := lib.ReadSessionJSON(s.dataDir)
		if err != nil || state == nil {
			return mcpmcp.NewToolResultError("no active session"), nil
		}
		if state.ID != sessionID {
			return mcpmcp.NewToolResultError(fmt.Sprintf("session mismatch: active is %s", state.ID)), nil
		}

		// Re-parse workflow to get step definitions.
		// Guard: resolved path must stay inside workflowDir — state.Workflow comes
		// from the YAML name field which may differ from the validated filename.
		absWorkflowDir, _ := filepath.Abs(s.workflowDir)
		wfPath := filepath.Join(s.workflowDir, state.Workflow+".yml")
		absWfPath, _ := filepath.Abs(wfPath)
		if !strings.HasPrefix(absWfPath, absWorkflowDir+string(filepath.Separator)) {
			return mcpmcp.NewToolResultError(fmt.Sprintf("invalid workflow name in session %q: resolves outside workflow directory", state.Workflow)), nil
		}
		if _, statErr := os.Stat(wfPath); os.IsNotExist(statErr) {
			wfPath = filepath.Join(s.workflowDir, state.Workflow+".yaml")
			absWfPath, _ = filepath.Abs(wfPath)
			if !strings.HasPrefix(absWfPath, absWorkflowDir+string(filepath.Separator)) {
				return mcpmcp.NewToolResultError(fmt.Sprintf("invalid workflow name in session %q: resolves outside workflow directory", state.Workflow)), nil
			}
		}
		wf, err := engine.ParseFile(wfPath)
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("parse workflow: %v", err)), nil
		}

		currentStep := wf.Steps[state.CurrentIndex]

		// Handle command steps — engine executes them
		if currentStep.Command != "" {
			cmd := engine.Interpolate(currentStep.Command, state.Input, state.Outputs)
			output, exitCode, cmdErr := s.runCommand(ctx, cmd)
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

		// Handle loop steps — delegate to handleLoopAdvance (Task 9)
		if currentStep.Loop != nil {
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
			// Workflow complete
			state.Status = "done"
			lib.WriteSessionJSON(s.dataDir, state)
			if s.db != nil {
				s.db.UpdateSessionStatus(state.ID, "done")
			}

			if state.Branch && s.git != nil {
				s.git.CommitAll(fmt.Sprintf("%s(%s): complete", state.Workflow, state.ID))
			}

			lib.ClearSessionJSON(s.dataDir)
			return mcpmcp.NewToolResultText(fmt.Sprintf("=== WORKFLOW COMPLETE ===\nSession: %s\nSteps completed: %d", state.ID, state.TotalSteps)), nil
		}

		// Write next step state
		nextStep := wf.Steps[nextIndex]
		state.CurrentStep = nextStep.ID
		state.CurrentIndex = nextIndex
		state.StepType = stepType(nextStep)
		lib.WriteSessionJSON(s.dataDir, state)

		response := s.formatStepResponse(wf, state, &nextStep, state.Input)
		return mcpmcp.NewToolResultText(response), nil
	}
}

func (s *Server) runCommand(ctx context.Context, command string) (string, int, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = s.repoRoot
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
			return "", 1, fmt.Errorf("command execution failed: %w", err)
		}
	}
	return out.String(), exitCode, nil
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

	// Check gate command if present
	if step.Loop.Gate != "" {
		gateCmd := engine.Interpolate(step.Loop.Gate, state.Input, state.Outputs)
		_, exitCode, err := s.runCommand(ctx, gateCmd)
		if err != nil {
			return mcpmcp.NewToolResultError(fmt.Sprintf("gate command failed: %v", err)), nil
		}
		if exitCode == 0 {
			// Gate passed — advance past loop
			return s.advancePastLoop(wf, state), nil
		}
		// Gate failed — continue loop
	}

	// Check "until" condition
	if step.Loop.Until != "" {
		if output, ok := state.Outputs[step.ID]; ok {
			if strings.Contains(strings.ToLower(output), strings.ToLower(step.Loop.Until)) {
				return s.advancePastLoop(wf, state), nil
			}
		}
	}

	// Check max iterations
	if state.LoopIteration >= state.LoopMax {
		return s.advancePastLoop(wf, state), nil
	}

	// Continue loop — return same step for another iteration
	lib.WriteSessionJSON(s.dataDir, state)
	response := fmt.Sprintf("=== LOOP ITERATION %d/%d: %s ===\n", state.LoopIteration, state.LoopMax, step.ID)
	response += s.formatStepResponse(wf, state, step, state.Input)
	return mcpmcp.NewToolResultText(response), nil
}

func (s *Server) advancePastLoop(wf *engine.Workflow, state *lib.SessionState) *mcpmcp.CallToolResult {
	nextIndex := state.CurrentIndex + 1
	// Reset loop state
	state.LoopIteration = 0
	state.LoopMax = 0

	if nextIndex >= len(wf.Steps) {
		state.Status = "done"
		lib.WriteSessionJSON(s.dataDir, state)
		if s.db != nil {
			s.db.UpdateSessionStatus(state.ID, "done")
		}
		lib.ClearSessionJSON(s.dataDir)
		return mcpmcp.NewToolResultText(fmt.Sprintf("=== WORKFLOW COMPLETE ===\nSession: %s\nSteps completed: %d", state.ID, state.TotalSteps))
	}

	nextStep := wf.Steps[nextIndex]
	state.CurrentStep = nextStep.ID
	state.CurrentIndex = nextIndex
	state.StepType = stepType(nextStep)
	lib.WriteSessionJSON(s.dataDir, state)

	response := s.formatStepResponse(wf, state, &nextStep, state.Input)
	return mcpmcp.NewToolResultText(response)
}
