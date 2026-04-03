package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteReport(repoRoot string, session *Session, steps []Step, stopReason string) error {
	dir := SessionDir(repoRoot, session.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var totalCost float64
	kept := 0
	reverted := 0
	for _, s := range steps {
		totalCost += s.CostUSD
		if s.Kept {
			kept++
		} else {
			reverted++
		}
	}

	var b strings.Builder
	workflow := session.Workflow
	if len(workflow) > 0 {
		workflow = strings.ToUpper(workflow[:1]) + workflow[1:]
	}
	fmt.Fprintf(&b, "# Devkit %s Report\n\n", workflow)
	fmt.Fprintf(&b, "**Session:** %s\n", session.ID)
	fmt.Fprintf(&b, "**Status:** %s\n", stopReason)
	fmt.Fprintf(&b, "**Iterations:** %d (%d kept, %d reverted)\n", len(steps), kept, reverted)
	fmt.Fprintf(&b, "**Total cost:** $%.4f\n\n", totalCost)

	if session.Target != "" {
		fmt.Fprintf(&b, "**Target:** %s\n", session.Target)
	}
	if session.Metric != "" {
		fmt.Fprintf(&b, "**Metric:** `%s`\n", session.Metric)
	}
	if session.Objective != "" {
		fmt.Fprintf(&b, "**Objective:** %s\n", session.Objective)
	}

	if len(steps) > 0 {
		b.WriteString("\n## Iteration Log\n\n")
		b.WriteString("| # | Agent | Status | Exit | Cost | Summary |\n")
		b.WriteString("|---|-------|--------|------|------|---------|\n")
		for _, s := range steps {
			summary := s.ChangeSummary
			if len(summary) > 60 {
				summary = summary[:60] + "..."
			}
			fmt.Fprintf(&b, "| %d | %s | %s | %d | $%.4f | %s |\n",
				s.Iteration, s.AgentName, s.Status, s.MetricExitCode, s.CostUSD, summary)
		}
	}

	path := filepath.Join(dir, "report.md")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
