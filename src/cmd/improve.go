package cmd

import (
	"fmt"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/loops"
	"github.com/spf13/cobra"
)

var improveCmd = &cobra.Command{
	Use:   "improve",
	Short: "Run a metric-gated improvement loop",
	Long:  "Spawns an AI agent per iteration. Each iteration: propose change, run metric, keep if pass, revert if fail.",
	Example: `  devkit improve --target src/ --metric "npm test" --objective "0 failing tests" --iterations 20
  devkit improve --metric "go test ./..." --iterations 10 --budget 5.00`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		metric, _ := cmd.Flags().GetString("metric")
		objective, _ := cmd.Flags().GetString("objective")
		iterations, _ := cmd.Flags().GetInt("iterations")
		budget, _ := cmd.Flags().GetFloat64("budget")

		if metric == "" {
			return fmt.Errorf("--metric is required — provide a command that exits 0 on success (e.g., \"npm test\")")
		}
		if iterations < 1 {
			return fmt.Errorf("--iterations must be at least 1")
		}
		if target == "" {
			target = "."
		}
		if objective == "" {
			objective = "improve the metric to pass"
		}

		dirty, err := (&lib.Git{Dir: repoRoot}).HasUncommittedChanges()
		if err != nil {
			return fmt.Errorf("check git status: %w", err)
		}
		if dirty {
			return fmt.Errorf("working tree has uncommitted changes — commit or stash before running devkit improve")
		}

		agentName, _ := cmd.Flags().GetString("agent")
		runner, err := resolveRunner(agentName)
		if err != nil {
			return err
		}

		git := &lib.Git{Dir: repoRoot}
		cfg := loops.ImproveConfig{
			Target:        target,
			Metric:        metric,
			Objective:     objective,
			MaxIterations: iterations,
			BudgetUSD:     budget,
			MaxFailures:   3,
			RepoRoot:      repoRoot,
		}

		result, err := loops.RunImproveLoop(cmd.Context(), db, runner, git, cfg)
		if err != nil {
			return err
		}

		printImproveResult(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(improveCmd)
	improveCmd.Flags().String("target", ".", "Directory or file to improve")
	improveCmd.Flags().String("metric", "", "Command that exits 0 on success (required)")
	improveCmd.Flags().String("objective", "", "What the improvement should achieve")
	improveCmd.Flags().Int("iterations", 10, "Maximum number of iterations")
	improveCmd.Flags().Float64("budget", 0, "Maximum spend in USD (0 = unlimited)")
}

func printImproveResult(r *loops.ImproveResult) {
	kept := 0
	reverted := 0
	var totalCost float64
	for _, s := range r.Steps {
		totalCost += s.CostUSD
		if s.Kept {
			kept++
		} else {
			reverted++
		}
	}

	fmt.Printf("\n=== Improve Session Complete ===\n")
	fmt.Printf("Session:    %s\n", r.Session.ID)
	fmt.Printf("Status:     %s\n", r.StopReason)
	fmt.Printf("Iterations: %d (%d kept, %d reverted)\n", len(r.Steps), kept, reverted)
	fmt.Printf("Total cost: $%.4f\n", totalCost)
	fmt.Printf("\nRun `devkit status %s` for full details.\n", r.Session.ID)
}
