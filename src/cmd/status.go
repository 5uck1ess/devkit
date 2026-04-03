package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [session-id]",
	Short: "Show session status",
	Long:  "Show all sessions or details for a specific session.",
	Example: `  devkit status
  devkit status abc123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return showSessionDetail(args[0])
		}
		return showAllSessions()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func showAllSessions() error {
	sessions, err := db.ListSessions()
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}
	if len(sessions) == 0 {
		fmt.Println("No sessions found. Run `devkit improve`, `devkit review`, or `devkit dispatch` to start one.")
		return nil
	}

	fmt.Printf("%-14s %-10s %-10s %-8s %s\n", "SESSION", "WORKFLOW", "STATUS", "COST", "CREATED")
	fmt.Printf("%-14s %-10s %-10s %-8s %s\n", "-------", "--------", "------", "----", "-------")

	for _, s := range sessions {
		cost, err := db.SessionTotalCost(s.ID)
		costStr := fmt.Sprintf("$%.4f", cost)
		if err != nil {
			costStr = "unknown"
		}
		age := formatAge(s.CreatedAt)
		fmt.Printf("%-14s %-10s %-10s %-8s %s\n", s.ID, s.Workflow, s.Status, costStr, age)
	}
	return nil
}

func showSessionDetail(id string) error {
	session, err := db.GetSession(id)
	if err != nil {
		return err
	}

	steps, err := db.GetSteps(id)
	if err != nil {
		return fmt.Errorf("get steps: %w", err)
	}

	totalCost, _ := db.SessionTotalCost(id)

	fmt.Printf("Session:    %s\n", session.ID)
	fmt.Printf("Workflow:   %s\n", session.Workflow)
	fmt.Printf("Status:     %s\n", session.Status)
	fmt.Printf("Created:    %s\n", session.CreatedAt.Format(time.RFC3339))

	if session.Target != "" {
		fmt.Printf("Target:     %s\n", session.Target)
	}
	if session.Metric != "" {
		fmt.Printf("Metric:     %s\n", session.Metric)
	}
	if session.Objective != "" {
		fmt.Printf("Objective:  %s\n", session.Objective)
	}
	if session.MaxIterations > 0 {
		fmt.Printf("Iterations: %d/%d\n", len(steps), session.MaxIterations)
	}
	if session.BudgetUSD > 0 {
		fmt.Printf("Budget:     $%.2f ($%.4f spent)\n", session.BudgetUSD, totalCost)
	}
	fmt.Printf("Total cost: $%.4f\n", totalCost)

	if len(steps) > 0 {
		fmt.Printf("\n%-5s %-10s %-10s %-8s %-8s %s\n", "ITER", "AGENT", "STATUS", "EXIT", "COST", "SUMMARY")
		fmt.Printf("%-5s %-10s %-10s %-8s %-8s %s\n", "----", "-----", "------", "----", "----", "-------")
		for _, s := range steps {
			summary := s.ChangeSummary
			if len(summary) > 50 {
				summary = summary[:50] + "..."
			}
			fmt.Printf("%-5d %-10s %-10s %-8d $%-7.4f %s\n", s.Iteration, s.AgentName, s.Status, s.MetricExitCode, s.CostUSD, summary)
		}
	}

	if session.Status == "paused" || session.Status == "failed" {
		fmt.Printf("\nResume with: devkit resume %s\n", session.ID)
	}
	return nil
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
