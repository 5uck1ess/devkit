package cmd

import (
	"fmt"
	"strings"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/loops"
	"github.com/5uck1ess/devkit/runners"
	"github.com/spf13/cobra"
)

var bugfixCmd = &cobra.Command{
	Use:   "bugfix [description]",
	Short: "Full bugfix lifecycle: diagnose, fix, verify",
	Long:  "Spawns Claude for each step: diagnose root cause → apply fix → run tests to verify.",
	Example: `  devkit bugfix "login returns 500 when email has a plus sign"
  devkit bugfix "race condition in cache invalidation" --test "go test ./..."`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		testCmd, _ := cmd.Flags().GetString("test")
		budget, _ := cmd.Flags().GetFloat64("budget")

		dirty, err := (&lib.Git{Dir: repoRoot}).HasUncommittedChanges()
		if err != nil {
			return fmt.Errorf("check git status: %w", err)
		}
		if dirty {
			return fmt.Errorf("working tree has uncommitted changes — commit or stash first")
		}

		available := runners.DetectRunners()
		runner := runners.FindRunner("claude", available)
		if runner == nil {
			return fmt.Errorf("claude CLI not found in PATH")
		}

		result, err := loops.RunBugfix(cmd.Context(), db, runner, &lib.Git{Dir: repoRoot}, loops.BugfixConfig{
			Description: strings.Join(args, " "),
			TestCmd:     testCmd,
			RepoRoot:    repoRoot,
			BudgetUSD:   budget,
		})
		if err != nil {
			return err
		}

		printBugfixResult(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(bugfixCmd)
	bugfixCmd.Flags().String("test", "", "Test command to verify the fix")
	bugfixCmd.Flags().Float64("budget", 0, "Maximum spend in USD (0 = unlimited)")
}

func printBugfixResult(r *loops.BugfixResult) {
	var totalCost float64
	for _, s := range r.Steps {
		totalCost += s.CostUSD
	}
	fmt.Printf("\n=== Bugfix Complete ===\n")
	fmt.Printf("Session: %s\n", r.Session.ID)
	fmt.Printf("Steps:   %d\n", len(r.Steps))
	fmt.Printf("Cost:    $%.4f\n", totalCost)
	fmt.Printf("\nRun `devkit status %s` for details.\n", r.Session.ID)
}
