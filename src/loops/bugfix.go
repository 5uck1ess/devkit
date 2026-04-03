package loops

import (
	"context"
	"fmt"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/runners"
)

type BugfixConfig struct {
	Description string
	TestCmd     string
	RepoRoot    string
	BudgetUSD   float64
}

type BugfixResult struct {
	Session *lib.Session
	Steps   []lib.Step
}

func RunBugfix(ctx context.Context, db *lib.DB, runner runners.Runner, git *lib.Git, cfg BugfixConfig) (*BugfixResult, error) {
	session := &lib.Session{
		ID:        lib.NewSessionID(),
		Workflow:  "bugfix",
		Metric:    cfg.TestCmd,
		Prompt:    cfg.Description,
		Status:    "running",
		BudgetUSD: cfg.BudgetUSD,
	}
	if err := db.CreateSession(session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if err := lib.EnsureSessionDir(cfg.RepoRoot, session.ID); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}

	branchName := fmt.Sprintf("bugfix/%s", session.ID)
	if err := git.CreateBranch(branchName); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}
	fmt.Printf("Bugfix session %s on branch %s\n\n", session.ID, branchName)

	var spentUSD float64
	opts := runners.RunOpts{
		WorkDir:      cfg.RepoRoot,
		AllowedTools: "Bash,Read,Edit,Write,Grep,Glob",
		MaxTurns:     25,
	}

	// Step 1: Diagnose
	fmt.Println("--- Step 1: Diagnose ---")
	diagStep := &lib.Step{SessionID: session.ID, Iteration: 1, Status: "running", AgentName: runner.Name()}
	db.CreateStep(diagStep)

	diagPrompt := fmt.Sprintf(
		`You are diagnosing a bug. Investigate the codebase to find the root cause.

Bug: %s

Read the relevant code. Identify the exact root cause. Report:
1. Root cause (what's wrong and why)
2. The specific file(s) and line(s)
3. Your proposed fix (describe, don't implement yet)`, cfg.Description)

	diagResult, err := runner.Run(ctx, diagPrompt, opts)
	if err != nil {
		diagStep.Status = "failed"
		diagStep.ChangeSummary = err.Error()
		db.UpdateStep(diagStep)
		db.UpdateSessionStatus(session.ID, "failed")
		return nil, fmt.Errorf("diagnose step failed: %w", err)
	}
	spentUSD += diagResult.CostUSD
	diagStep.Status = "kept"
	diagStep.Kept = true
	diagStep.CostUSD = diagResult.CostUSD
	diagStep.ChangeSummary = truncate(diagResult.Output, 200)
	db.UpdateStep(diagStep)
	fmt.Printf("  Diagnosis complete ($%.4f)\n\n", diagResult.CostUSD)

	// Step 2: Fix
	fmt.Println("--- Step 2: Fix ---")
	fixStep := &lib.Step{SessionID: session.ID, Iteration: 2, Status: "running", AgentName: runner.Name()}
	db.CreateStep(fixStep)

	fixResult, err := runner.Run(ctx, fmt.Sprintf(
		`You are fixing a bug. Apply the fix based on this diagnosis.

Bug: %s

Diagnosis:
%s

Make the minimal change needed to fix the bug. Do not refactor unrelated code.`, cfg.Description, diagResult.Output), opts)
	if err != nil {
		git.RevertAll()
		fixStep.Status = "failed"
		fixStep.ChangeSummary = err.Error()
		db.UpdateStep(fixStep)
		db.UpdateSessionStatus(session.ID, "failed")
		return nil, fmt.Errorf("fix step failed: %w", err)
	}
	spentUSD += fixResult.CostUSD
	summary, _ := git.DiffStat()
	git.CommitAll(fmt.Sprintf("bugfix(%s): fix", session.ID))
	fixStep.Status = "kept"
	fixStep.Kept = true
	fixStep.CostUSD = fixResult.CostUSD
	fixStep.ChangeSummary = summary
	db.UpdateStep(fixStep)
	fmt.Printf("  Fix applied ($%.4f)\n\n", fixResult.CostUSD)

	// Step 3: Verify
	if cfg.TestCmd != "" {
		fmt.Println("--- Step 3: Verify ---")
		testMetric := lib.RunMetric(ctx, cfg.TestCmd, cfg.RepoRoot)
		if testMetric.ExitCode == 0 {
			fmt.Println("  Tests passing — fix verified")
		} else {
			fmt.Println("  Tests still failing, attempting repair...")
			repairStep := &lib.Step{SessionID: session.ID, Iteration: 3, Status: "running", AgentName: runner.Name()}
			db.CreateStep(repairStep)

			repairResult, err := runner.Run(ctx, fmt.Sprintf(
				`The fix was applied but tests are still failing. Adjust the fix.

Bug: %s
Test command: %s
Test output:
%s

Fix the remaining failures.`, cfg.Description, cfg.TestCmd, testMetric.Output), opts)
			if err == nil {
				spentUSD += repairResult.CostUSD
				verifyMetric := lib.RunMetric(ctx, cfg.TestCmd, cfg.RepoRoot)
				if verifyMetric.ExitCode == 0 {
					git.CommitAll(fmt.Sprintf("bugfix(%s): repair", session.ID))
					repairStep.Status = "kept"
					repairStep.Kept = true
					repairStep.CostUSD = repairResult.CostUSD
					fmt.Println("  Repair successful — tests passing")
				} else {
					git.RevertAll()
					repairStep.Status = "reverted"
					repairStep.ChangeSummary = "tests still failing after repair"
					fmt.Println("  Repair failed — reverted")
				}
			} else {
				repairStep.Status = "failed"
				repairStep.ChangeSummary = err.Error()
			}
			db.UpdateStep(repairStep)
		}
	}

	db.UpdateSessionStatus(session.ID, "done")
	allSteps, _ := db.GetSteps(session.ID)
	lib.WriteReport(cfg.RepoRoot, session, allSteps, "completed")

	return &BugfixResult{Session: session, Steps: allSteps}, nil
}
