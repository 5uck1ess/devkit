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
	os.MkdirAll(scratchDir, 0o755)

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

	// Walk steps sequentially, with branch jumps
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

		// Skip steps that are only referenced by parallel dispatchers
		// (they're executed inline when the parallel step runs)
		if step.Prompt == "" && len(step.Parallel) > 0 {
			cost, err := e.runParallel(ctx, step, wf.Steps, stepIndex, session, cfg.Input, outputs, opts, &iterNum)
			if err != nil {
				e.DB.UpdateSessionStatus(session.ID, "failed")
				break
			}
			totalUSD += cost
			i++
			continue
		}

		// Regular step
		if step.Prompt == "" {
			i++
			continue
		}

		if step.Loop != nil {
			cost, err := e.runLoop(ctx, step, session, cfg.Input, outputs, opts, &iterNum)
			if err != nil {
				e.DB.UpdateSessionStatus(session.ID, "failed")
				break
			}
			totalUSD += cost
		} else {
			cost, output, err := e.runStep(ctx, step, session, cfg.Input, outputs, opts, &iterNum)
			if err != nil {
				e.DB.UpdateSessionStatus(session.ID, "failed")
				break
			}
			totalUSD += cost
			outputs[step.ID] = output

			// Evaluate branch
			if len(step.Branch) > 0 {
				if target := EvalBranch(output, step.Branch); target != "" {
					if idx, ok := stepIndex[target]; ok {
						fmt.Printf("  → branching to %s\n\n", target)
						i = idx
						continue
					}
				}
			}
		}

		i++
	}

	// Clean up scratchpad
	os.Remove(filepath.Join(scratchDir, "current.md"))

	// Commit any remaining changes
	e.Git.CommitAll(fmt.Sprintf("%s(%s): complete", session.Workflow, session.ID))

	e.DB.UpdateSessionStatus(session.ID, "done")
	allSteps, _ := e.DB.GetSteps(session.ID)
	lib.WriteReport(e.RepoRoot, session, allSteps, "completed")

	return &Result{
		Session:  session,
		Steps:    allSteps,
		Outputs:  outputs,
		TotalUSD: totalUSD,
	}, nil
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
func (e *Engine) runLoop(ctx context.Context, step *WfStep, session *lib.Session, input string, outputs map[string]string, opts runners.RunOpts, iterNum *int) (float64, error) {
	var totalCost float64
	maxIter := step.Loop.Max
	if maxIter <= 0 {
		maxIter = 1
	}

	for attempt := 1; attempt <= maxIter; attempt++ {
		if ctx.Err() != nil {
			return totalCost, ctx.Err()
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
			// Loop continues on failure — try again
			totalCost += result.CostUSD
			fmt.Printf("  failed, retrying\n\n")
			continue
		}

		totalCost += result.CostUSD
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
			break
		}
	}

	return totalCost, nil
}

// runParallel dispatches multiple steps concurrently.
func (e *Engine) runParallel(ctx context.Context, dispatcher *WfStep, allSteps []WfStep, stepIndex map[string]int, session *lib.Session, input string, outputs map[string]string, opts runners.RunOpts, iterNum *int) (float64, error) {
	fmt.Printf("--- %s (parallel: %s) ---\n\n", dispatcher.ID, strings.Join(dispatcher.Parallel, ", "))

	type parallelResult struct {
		id     string
		output string
		cost   float64
		err    error
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

			prompt := Interpolate(step.Prompt, input, outputs)
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
	for _, r := range results {
		if r.err != nil {
			fmt.Printf("  %s: failed (%v)\n", r.id, r.err)
			continue
		}
		outputs[r.id] = r.output
		totalCost += r.cost
		fmt.Printf("  %s: done ($%.4f)\n", r.id, r.cost)
	}
	fmt.Println()

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
