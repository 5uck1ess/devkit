package loops

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/runners"
)

type ReviewConfig struct {
	Prompt   string
	Security bool
	Agents   []string // filter to specific agents
	RepoRoot string
}

type AgentResult struct {
	Agent  string
	Output string
	Error  error
	Cost   float64
}

type ReviewResult struct {
	Session *lib.Session
	Results []AgentResult
}

func RunReview(ctx context.Context, db *lib.DB, available []runners.Runner, git *lib.Git, cfg ReviewConfig) (*ReviewResult, error) {
	diff, err := git.DiffFromMain()
	if err != nil {
		return nil, fmt.Errorf("get diff: %w", err)
	}
	if diff == "" {
		return nil, fmt.Errorf("no diff found — nothing to review")
	}

	// Truncate diff for prompt
	const maxDiff = 30000
	if len(diff) > maxDiff {
		diff = diff[:maxDiff] + "\n... (diff truncated)"
	}

	prompt := cfg.Prompt
	if prompt == "" {
		prompt = "Review this code diff. For each issue found, report: file and line number, severity (critical/warning/suggestion), description, and suggested fix."
	}
	if cfg.Security {
		prompt += "\n\nFocus specifically on security issues: OWASP top 10, hardcoded secrets, injection vulnerabilities, authentication flaws."
	}

	fullPrompt := fmt.Sprintf("%s\n\n```diff\n%s\n```", prompt, diff)

	// Filter runners
	selected := filterRunners(available, cfg.Agents)
	if len(selected) == 0 {
		return nil, fmt.Errorf("no agents available — need at least claude CLI installed")
	}

	session := &lib.Session{
		ID:       lib.NewSessionID(),
		Workflow: "review",
		Prompt:   prompt,
		Status:   "running",
	}
	if err := db.CreateSession(session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	fmt.Printf("Review session %s — dispatching to %d agent(s)\n", session.ID, len(selected))

	// Dispatch in parallel
	results := make([]AgentResult, len(selected))
	var wg sync.WaitGroup

	for i, r := range selected {
		wg.Add(1)
		go func(idx int, runner runners.Runner) {
			defer wg.Done()
			fmt.Printf("  [%s] running...\n", runner.Name())

			res, err := runner.Run(ctx, fullPrompt, runners.RunOpts{
				WorkDir:  cfg.RepoRoot,
				MaxTurns: 10,
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

	return &ReviewResult{Session: session, Results: results}, nil
}

func filterRunners(available []runners.Runner, names []string) []runners.Runner {
	if len(names) == 0 {
		return available
	}
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[strings.ToLower(n)] = true
	}
	var filtered []runners.Runner
	for _, r := range available {
		if nameSet[r.Name()] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
