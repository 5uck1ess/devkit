package cmd

import (
	"fmt"
	"strings"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/loops"
	"github.com/spf13/cobra"
)

var testGenCmd = &cobra.Command{
	Use:   "test-gen [target]",
	Short: "Generate tests for target code, run them, fix failures",
	Long:  "Analyzes target code, generates comprehensive tests, runs them, and iterates until green.",
	Example: `  devkit test-gen src/auth/
  devkit test-gen lib/parser.go --test "go test ./..."
  devkit test-gen src/ --test "npm test" --budget 5.00`,
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

		agentName, _ := cmd.Flags().GetString("agent")
		runner, err := resolveRunner(agentName)
		if err != nil {
			return err
		}

		result, err := loops.RunTestGen(cmd.Context(), db, runner, &lib.Git{Dir: repoRoot}, loops.TestGenConfig{
			Target:    strings.Join(args, " "),
			TestCmd:   testCmd,
			RepoRoot:  repoRoot,
			BudgetUSD: budget,
		})
		if err != nil {
			return err
		}

		var totalCost float64
		for _, s := range result.Steps {
			totalCost += s.CostUSD
		}
		fmt.Printf("\n=== Test Generation Complete ===\n")
		fmt.Printf("Session: %s\n", result.Session.ID)
		fmt.Printf("Steps:   %d\n", len(result.Steps))
		fmt.Printf("Cost:    $%.4f\n", totalCost)
		fmt.Printf("\nRun `devkit status %s` for details.\n", result.Session.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testGenCmd)
	testGenCmd.Flags().String("test", "", "Test command to run generated tests")
	testGenCmd.Flags().Float64("budget", 0, "Maximum spend in USD (0 = unlimited)")
}
