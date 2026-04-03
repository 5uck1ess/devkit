package lib

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func NewSessionID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func SessionDir(repoRoot, sessionID string) string {
	return filepath.Join(repoRoot, ".devkit", "sessions", sessionID)
}

func EnsureSessionDir(repoRoot, sessionID string) error {
	dir := SessionDir(repoRoot, sessionID)
	return os.MkdirAll(dir, 0o755)
}

func HandoffPath(repoRoot, sessionID string) string {
	return filepath.Join(SessionDir(repoRoot, sessionID), "handoff.md")
}

func WriteHandoff(repoRoot string, session *Session, steps []Step, baseline MetricResult) error {
	dir := SessionDir(repoRoot, session.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var totalCost float64
	var consecutiveFailures int
	for i := len(steps) - 1; i >= 0; i-- {
		totalCost += steps[i].CostUSD
		if !steps[i].Kept {
			consecutiveFailures++
		} else {
			break
		}
	}

	remaining := session.BudgetUSD - totalCost
	lastIter := len(steps)

	var history strings.Builder
	history.WriteString("| # | Status | Metric Exit | Summary |\n")
	history.WriteString("|---|--------|-------------|----------|\n")
	for _, s := range steps {
		status := "kept"
		if !s.Kept {
			status = "reverted"
		}
		summary := s.ChangeSummary
		if len(summary) > 60 {
			summary = summary[:60] + "..."
		}
		fmt.Fprintf(&history, "| %d | %s | %d | %s |\n", s.Iteration, status, s.MetricExitCode, summary)
	}

	lastMetric := "N/A"
	if len(steps) > 0 {
		last := steps[len(steps)-1]
		output := last.MetricOutput
		if len(output) > 200 {
			output = output[:200] + "..."
		}
		lastMetric = fmt.Sprintf("exit %d — %s", last.MetricExitCode, output)
	} else {
		output := baseline.Output
		if len(output) > 200 {
			output = output[:200] + "..."
		}
		lastMetric = fmt.Sprintf("exit %d — %s", baseline.ExitCode, output)
	}

	content := fmt.Sprintf(`# Devkit Improve — Session Handoff
Session: %s
Iteration: %d of %d
Target: %s
Objective: %s
Last metric: %s
Consecutive failures: %d
Remaining budget: $%.2f

## Iteration History
%s
## Instructions
You are iteration %d. Make ONE focused change to %s that moves toward: %s.
Do not repeat approaches from reverted iterations above.
Run %s to verify your change before finishing.
`,
		session.ID,
		lastIter+1, session.MaxIterations,
		session.Target,
		session.Objective,
		lastMetric,
		consecutiveFailures,
		remaining,
		history.String(),
		lastIter+1, session.Target, session.Objective,
		"`"+session.Metric+"`",
	)

	return os.WriteFile(HandoffPath(repoRoot, session.ID), []byte(content), 0o644)
}
