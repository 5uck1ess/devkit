package loops

import (
	"context"
	"fmt"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/runners"
)

type FeatureConfig struct {
	Description string
	Target      string
	TestCmd     string
	LintCmd     string
	RepoRoot    string
	BudgetUSD   float64
}

type FeatureResult struct {
	Session *lib.Session
	Steps   []lib.Step
}

func RunFeature(ctx context.Context, db *lib.DB, runner runners.Runner, git *lib.Git, cfg FeatureConfig) (*FeatureResult, error) {
	session := &lib.Session{
		ID:       lib.NewSessionID(),
		Workflow: "feature",
		Target:   cfg.Target,
		Prompt:   cfg.Description,
		Status:   "running",
		BudgetUSD: cfg.BudgetUSD,
	}
	if err := db.CreateSession(session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if err := lib.EnsureSessionDir(cfg.RepoRoot, session.ID); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}

	branchName := fmt.Sprintf("feature/%s", session.ID)
	if err := git.CreateBranch(branchName); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}
	fmt.Printf("Feature session %s on branch %s\n\n", session.ID, branchName)

	var spentUSD float64
	checkBudget := func() bool {
		return cfg.BudgetUSD > 0 && spentUSD >= cfg.BudgetUSD
	}
	opts := runners.RunOpts{
		WorkDir:      cfg.RepoRoot,
		AllowedTools: "Bash,Read,Edit,Write,Grep,Glob",
		MaxTurns:     30,
	}

	// Step 1: Plan
	fmt.Println("--- Step 1: Plan ---")
	planStep := &lib.Step{SessionID: session.ID, Iteration: 1, Status: "running", AgentName: runner.Name()}
	db.CreateStep(planStep)

	planResult, err := runner.Run(ctx, fmt.Sprintf(
		`You are planning a feature. Think through the design, then produce a numbered implementation plan.
Each item should be a single, testable change. Order by dependency.

Feature: %s
Target: %s

Output ONLY the plan as a numbered list. Do not write any code yet.`, cfg.Description, cfg.Target), opts)
	if err != nil {
		planStep.Status = "failed"
		planStep.ChangeSummary = err.Error()
		db.UpdateStep(planStep)
		db.UpdateSessionStatus(session.ID, "failed")
		return nil, fmt.Errorf("plan step failed: %w", err)
	}
	spentUSD += planResult.CostUSD
	planStep.Status = "kept"
	planStep.Kept = true
	planStep.CostUSD = planResult.CostUSD
	planStep.ChangeSummary = truncate(planResult.Output, 200)
	db.UpdateStep(planStep)
	fmt.Printf("  Plan complete ($%.4f)\n\n", planResult.CostUSD)

	if checkBudget() {
		fmt.Printf("  Budget exhausted ($%.2f of $%.2f) — stopping after plan\n", spentUSD, cfg.BudgetUSD)
		db.UpdateSessionStatus(session.ID, "failed")
		allSteps, _ := db.GetSteps(session.ID)
		return &FeatureResult{Session: session, Steps: allSteps}, nil
	}

	// Step 2: Implement
	fmt.Println("--- Step 2: Implement ---")
	implStep := &lib.Step{SessionID: session.ID, Iteration: 2, Status: "running", AgentName: runner.Name()}
	db.CreateStep(implStep)

	implResult, err := runner.Run(ctx, fmt.Sprintf(
		`You are implementing a feature. Follow this plan exactly, implementing each item in order.

Feature: %s
Target: %s

Plan:
%s

Write the code. Make all necessary changes. Do not skip any plan items.`, cfg.Description, cfg.Target, planResult.Output), opts)
	if err != nil {
		git.RevertAll()
		implStep.Status = "failed"
		implStep.ChangeSummary = err.Error()
		db.UpdateStep(implStep)
		db.UpdateSessionStatus(session.ID, "failed")
		return nil, fmt.Errorf("implement step failed: %w", err)
	}
	spentUSD += implResult.CostUSD
	implStep.CostUSD = implResult.CostUSD
	fmt.Printf("  Implemented ($%.4f)\n\n", implResult.CostUSD)

	// Step 3: Test — verify BEFORE committing
	testsPass := cfg.TestCmd == ""
	if cfg.TestCmd != "" && !checkBudget() {
		fmt.Println("--- Step 3: Test ---")
		for attempt := 1; attempt <= 3; attempt++ {
			if ctx.Err() != nil {
				break
			}
			testMetric := lib.RunMetric(ctx, cfg.TestCmd, cfg.RepoRoot)
			if testMetric.ExitCode == 0 {
				fmt.Printf("  Tests passing (attempt %d)\n\n", attempt)
				testsPass = true
				break
			}

			fmt.Printf("  Tests failing (attempt %d), fixing...\n", attempt)
			testStep := &lib.Step{SessionID: session.ID, Iteration: 2 + attempt, Status: "running", AgentName: runner.Name()}
			db.CreateStep(testStep)

			fixResult, err := runner.Run(ctx, fmt.Sprintf(
				`Tests are failing. Fix the failures without changing test expectations.

Test command: %s
Test output:
%s

Fix the code so tests pass.`, cfg.TestCmd, testMetric.Output), opts)
			if err != nil {
				testStep.Status = "failed"
				testStep.ChangeSummary = err.Error()
				db.UpdateStep(testStep)
				continue
			}
			spentUSD += fixResult.CostUSD
			git.CommitAll(fmt.Sprintf("feature(%s): fix tests attempt %d", session.ID, attempt))
			testStep.Status = "kept"
			testStep.Kept = true
			testStep.CostUSD = fixResult.CostUSD
			db.UpdateStep(testStep)
		}
	}

	// Commit implementation only after tests pass (or no test command)
	if testsPass {
		summary, _ := git.DiffStat()
		git.CommitAll(fmt.Sprintf("feature(%s): implement", session.ID))
		implStep.Status = "kept"
		implStep.Kept = true
		implStep.ChangeSummary = summary
	} else {
		git.RevertAll()
		implStep.Status = "reverted"
		implStep.ChangeSummary = "tests never passed — reverted"
		fmt.Println("  Tests never passed — implementation reverted")
	}
	db.UpdateStep(implStep)

	// Step 4: Lint (if lint command provided)
	if cfg.LintCmd != "" && !checkBudget() {
		fmt.Println("--- Step 4: Lint ---")
		lintMetric := lib.RunMetric(ctx, cfg.LintCmd, cfg.RepoRoot)
		if lintMetric.ExitCode != 0 {
			nextIter, _ := db.LastIteration(session.ID)
			lintStep := &lib.Step{SessionID: session.ID, Iteration: nextIter + 1, Status: "running", AgentName: runner.Name()}
			db.CreateStep(lintStep)

			lintResult, err := runner.Run(ctx, fmt.Sprintf(
				`Fix these lint errors without changing code behavior.

Lint command: %s
Lint output:
%s`, cfg.LintCmd, lintMetric.Output), opts)
			if err == nil {
				spentUSD += lintResult.CostUSD
				git.CommitAll(fmt.Sprintf("feature(%s): fix lint", session.ID))
				lintStep.Status = "kept"
				lintStep.Kept = true
				lintStep.CostUSD = lintResult.CostUSD
			} else {
				lintStep.Status = "failed"
				lintStep.ChangeSummary = err.Error()
			}
			db.UpdateStep(lintStep)
		} else {
			fmt.Println("  Lint clean")
		}
	}

	db.UpdateSessionStatus(session.ID, "done")
	allSteps, _ := db.GetSteps(session.ID)
	lib.WriteReport(cfg.RepoRoot, session, allSteps, "completed")

	return &FeatureResult{Session: session, Steps: allSteps}, nil
}
