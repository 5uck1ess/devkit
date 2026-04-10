package mcp

import (
	"context"
	"fmt"
	"os"
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
	tool := mcpmcp.NewTool("workflow_advance",
		mcpmcp.WithDescription("Advance a workflow step (stub)"),
	)
	handler := func(ctx context.Context, req mcpmcp.CallToolRequest) (*mcpmcp.CallToolResult, error) {
		return mcpmcp.NewToolResultText("not implemented"), nil
	}
	return tool, handler
}
