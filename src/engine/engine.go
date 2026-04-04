package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/runners"
)

// Engine executes parsed workflows using a runner and database.
type Engine struct {
	DB       *lib.DB
	Git      *lib.Git
	Runner   runners.Runner
	RepoRoot string
}

// RunConfig holds per-invocation settings.
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
	session := &lib.Session{
		ID:        lib.NewSessionID(),
		Workflow:  strings.ToLower(wf.Name),
		Prompt:    cfg.Input,
		Status:    "running",
		BudgetUSD: cfg.BudgetUSD,
	}
	if err := e.DB.CreateSession(session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if err := lib.EnsureSessionDir(e.RepoRoot, session.ID); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}

	branchName := fmt.Sprintf("%s/%s", session.Workflow, session.ID)
	if err := e.Git.CreateBranch(branchName); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}
	fmt.Printf("%s session %s on branch %s\n\n", wf.Name, session.ID, branchName)

	// Ensure scratchpad directory exists
	scratchDir := filepath.Join(e.RepoRoot, ".devkit", "scratchpads")
	if err := os.MkdirAll(scratchDir, 0o755); err != nil {
		return nil, fmt.Errorf("create scratchpad dir: %w", err)
	}

	outputs := make(map[string]string)
	stepIndex := buildStepIndex(wf.Steps)
	var totalUSD float64
	var iterNum int

	opts := runners.RunOpts{
		WorkDir:      e.RepoRoot,
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
				e.DB.UpdateSessionStatus(session.ID, "failed")
				failed = true
				stepErr = err
				break
			}
			totalUSD += cost
			i++
			continue
		}

		// Skip empty steps
		if step.Prompt == "" {
			i++
			continue
		}

		if step.Loop != nil {
			// Note: addCost updates totalUSD live for budget checks inside the loop,
			// so we don't add the returned cost again here.
			_, err := e.runLoop(ctx, step, session, cfg.Input, outputs, opts, &iterNum, overBudget, addCost)
			if err != nil {
				e.DB.UpdateSessionStatus(session.ID, "failed")
				failed = true
				stepErr = err
				break
			}

			// Evaluate branch after loop (fix #10: branches on loop steps)
			if len(step.Branch) > 0 {
				if output, ok := outputs[step.ID]; ok {
					if target := EvalBranch(output, step.Branch); target != "" {
						branchCount++
						if branchCount > maxBranches {
							fmt.Println("  → branch limit reached, stopping")
							failed = true
							stepErr = fmt.Errorf("branch limit exceeded (%d jumps)", maxBranches)
							break
						}
						fmt.Printf("  → branching to %s\n\n", target)
						i = stepIndex[target]
						continue
					}
				}
			}
		} else {
			cost, output, err := e.runStep(ctx, step, session, cfg.Input, outputs, opts, &iterNum)
			if err != nil {
				e.DB.UpdateSessionStatus(session.ID, "failed")
				failed = true
				stepErr = err
				break
			}
			totalUSD += cost
			outputs[step.ID] = output

			// Evaluate branch
			if len(step.Branch) > 0 {
				if target := EvalBranch(output, step.Branch); target != "" {
					branchCount++
					if branchCount > maxBranches {
						fmt.Println("  → branch limit reached, stopping")
						failed = true
						stepErr = fmt.Errorf("branch limit exceeded (%d jumps)", maxBranches)
						break
					}
					fmt.Printf("  → branching to %s\n\n", target)
					i = stepIndex[target]
					continue
				}
			}
		}

		i++
	}

	// Clean up scratchpad (best-effort)
	_ = os.Remove(filepath.Join(scratchDir, "current.md"))

	// Only mark done on clean exit (fix #3: don't overwrite "failed")
	if !failed && ctx.Err() == nil {
		e.Git.CommitAll(fmt.Sprintf("%s(%s): complete", session.Workflow, session.ID))
		e.DB.UpdateSessionStatus(session.ID, "done")
	} else if !failed {
		e.DB.UpdateSessionStatus(session.ID, "cancelled")
	}

	allSteps, _ := e.DB.GetSteps(session.ID)
	stopReason := "completed"
	if failed {
		stopReason = "failed"
	} else if ctx.Err() != nil {
		stopReason = "cancelled"
	} else if overBudget() {
		stopReason = "budget_exhausted"
	}
	lib.WriteReport(e.RepoRoot, session, allSteps, stopReason)

	return &Result{
		Session:  session,
		Steps:    allSteps,
		Outputs:  outputs,
		TotalUSD: totalUSD,
	}, stepErr
}

// runStep executes a single step and records it in the database.
func (e *Engine) runStep(ctx context.Context, step *WfStep, session *lib.Session, input string, outputs map[string]string, opts runners.RunOpts, iterNum *int) (float64, string, error) {
	*iterNum++
	fmt.Printf("--- %s (step %d) ---\n", step.ID, *iterNum)

	prompt := Interpolate(step.Prompt, input, outputs)
	dbStep := &lib.Step{
		SessionID: session.ID,
		Iteration: *iterNum,
		Status:    "running",
		AgentName: e.Runner.Name(),
	}
	e.DB.CreateStep(dbStep)

	result, err := e.Runner.Run(ctx, prompt, opts)
	if err != nil {
		dbStep.Status = "failed"
		dbStep.ChangeSummary = err.Error()
		e.DB.UpdateStep(dbStep)
		return 0, "", fmt.Errorf("step %s failed: %w", step.ID, err)
	}

	dbStep.Status = "kept"
	dbStep.Kept = true
	dbStep.CostUSD = result.CostUSD
	dbStep.ChangeSummary = runners.TruncStr(result.Output, 200)
	e.DB.UpdateStep(dbStep)
	fmt.Printf("  done ($%.4f)\n\n", result.CostUSD)

	return result.CostUSD, result.Output, nil
}

// runLoop executes a step repeatedly until the until-string is found or max iterations reached.
// Returns an error if all iterations fail. Respects budget via overBudget, reports cost via addCost.
func (e *Engine) runLoop(ctx context.Context, step *WfStep, session *lib.Session, input string, outputs map[string]string, opts runners.RunOpts, iterNum *int, overBudget func() bool, addCost func(float64)) (float64, error) {
	var totalCost float64
	maxIter := step.Loop.Max
	if maxIter <= 0 {
		maxIter = 1
	}

	succeeded := false
	consecutiveFailures := 0

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
			AgentName: e.Runner.Name(),
		}
		e.DB.CreateStep(dbStep)

		result, err := e.Runner.Run(ctx, prompt, opts)
		if err != nil {
			dbStep.Status = "failed"
			dbStep.ChangeSummary = err.Error()
			e.DB.UpdateStep(dbStep)
			consecutiveFailures++
			fmt.Printf("  failed, retrying\n\n")
			continue
		}

		consecutiveFailures = 0
		succeeded = true
		totalCost += result.CostUSD
		if addCost != nil {
			addCost(result.CostUSD)
		}
		dbStep.Status = "kept"
		dbStep.Kept = true
		dbStep.CostUSD = result.CostUSD
		dbStep.ChangeSummary = runners.TruncStr(result.Output, 200)
		e.DB.UpdateStep(dbStep)

		outputs[step.ID] = result.Output
		fmt.Printf("  done ($%.4f)\n\n", result.CostUSD)

		// Commit after each loop iteration
		e.Git.CommitAll(fmt.Sprintf("%s: %s iteration %d", session.Workflow, step.ID, attempt))

		// Check until condition
		if step.Loop.Until != "" && strings.Contains(strings.ToUpper(result.Output), strings.ToUpper(step.Loop.Until)) {
			fmt.Printf("  → loop complete (%s found)\n\n", step.Loop.Until)
			return totalCost, nil
		}
	}

	if !succeeded {
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
				AgentName: e.Runner.Name(),
			}

			mu.Lock()
			e.DB.CreateStep(dbStep)
			mu.Unlock()

			result, err := e.Runner.Run(ctx, prompt, opts)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				dbStep.Status = "failed"
				dbStep.ChangeSummary = err.Error()
				e.DB.UpdateStep(dbStep)
				results[j] = parallelResult{id: pid, err: err}
				return
			}

			dbStep.Status = "kept"
			dbStep.Kept = true
			dbStep.CostUSD = result.CostUSD
			dbStep.ChangeSummary = runners.TruncStr(result.Output, 200)
			e.DB.UpdateStep(dbStep)

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
