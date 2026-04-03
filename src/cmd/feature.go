package cmd

import (
	"fmt"
	"strings"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/loops"
	"github.com/spf13/cobra"
)

var featureCmd = &cobra.Command{
	Use:   "feature [description]",
	Short: "Full feature lifecycle: plan, implement, test, lint",
	Long:  "Spawns Claude for each step: plan → implement → test (loop until green) → lint.",
	Example: `  devkit feature "add JWT authentication" --target src/auth/
  devkit feature "add search endpoint" --test "npm test" --lint "npm run lint"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		testCmd, _ := cmd.Flags().GetString("test")
		lintCmd, _ := cmd.Flags().GetString("lint")
		budget, _ := cmd.Flags().GetFloat64("budget")

		dirty, err := (&lib.Git{Dir: repoRoot}).HasUncommittedChanges()
		if err != nil {
			return fmt.Errorf("check git status: %w", err)
		}
		if dirty {
			return fmt.Errorf("working tree has uncommitted changes — commit or stash first")
		}

		agentName, _ := cmd.Flags().GetString("agent")
		runner, err := resolveRunner(agentName)
		if err != nil {
			return err
		}

		result, err := loops.RunFeature(cmd.Context(), db, runner, &lib.Git{Dir: repoRoot}, loops.FeatureConfig{
			Description: strings.Join(args, " "),
			Target:      target,
			TestCmd:     testCmd,
			LintCmd:     lintCmd,
			RepoRoot:    repoRoot,
			BudgetUSD:   budget,
		})
		if err != nil {
			return err
		}

		printFeatureResult(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(featureCmd)
	featureCmd.Flags().String("target", ".", "Directory or file to modify")
	featureCmd.Flags().String("test", "", "Test command (runs after implementation)")
	featureCmd.Flags().String("lint", "", "Lint command (runs after tests)")
	featureCmd.Flags().Float64("budget", 0, "Maximum spend in USD (0 = unlimited)")
	featureCmd.Flags().String("agent", "claude", "AI agent to use (claude, codex, gemini)")
}

func printFeatureResult(r *loops.FeatureResult) {
	var totalCost float64
	for _, s := range r.Steps {
		totalCost += s.CostUSD
	}
	fmt.Printf("\n=== Feature Complete ===\n")
	fmt.Printf("Session: %s\n", r.Session.ID)
	fmt.Printf("Steps:   %d\n", len(r.Steps))
	fmt.Printf("Cost:    $%.4f\n", totalCost)
	fmt.Printf("\nRun `devkit status %s` for details.\n", r.Session.ID)
}
