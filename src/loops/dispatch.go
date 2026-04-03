package loops

import (
	"context"
	"fmt"
	"sync"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/runners"
)

type DispatchConfig struct {
	Prompt   string
	Agents   []string
	RepoRoot string
}

type DispatchResult struct {
	Session *lib.Session
	Results []AgentResult
}

func RunDispatch(ctx context.Context, db *lib.DB, available []runners.Runner, cfg DispatchConfig) (*DispatchResult, error) {
	selected := filterRunners(available, cfg.Agents)
	if len(selected) == 0 {
		return nil, fmt.Errorf("no agents available — need at least claude CLI installed")
	}

	session := &lib.Session{
		ID:       lib.NewSessionID(),
		Workflow: "dispatch",
		Prompt:   cfg.Prompt,
		Status:   "running",
	}
	if err := db.CreateSession(session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	fmt.Printf("Dispatch session %s — sending to %d agent(s)\n", session.ID, len(selected))

	results := make([]AgentResult, len(selected))
	var wg sync.WaitGroup

	for i, r := range selected {
		wg.Add(1)
		go func(idx int, runner runners.Runner) {
			defer wg.Done()
			fmt.Printf("  [%s] running...\n", runner.Name())

			res, err := runner.Run(ctx, cfg.Prompt, runners.RunOpts{
				WorkDir:  cfg.RepoRoot,
				MaxTurns: 15,
			})
			results[idx] = AgentResult{
				Agent:  runner.Name(),
				Output: res.Output,
				Error:  err,
				Cost:   res.CostUSD,
			}

			step := &lib.Step{
				SessionID:     session.ID,
				Iteration:     idx + 1,
				AgentName:     runner.Name(),
				Status:        "done",
				ChangeSummary: truncate(res.Output, 200),
				CostUSD:       res.CostUSD,
			}
			if err != nil {
				step.Status = "failed"
				step.ChangeSummary = err.Error()
			}
			db.CreateStep(step)
			db.UpdateStep(step)

			fmt.Printf("  [%s] done ($%.4f)\n", runner.Name(), res.CostUSD)
		}(i, r)
	}
	wg.Wait()

	db.UpdateSessionStatus(session.ID, "done")

	return &DispatchResult{Session: session, Results: results}, nil
}
