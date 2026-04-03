package loops

import (
	"context"
	"fmt"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/runners"
)

type RefactorConfig struct {
	Description string
	Target      string
	TestCmd     string
	RepoRoot    string
	BudgetUSD   float64
}

type RefactorResult struct {
	Session *lib.Session
	Steps   []lib.Step
}

func RunRefactor(ctx context.Context, db *lib.DB, runner runners.Runner, git *lib.Git, cfg RefactorConfig) (*RefactorResult, error) {
	session := &lib.Session{
		ID:        lib.NewSessionID(),
		Workflow:  "refactor",
		Target:    cfg.Target,
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

	branchName := fmt.Sprintf("refactor/%s", session.ID)
	if err := git.CreateBranch(branchName); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}
	fmt.Printf("Refactor session %s on branch %s\n\n", session.ID, branchName)

	var spentUSD float64
	opts := runners.RunOpts{
		WorkDir:      cfg.RepoRoot,
		AllowedTools: "Bash,Read,Edit,Write,Grep,Glob",
		MaxTurns:     25,
	}

	// Capture baseline metric
	var baselineMetric lib.MetricResult
	if cfg.TestCmd != "" {
		fmt.Println("Running baseline tests...")
		baselineMetric = lib.RunMetric(ctx, cfg.TestCmd, cfg.RepoRoot)
		fmt.Printf("Baseline: exit %d\n\n", baselineMetric.ExitCode)
	}

	// Step 1: Analyze
	fmt.Println("--- Step 1: Analyze ---")
	analyzeStep := &lib.Step{SessionID: session.ID, Iteration: 1, Status: "running", AgentName: runner.Name()}
	db.CreateStep(analyzeStep)

	analyzeResult, err := runner.Run(ctx, fmt.Sprintf(
		`You are analyzing code for refactoring. Read the target code and identify what to change.

Target: %s
Goal: %s

Report:
1. Current code smells or issues
2. Proposed transformations (ordered by priority)
3. Risk areas (what could break)

Do not make any changes yet — analysis only.`, cfg.Target, cfg.Description), opts)
	if err != nil {
		analyzeStep.Status = "failed"
		analyzeStep.ChangeSummary = err.Error()
		db.UpdateStep(analyzeStep)
		db.UpdateSessionStatus(session.ID, "failed")
		return nil, fmt.Errorf("analyze step failed: %w", err)
	}
	spentUSD += analyzeResult.CostUSD
	analyzeStep.Status = "kept"
	analyzeStep.Kept = true
	analyzeStep.CostUSD = analyzeResult.CostUSD
	analyzeStep.ChangeSummary = truncate(analyzeResult.Output, 200)
	db.UpdateStep(analyzeStep)
	fmt.Printf("  Analysis complete ($%.4f)\n\n", analyzeResult.CostUSD)

	// Step 2: Transform
	fmt.Println("--- Step 2: Transform ---")
	transformStep := &lib.Step{SessionID: session.ID, Iteration: 2, Status: "running", AgentName: runner.Name()}
	db.CreateStep(transformStep)

	transformResult, err := runner.Run(ctx, fmt.Sprintf(
		`You are refactoring code. Apply the transformations from this analysis.

Target: %s
Goal: %s

Analysis:
%s

Apply each transformation. Preserve all existing behavior — no functional changes.`, cfg.Target, cfg.Description, analyzeResult.Output), opts)
	if err != nil {
		git.RevertAll()
		transformStep.Status = "failed"
		transformStep.ChangeSummary = err.Error()
		db.UpdateStep(transformStep)
		db.UpdateSessionStatus(session.ID, "failed")
		return nil, fmt.Errorf("transform step failed: %w", err)
	}
	spentUSD += transformResult.CostUSD
	summary, _ := git.DiffStat()
	transformStep.CostUSD = transformResult.CostUSD
	transformStep.ChangeSummary = summary

	// Step 3: Verify — tests must still pass
	if cfg.TestCmd != "" {
		fmt.Println("--- Step 3: Verify ---")
		verifyMetric := lib.RunMetric(ctx, cfg.TestCmd, cfg.RepoRoot)
		if verifyMetric.ExitCode == 0 {
			git.CommitAll(fmt.Sprintf("refactor(%s): transform", session.ID))
			transformStep.Status = "kept"
			transformStep.Kept = true
			fmt.Println("  Tests still passing — refactor verified")
		} else {
			git.RevertAll()
			transformStep.Status = "reverted"
			transformStep.ChangeSummary = fmt.Sprintf("tests broke after refactor (exit %d) — reverted", verifyMetric.ExitCode)
			fmt.Printf("  Tests broke (exit %d) — refactor reverted\n\n", verifyMetric.ExitCode)
		}
	} else {
		git.CommitAll(fmt.Sprintf("refactor(%s): transform", session.ID))
		transformStep.Status = "kept"
		transformStep.Kept = true
		fmt.Println("  No test command — committed without verification")
	}
	db.UpdateStep(transformStep)

	db.UpdateSessionStatus(session.ID, "done")
	allSteps, _ := db.GetSteps(session.ID)
	lib.WriteReport(cfg.RepoRoot, session, allSteps, "completed")

	return &RefactorResult{Session: session, Steps: allSteps}, nil
}
