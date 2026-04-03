package loops

import (
	"context"
	"fmt"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/runners"
)

type TestGenConfig struct {
	Target   string
	TestCmd  string
	RepoRoot string
	BudgetUSD float64
}

type TestGenResult struct {
	Session *lib.Session
	Steps   []lib.Step
}

func RunTestGen(ctx context.Context, db *lib.DB, runner runners.Runner, git *lib.Git, cfg TestGenConfig) (*TestGenResult, error) {
	session := &lib.Session{
		ID:        lib.NewSessionID(),
		Workflow:  "test-gen",
		Target:    cfg.Target,
		Metric:    cfg.TestCmd,
		Status:    "running",
		BudgetUSD: cfg.BudgetUSD,
	}
	if err := db.CreateSession(session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if err := lib.EnsureSessionDir(cfg.RepoRoot, session.ID); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}

	branchName := fmt.Sprintf("test-gen/%s", session.ID)
	if err := git.CreateBranch(branchName); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}
	fmt.Printf("Test-gen session %s on branch %s\n\n", session.ID, branchName)

	var spentUSD float64
	checkBudget := func() bool {
		return cfg.BudgetUSD > 0 && spentUSD >= cfg.BudgetUSD
	}
	opts := runners.RunOpts{
		WorkDir:      cfg.RepoRoot,
		AllowedTools: "Bash,Read,Edit,Write,Grep,Glob",
		MaxTurns:     30,
	}

	// Step 1: Analyze target and generate tests
	fmt.Println("--- Step 1: Generate Tests ---")
	genStep := &lib.Step{SessionID: session.ID, Iteration: 1, Status: "running", AgentName: runner.Name()}
	db.CreateStep(genStep)

	genResult, err := runner.Run(ctx, fmt.Sprintf(
		`Analyze the code at %s and generate a comprehensive test suite.

1. Detect the language, test framework, and existing test patterns.
2. Identify all public functions, methods, and API endpoints.
3. Write tests covering: happy paths, edge cases, error conditions, boundary values.
4. Use the project's existing test framework and conventions.
5. Place tests in the project's standard test location.

Write actual test code — no placeholders or TODOs.`, cfg.Target), opts)
	if err != nil {
		genStep.Status = "failed"
		genStep.ChangeSummary = err.Error()
		db.UpdateStep(genStep)
		db.UpdateSessionStatus(session.ID, "failed")
		return nil, fmt.Errorf("generate step failed: %w", err)
	}
	spentUSD += genResult.CostUSD
	if err := git.CommitAll(fmt.Sprintf("test-gen(%s): generate tests", session.ID)); err != nil {
		fmt.Printf("  Warning: commit failed: %s\n", err)
	}
	genStep.Status = "kept"
	genStep.Kept = true
	genStep.CostUSD = genResult.CostUSD
	genStep.ChangeSummary = truncate(genResult.Output, 200)
	db.UpdateStep(genStep)
	fmt.Printf("  Tests generated ($%.4f)\n\n", genResult.CostUSD)

	// Step 2: Run tests and fix failures (up to 5 attempts)
	testsPass := cfg.TestCmd == ""
	if cfg.TestCmd != "" && !checkBudget() {
		fmt.Println("--- Step 2: Run & Fix ---")
		for attempt := 1; attempt <= 5; attempt++ {
			if ctx.Err() != nil || checkBudget() {
				break
			}
			testMetric := lib.RunMetric(ctx, cfg.TestCmd, cfg.RepoRoot)
			if testMetric.ExitCode == 0 {
				fmt.Printf("  All tests passing (attempt %d)\n", attempt)
				testsPass = true
				break
			}

			fmt.Printf("  Tests failing (attempt %d), fixing...\n", attempt)
			fixStep := &lib.Step{SessionID: session.ID, Iteration: 1 + attempt, Status: "running", AgentName: runner.Name()}
			db.CreateStep(fixStep)

			fixResult, err := runner.Run(ctx, fmt.Sprintf(
				`The generated tests are failing. Fix them so they pass.
Determine if the bug is in the test or the implementation.
If the test expectation is wrong, fix the test. If the code has a bug, fix the code.

Test command: %s
Test output:
%s`, cfg.TestCmd, testMetric.Output), opts)
			if err != nil {
				fixStep.Status = "failed"
				fixStep.ChangeSummary = err.Error()
				db.UpdateStep(fixStep)
				continue
			}
			spentUSD += fixResult.CostUSD
			if err := git.CommitAll(fmt.Sprintf("test-gen(%s): fix tests attempt %d", session.ID, attempt)); err != nil {
				fmt.Printf("  Warning: commit failed: %s\n", err)
			}
			fixStep.Status = "kept"
			fixStep.Kept = true
			fixStep.CostUSD = fixResult.CostUSD
			db.UpdateStep(fixStep)
		}
	}

	status := "done"
	if !testsPass {
		status = "failed"
	}
	db.UpdateSessionStatus(session.ID, status)
	allSteps, err := db.GetSteps(session.ID)
	if err != nil {
		fmt.Printf("  Warning: failed to get steps for report: %s\n", err)
	}
	lib.WriteReport(cfg.RepoRoot, session, allSteps, status)

	return &TestGenResult{Session: session, Steps: allSteps}, nil
}
