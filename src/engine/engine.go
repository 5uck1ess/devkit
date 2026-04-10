package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/runners"
)

// Engine executes parsed workflows using a runner and database.
type Engine struct {
	db       *lib.DB
	git      *lib.Git
	runner   runners.Runner
	repoRoot string
}

// NewEngine creates a validated Engine. All fields are required.
func NewEngine(db *lib.DB, git *lib.Git, runner runners.Runner, repoRoot string) (*Engine, error) {
	if db == nil {
		return nil, fmt.Errorf("engine: db is required")
	}
	if git == nil {
		return nil, fmt.Errorf("engine: git is required")
	}
	if runner == nil {
		return nil, fmt.Errorf("engine: runner is required")
	}
	if repoRoot == "" {
		return nil, fmt.Errorf("engine: repoRoot is required")
	}
	return &Engine{db: db, git: git, runner: runner, repoRoot: repoRoot}, nil
}

// RunConfig holds per-invocation settings.
// BudgetUSD of 0 means unlimited. Negative values are rejected.
type RunConfig struct {
	Input     string
	BudgetUSD float64
}

// Result contains workflow execution results.
type Result struct {
	Session  *lib.Session
	Steps    []lib.Step
	Outputs  map[string]string
	TotalUSD float64
}

// RunWorkflow executes a parsed workflow end-to-end.
func (e *Engine) RunWorkflow(ctx context.Context, wf *Workflow, cfg RunConfig) (*Result, error) {
	// Validate inputs at the engine boundary
	if err := wf.Validate(); err != nil {
		return nil, fmt.Errorf("invalid workflow: %w", err)
	}
	if cfg.BudgetUSD < 0 {
		return nil, fmt.Errorf("invalid budget: %.2f (must be >= 0)", cfg.BudgetUSD)
	}

	session := &lib.Session{
		ID:        lib.NewSessionID(),
		Workflow:  strings.ToLower(wf.Name),
		Prompt:    cfg.Input,
		Status:    "running",
		BudgetUSD: cfg.BudgetUSD,
	}
	if err := e.db.CreateSession(session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if err := lib.EnsureSessionDir(e.repoRoot, session.ID); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}

	branchName := fmt.Sprintf("%s/%s", session.Workflow, session.ID)
	if err := e.git.CreateBranch(branchName); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}
	fmt.Printf("%s session %s on branch %s\n\n", wf.Name, session.ID, branchName)

	// Ensure scratchpad directory exists
	scratchDir := filepath.Join(e.repoRoot, ".devkit", "scratchpads")
	if err := os.MkdirAll(scratchDir, 0o755); err != nil {
		return nil, fmt.Errorf("create scratchpad dir: %w", err)
	}

	outputs := make(map[string]string)
	stepIndex := buildStepIndex(wf.Steps)
	var totalUSD float64
	var iterNum int

	opts := runners.RunOpts{
		WorkDir:      e.repoRoot,
		AllowedTools: "Bash,Read,Edit,Write,Grep,Glob",
		MaxTurns:     30,
	}

	overBudget := func() bool {
		return cfg.BudgetUSD > 0 && totalUSD >= cfg.BudgetUSD
	}
	// addCost updates the running total (used by loops to keep overBudget accurate)
	addCost := func(c float64) { totalUSD += c }

	// Build set of step IDs that are dispatched by parallel steps,
	// so we skip them during sequential walk (they run inside runParallel).
	parallelChildren := make(map[string]bool)
	for _, s := range wf.Steps {
		for _, pid := range s.Parallel {
			parallelChildren[pid] = true
		}
	}

	// Walk steps sequentially, with branch jumps
	failed := false
	var stepErr error
	branchCount := 0
	const maxBranches = 100

	// evalBranch checks branch conditions and returns the next step index, or -1 for fall-through.
	evalBranch := func(step *WfStep) (int, error) {
		if len(step.Branch) == 0 {
			return -1, nil
		}
		output, ok := outputs[step.ID]
		if !ok {
			return -1, nil
		}
		target := EvalBranch(output, step.Branch)
		if target == "" {
			return -1, nil
		}
		branchCount++
		if branchCount > maxBranches {
			fmt.Println("  → branch limit reached, stopping")
			return -1, fmt.Errorf("branch limit exceeded (%d jumps)", maxBranches)
		}
		fmt.Printf("  → branching to %s\n\n", target)
		return stepIndex[target], nil
	}

	i := 0
	for i < len(wf.Steps) {
		if ctx.Err() != nil {
			break
		}
		if overBudget() {
			fmt.Printf("  Budget exhausted ($%.2f of $%.2f)\n", totalUSD, cfg.BudgetUSD)
			break
		}

		step := &wf.Steps[i]

		// Skip steps that are dispatched by a parallel step
		if parallelChildren[step.ID] {
			i++
			continue
		}

		// Parallel dispatcher step
		if step.Prompt == "" && len(step.Parallel) > 0 {
			cost, err := e.runParallel(ctx, step, wf.Steps, stepIndex, session, cfg.Input, outputs, opts, &iterNum)
			if err != nil {
				e.db.UpdateSessionStatus(session.ID, "failed")
				failed = true
				stepErr = err
				break
			}
			totalUSD += cost
			i++
			continue
		}

		// Skip empty steps (no prompt, no command)
		if step.Prompt == "" && step.Command == "" {
			i++
			continue
		}

		if step.Loop != nil {
			// Note: addCost updates totalUSD live for budget checks inside the loop,
			// so we don't add the returned cost again here.
			_, err := e.runLoop(ctx, step, session, cfg.Input, outputs, opts, &iterNum, overBudget, addCost)
			if err != nil {
				e.db.UpdateSessionStatus(session.ID, "failed")
				failed = true
				stepErr = err
				break
			}
		} else {
			cost, output, err := e.runStep(ctx, step, session, cfg.Input, outputs, opts, &iterNum)
			if err != nil {
				e.db.UpdateSessionStatus(session.ID, "failed")
				failed = true
				stepErr = err
				break
			}
			totalUSD += cost
			outputs[step.ID] = output
		}

		// Evaluate branch (applies to both loop and regular steps)
		jump, err := evalBranch(step)
		if err != nil {
			failed = true
			stepErr = err
			break
		}
		if jump >= 0 {
			i = jump
			continue
		}

		i++
	}

	// Clean up scratchpad (best-effort)
	_ = os.Remove(filepath.Join(scratchDir, "current.md"))

	// Only mark done on clean exit (fix #3: don't overwrite "failed")
	if !failed && ctx.Err() == nil {
		e.git.CommitAll(fmt.Sprintf("%s(%s): complete", session.Workflow, session.ID))
		e.db.UpdateSessionStatus(session.ID, "done")
	} else if !failed {
		e.db.UpdateSessionStatus(session.ID, "cancelled")
	}

	allSteps, _ := e.db.GetSteps(session.ID)
	stopReason := "completed"
	if failed {
		stopReason = "failed"
	} else if ctx.Err() != nil {
		stopReason = "cancelled"
	} else if overBudget() {
		stopReason = "budget_exhausted"
	}
	lib.WriteReport(e.repoRoot, session, allSteps, stopReason)

	return &Result{
		Session:  session,
		Steps:    allSteps,
		Outputs:  outputs,
		TotalUSD: totalUSD,
	}, stepErr
}

// runCommand executes a shell command and returns its combined output.
// The caller passes input and outputs through env vars (DEVKIT_INPUT and
// DEVKIT_OUT_<id>) rather than interpolating them into the command string,
// to eliminate shell injection via LLM-chosen input or contaminated
// prior-step output.
func (e *Engine) runCommand(ctx context.Context, command, input string, outputs map[string]string) (string, int, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = e.repoRoot
	cmd.Env = append(os.Environ(), buildCommandEnv(input, outputs)...)

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
			return out.String(), 1, fmt.Errorf("command execution failed: %w", err)
		}
	}
	return out.String(), exitCode, nil
}

// buildCommandEnv returns DEVKIT_INPUT and DEVKIT_OUT_<step_id> env vars
// for use by command/gate steps.
func buildCommandEnv(input string, outputs map[string]string) []string {
	env := []string{"DEVKIT_INPUT=" + input}
	for id, out := range outputs {
		env = append(env, "DEVKIT_OUT_"+envKey(id)+"="+out)
	}
	return env
}

// envKey maps a step ID (may contain hyphens) to a POSIX env var suffix.
func envKey(id string) string {
	b := make([]byte, 0, len(id))
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z':
			b = append(b, c-32)
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	return string(b)
}

// runStep executes a single step and records it in the database.
func (e *Engine) runStep(ctx context.Context, step *WfStep, session *lib.Session, input string, outputs map[string]string, opts runners.RunOpts, iterNum *int) (float64, string, error) {
	*iterNum++

	// Command step: run shell command directly, no LLM cost.
	if step.Command != "" {
		fmt.Printf("--- %s (step %d, command) ---\n", step.ID, *iterNum)

		dbStep := &lib.Step{
			SessionID: session.ID,
			Iteration: *iterNum,
			Status:    "running",
			AgentName: "shell",
		}
		e.db.CreateStep(dbStep)

		// Command string is literal — no {{...}} expansion. Values
		// come via env vars ($DEVKIT_INPUT, $DEVKIT_OUT_<step_id>).
		output, exitCode, err := e.runCommand(ctx, step.Command, input, outputs)
		if err != nil {
			dbStep.Status = "failed"
			dbStep.ChangeSummary = err.Error()
			e.db.UpdateStep(dbStep)
			return 0, "", fmt.Errorf("step %s command failed: %w", step.ID, err)
		}

		// Check expect condition on exit code.
		if step.Expect == "failure" && exitCode == 0 {
			reason := "expected failure but got exit code 0"
			dbStep.Status = "failed"
			dbStep.ChangeSummary = reason
			e.db.UpdateStep(dbStep)
			fmt.Printf("  %s\n\n", reason)
			return 0, "", fmt.Errorf("step %s: %s", step.ID, reason)
		}
		if step.Expect == "success" && exitCode != 0 {
			reason := fmt.Sprintf("expected success but got exit code %d", exitCode)
			dbStep.Status = "failed"
			dbStep.ChangeSummary = reason
			e.db.UpdateStep(dbStep)
			fmt.Printf("  %s\n\n", reason)
			return 0, "", fmt.Errorf("step %s: %s", step.ID, reason)
		}

		// Include exit code in output so downstream steps can check it
		fullOutput := fmt.Sprintf("%s\nexit code: %d", strings.TrimRight(output, "\n"), exitCode)

		dbStep.Status = "kept"
		dbStep.Kept = true
		dbStep.ChangeSummary = runners.TruncStr(fullOutput, 200)
		e.db.UpdateStep(dbStep)
		fmt.Printf("  done (exit %d)\n\n", exitCode)

		return 0, fullOutput, nil
	}

	// Prompt step: run through LLM runner.
	fmt.Printf("--- %s (step %d) ---\n", step.ID, *iterNum)

	prompt := Interpolate(step.Prompt, input, outputs)
	dbStep := &lib.Step{
		SessionID: session.ID,
		Iteration: *iterNum,
		Status:    "running",
		AgentName: e.runner.Name(),
	}
	e.db.CreateStep(dbStep)

	result, err := e.runner.Run(ctx, prompt, opts)
	if err != nil {
		dbStep.Status = "failed"
		dbStep.ChangeSummary = err.Error()
		e.db.UpdateStep(dbStep)
		return 0, "", fmt.Errorf("step %s failed: %w", step.ID, err)
	}

	dbStep.Status = "kept"
	dbStep.Kept = true
	dbStep.CostUSD = result.CostUSD
	dbStep.ChangeSummary = runners.TruncStr(result.Output, 200)
	e.db.UpdateStep(dbStep)
	fmt.Printf("  done ($%.4f)\n\n", result.CostUSD)

	return result.CostUSD, result.Output, nil
}

// runLoop executes a step repeatedly until the until-string is found or max iterations reached.
// Returns an error if all iterations fail. Respects budget via overBudget, reports cost via addCost.
// If a gate command is set, it runs after each iteration — non-zero exit reverts the iteration.
func (e *Engine) runLoop(ctx context.Context, step *WfStep, session *lib.Session, input string, outputs map[string]string, opts runners.RunOpts, iterNum *int, overBudget func() bool, addCost func(float64)) (float64, error) {
	var totalCost float64
	maxIter := step.Loop.Max
	if maxIter <= 0 {
		maxIter = 1
	}

	succeeded := false
	consecutiveFailures := 0
	stuckDetected := false

	for attempt := 1; attempt <= maxIter; attempt++ {
		if ctx.Err() != nil {
			return totalCost, ctx.Err()
		}
		if overBudget != nil && overBudget() {
			fmt.Printf("  → budget exhausted, stopping loop\n")
			break
		}

		*iterNum++
		fmt.Printf("--- %s [%d/%d] (step %d) ---\n", step.ID, attempt, maxIter, *iterNum)

		prompt := Interpolate(step.Prompt, input, outputs)
		dbStep := &lib.Step{
			SessionID: session.ID,
			Iteration: *iterNum,
			Status:    "running",
			AgentName: e.runner.Name(),
		}
		e.db.CreateStep(dbStep)

		result, err := e.runner.Run(ctx, prompt, opts)
		if err != nil {
			dbStep.Status = "failed"
			dbStep.ChangeSummary = err.Error()
			e.db.UpdateStep(dbStep)
			consecutiveFailures++
			fmt.Printf("  failed, retrying\n\n")
			continue
		}

		iterCost := result.CostUSD

		// Gate check: run shell command, revert if non-zero exit.
		// Gate string is literal — values come via env vars.
		if step.Loop.Gate != "" {
			fmt.Printf("  gate: %s\n", runners.TruncStr(step.Loop.Gate, 80))
			_, exitCode, gateErr := e.runCommand(ctx, step.Loop.Gate, input, outputs)

			// Distinguish "gate couldn't execute" from "gate ran and returned non-zero".
			// A startup/context error is fatal — the gate never validated anything.
			if gateErr != nil {
				dbStep.Status = "failed"
				dbStep.CostUSD = iterCost
				dbStep.ChangeSummary = fmt.Sprintf("gate error: %s", gateErr)
				e.db.UpdateStep(dbStep)
				totalCost += iterCost
				if addCost != nil {
					addCost(iterCost)
				}
				return totalCost, fmt.Errorf("loop %s: gate command failed: %w", step.ID, gateErr)
			}

			if exitCode != 0 {
				reason := fmt.Sprintf("gate failed (exit %d)", exitCode)
				fmt.Printf("  → %s, reverting iteration\n\n", reason)
				if revertErr := e.git.RevertAll(); revertErr != nil {
					fmt.Printf("  → revert failed: %s\n", revertErr)
					dbStep.Status = "failed"
					dbStep.ChangeSummary = fmt.Sprintf("%s; revert failed: %s", reason, revertErr)
					e.db.UpdateStep(dbStep)
					return totalCost, fmt.Errorf("loop %s: revert failed after gate failure: %w", step.ID, revertErr)
				}
				dbStep.Status = "reverted"
				dbStep.Kept = false
				dbStep.CostUSD = iterCost
				dbStep.ChangeSummary = reason
				e.db.UpdateStep(dbStep)
				consecutiveFailures++
				totalCost += iterCost
				if addCost != nil {
					addCost(iterCost)
				}
				if consecutiveFailures >= 3 {
					fmt.Printf("  → 3 consecutive gate failures, stopping loop\n\n")
					stuckDetected = true
					break
				}
				continue
			}
			fmt.Printf("  → gate passed\n")
		}

		consecutiveFailures = 0
		succeeded = true
		totalCost += iterCost
		if addCost != nil {
			addCost(iterCost)
		}
		dbStep.Status = "kept"
		dbStep.Kept = true
		dbStep.CostUSD = iterCost
		dbStep.ChangeSummary = runners.TruncStr(result.Output, 200)
		e.db.UpdateStep(dbStep)

		outputs[step.ID] = result.Output
		fmt.Printf("  done ($%.4f)\n\n", iterCost)

		// Commit after each successful loop iteration
		if commitErr := e.git.CommitAll(fmt.Sprintf("%s: %s iteration %d", session.Workflow, step.ID, attempt)); commitErr != nil {
			fmt.Printf("  → commit failed: %s\n", commitErr)
		}

		// Check until condition. Line-anchored (see MatchUntil) —
		// sentinel must appear on its own line to avoid matching
		// conversational text.
		if step.Loop.Until != "" && MatchUntil(result.Output, step.Loop.Until) {
			fmt.Printf("  → loop complete (%s found)\n\n", step.Loop.Until)
			return totalCost, nil
		}
	}

	if !succeeded {
		if stuckDetected {
			return totalCost, fmt.Errorf("loop %s: stuck after %d consecutive gate failures (ran %d of %d iterations)", step.ID, consecutiveFailures, consecutiveFailures, maxIter)
		}
		return totalCost, fmt.Errorf("loop %s: all %d iterations failed", step.ID, maxIter)
	}

	return totalCost, nil
}

// runParallel dispatches multiple steps concurrently.
// Returns an error if ALL parallel steps fail. Partial failures are logged but not fatal.
func (e *Engine) runParallel(ctx context.Context, dispatcher *WfStep, allSteps []WfStep, stepIndex map[string]int, session *lib.Session, input string, outputs map[string]string, opts runners.RunOpts, iterNum *int) (float64, error) {
	fmt.Printf("--- %s (parallel: %s) ---\n\n", dispatcher.ID, strings.Join(dispatcher.Parallel, ", "))

	type parallelResult struct {
		id     string
		output string
		cost   float64
		err    error
	}

	// Snapshot outputs before launching goroutines to avoid data race
	outputSnap := make(map[string]string, len(outputs))
	for k, v := range outputs {
		outputSnap[k] = v
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	results := make([]parallelResult, len(dispatcher.Parallel))

	for j, pid := range dispatcher.Parallel {
		idx, ok := stepIndex[pid]
		if !ok {
			return 0, fmt.Errorf("parallel step %q not found", pid)
		}
		step := &allSteps[idx]

		wg.Add(1)
		go func(j int, step *WfStep, pid string) {
			defer wg.Done()

			mu.Lock()
			*iterNum++
			myIter := *iterNum
			mu.Unlock()

			// Use snapshot for interpolation — safe for concurrent reads
			prompt := Interpolate(step.Prompt, input, outputSnap)
			dbStep := &lib.Step{
				SessionID: session.ID,
				Iteration: myIter,
				Status:    "running",
				AgentName: e.runner.Name(),
			}

			mu.Lock()
			e.db.CreateStep(dbStep)
			mu.Unlock()

			result, err := e.runner.Run(ctx, prompt, opts)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				dbStep.Status = "failed"
				dbStep.ChangeSummary = err.Error()
				e.db.UpdateStep(dbStep)
				results[j] = parallelResult{id: pid, err: err}
				return
			}

			dbStep.Status = "kept"
			dbStep.Kept = true
			dbStep.CostUSD = result.CostUSD
			dbStep.ChangeSummary = runners.TruncStr(result.Output, 200)
			e.db.UpdateStep(dbStep)

			results[j] = parallelResult{id: pid, output: result.Output, cost: result.CostUSD}
		}(j, step, pid)
	}

	wg.Wait()

	var totalCost float64
	var failCount int
	var firstErr error
	for _, r := range results {
		if r.err != nil {
			fmt.Printf("  %s: failed (%v)\n", r.id, r.err)
			failCount++
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		outputs[r.id] = r.output
		totalCost += r.cost
		fmt.Printf("  %s: done ($%.4f)\n", r.id, r.cost)
	}
	fmt.Println()

	// Fail only if ALL parallel steps failed
	if failCount == len(results) {
		return totalCost, fmt.Errorf("all parallel steps failed, first: %w", firstErr)
	}

	return totalCost, nil
}

// buildStepIndex maps step IDs to their index in the steps slice.
func buildStepIndex(steps []WfStep) map[string]int {
	idx := make(map[string]int, len(steps))
	for i, s := range steps {
		idx[s.ID] = i
	}
	return idx
}
