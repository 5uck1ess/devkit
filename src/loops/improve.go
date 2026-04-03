package loops

import (
	"context"
	"fmt"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/runners"
)

type ImproveConfig struct {
	Target        string
	Metric        string
	Objective     string
	MaxIterations int
	BudgetUSD     float64
	MaxFailures   int
	RepoRoot      string
}

type ImproveResult struct {
	Session    *lib.Session
	Steps      []lib.Step
	Baseline   lib.MetricResult
	StopReason string
}

func RunImproveLoop(ctx context.Context, db *lib.DB, runner runners.Runner, git *lib.Git, cfg ImproveConfig) (*ImproveResult, error) {
	session := &lib.Session{
		ID:            lib.NewSessionID(),
		Workflow:      "improve",
		Target:        cfg.Target,
		Metric:        cfg.Metric,
		Objective:     cfg.Objective,
		MaxIterations: cfg.MaxIterations,
		BudgetUSD:     cfg.BudgetUSD,
		Status:        "running",
	}
	if err := db.CreateSession(session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if err := lib.EnsureSessionDir(cfg.RepoRoot, session.ID); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}

	branchName := fmt.Sprintf("self-improve/%s", session.ID)
	if err := git.CreateBranch(branchName); err != nil {
		return nil, fmt.Errorf("create branch %s: %w", branchName, err)
	}

	fmt.Printf("Session %s started on branch %s\n", session.ID, branchName)
	fmt.Printf("Metric: %s\n", cfg.Metric)
	fmt.Printf("Running baseline...\n")

	baseline := lib.RunMetric(ctx, cfg.Metric, cfg.RepoRoot)
	fmt.Printf("Baseline: exit %d (%s)\n\n", baseline.ExitCode, baseline.Duration)

	return runIterations(ctx, db, runner, git, cfg, session, baseline, 1)
}

func ResumeImproveLoop(ctx context.Context, db *lib.DB, runner runners.Runner, git *lib.Git, session *lib.Session, repoRoot string) (*ImproveResult, error) {
	lastIter, err := db.LastIteration(session.ID)
	if err != nil {
		return nil, fmt.Errorf("get last iteration: %w", err)
	}

	cfg := ImproveConfig{
		Target:        session.Target,
		Metric:        session.Metric,
		Objective:     session.Objective,
		MaxIterations: session.MaxIterations,
		BudgetUSD:     session.BudgetUSD,
		MaxFailures:   3,
		RepoRoot:      repoRoot,
	}

	fmt.Printf("Resuming session %s from iteration %d\n", session.ID, lastIter+1)

	baseline := lib.RunMetric(ctx, cfg.Metric, repoRoot)
	if err := db.UpdateSessionStatus(session.ID, "running"); err != nil {
		return nil, err
	}

	return runIterations(ctx, db, runner, git, cfg, session, baseline, lastIter+1)
}

func runIterations(ctx context.Context, db *lib.DB, runner runners.Runner, git *lib.Git, cfg ImproveConfig, session *lib.Session, baseline lib.MetricResult, startIter int) (*ImproveResult, error) {
	if cfg.MaxFailures == 0 {
		cfg.MaxFailures = 3
	}

	const similarityThreshold = 0.90
	const maxSimilarOutputs = 2

	var spentUSD float64
	if startIter > 1 {
		spent, _ := db.SessionTotalCost(session.ID)
		spentUSD = spent
	}

	consecutiveFailures := 0
	consecutiveSimilar := 0
	lastMetricOutput := ""
	stopReason := "completed"

	for i := startIter; i <= cfg.MaxIterations; i++ {
		if ctx.Err() != nil {
			stopReason = "interrupted"
			break
		}
		if cfg.BudgetUSD > 0 && spentUSD >= cfg.BudgetUSD {
			stopReason = fmt.Sprintf("budget exhausted ($%.2f of $%.2f)", spentUSD, cfg.BudgetUSD)
			break
		}
		if consecutiveFailures >= cfg.MaxFailures {
			stopReason = fmt.Sprintf("stuck — %d consecutive failures", consecutiveFailures)
			break
		}
		if consecutiveSimilar >= maxSimilarOutputs {
			stopReason = fmt.Sprintf("stuck — %d consecutive similar outputs (>%.0f%% match), agent is repeating itself", consecutiveSimilar, similarityThreshold*100)
			break
		}

		steps, _ := db.GetSteps(session.ID)
		if err := lib.WriteHandoff(cfg.RepoRoot, session, steps, baseline); err != nil {
			return nil, fmt.Errorf("write handoff: %w", err)
		}

		step := &lib.Step{
			SessionID: session.ID,
			Iteration: i,
			Status:    "running",
			AgentName: runner.Name(),
		}
		if err := db.CreateStep(step); err != nil {
			return nil, fmt.Errorf("create step: %w", err)
		}

		fmt.Printf("--- Iteration %d/%d ---\n", i, cfg.MaxIterations)

		prompt := buildImprovePrompt(cfg)
		result, err := runner.Run(ctx, prompt, runners.RunOpts{
			WorkDir:                cfg.RepoRoot,
			AllowedTools:           "Bash,Read,Edit,Write,Grep,Glob",
			AppendSystemPromptFile: lib.HandoffPath(cfg.RepoRoot, session.ID),
			MaxTurns:               25,
		})
		if err != nil {
			// Revert any partial changes the agent made before failing
			if revertErr := git.RevertAll(); revertErr != nil {
				fmt.Printf("  Warning: revert after agent error failed: %s\n", revertErr)
			}
			step.Status = "failed"
			step.ChangeSummary = err.Error()
			db.UpdateStep(step)
			consecutiveFailures++
			fmt.Printf("  Agent error: %s\n", err)
			continue
		}

		spentUSD += result.CostUSD
		step.TokensUsed = result.TokensIn + result.TokensOut
		step.CostUSD = result.CostUSD

		metricResult := lib.RunMetric(ctx, cfg.Metric, cfg.RepoRoot)
		step.MetricOutput = metricResult.Output
		step.MetricExitCode = metricResult.ExitCode

		if metricResult.ExitCode == 0 {
			summary, _ := git.DiffStat()
			if err := git.CommitAll(fmt.Sprintf("self-improve: iteration %d — passed", i)); err != nil {
				fmt.Printf("  Warning: commit failed: %s\n", err)
			}
			step.Status = "kept"
			step.Kept = true
			step.ChangeSummary = summary
			consecutiveFailures = 0
			consecutiveSimilar = 0
			lastMetricOutput = metricResult.Output
			fmt.Printf("  KEPT (exit 0) — $%.4f\n", result.CostUSD)
		} else {
			if revertErr := git.RevertAll(); revertErr != nil {
				fmt.Printf("  Warning: revert failed: %s\n", revertErr)
			}
			step.Status = "reverted"
			step.Kept = false
			step.ChangeSummary = fmt.Sprintf("metric exit %d", metricResult.ExitCode)
			consecutiveFailures++

			// Detect Groundhog Day: agent keeps producing near-identical failing output
			if lastMetricOutput != "" && lib.Similarity(lastMetricOutput, metricResult.Output) >= similarityThreshold {
				consecutiveSimilar++
				fmt.Printf("  REVERTED (exit %d, similar output %d/%d) — $%.4f\n", metricResult.ExitCode, consecutiveSimilar, maxSimilarOutputs, result.CostUSD)
			} else {
				consecutiveSimilar = 0
				fmt.Printf("  REVERTED (exit %d) — $%.4f\n", metricResult.ExitCode, result.CostUSD)
			}
			lastMetricOutput = metricResult.Output
		}

		db.UpdateStep(step)
	}

	status := "done"
	if stopReason == "interrupted" {
		status = "paused"
	} else if stopReason != "completed" {
		status = "failed"
	}
	db.UpdateSessionStatus(session.ID, status)

	allSteps, _ := db.GetSteps(session.ID)
	lib.WriteReport(cfg.RepoRoot, session, allSteps, stopReason)

	return &ImproveResult{
		Session:    session,
		Steps:      allSteps,
		Baseline:   baseline,
		StopReason: stopReason,
	}, nil
}

func buildImprovePrompt(cfg ImproveConfig) string {
	return fmt.Sprintf(
		`You are an AI code improver. Your task:

Target: %s
Objective: %s
Metric command: %s

Make ONE focused change that moves toward the objective. Do not make multiple unrelated changes.
Read the handoff file in your system prompt for iteration history and what to avoid.
After making your change, run the metric command to verify it passes.`,
		cfg.Target, cfg.Objective, cfg.Metric,
	)
}
