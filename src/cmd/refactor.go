package cmd

import (
	"fmt"
	"strings"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/loops"
	"github.com/spf13/cobra"
)

var refactorCmd = &cobra.Command{
	Use:   "refactor [description]",
	Short: "Full refactor lifecycle: analyze, transform, verify",
	Long:  "Spawns Claude for each step: analyze code smells → apply transformations → verify tests still pass.",
	Example: `  devkit refactor "extract auth middleware into shared package" --target src/
  devkit refactor "flatten nested callbacks to async/await" --test "npm test"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
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

		result, err := loops.RunRefactor(cmd.Context(), db, runner, &lib.Git{Dir: repoRoot}, loops.RefactorConfig{
			Description: strings.Join(args, " "),
			Target:      target,
			TestCmd:     testCmd,
			RepoRoot:    repoRoot,
			BudgetUSD:   budget,
		})
		if err != nil {
			return err
		}

		printRefactorResult(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(refactorCmd)
	refactorCmd.Flags().String("target", ".", "Directory or file to refactor")
	refactorCmd.Flags().String("test", "", "Test command to verify no behavior change")
	refactorCmd.Flags().Float64("budget", 0, "Maximum spend in USD (0 = unlimited)")
	refactorCmd.Flags().String("agent", "claude", "AI agent to use (claude, codex, gemini)")
}

func printRefactorResult(r *loops.RefactorResult) {
	var totalCost float64
	for _, s := range r.Steps {
		totalCost += s.CostUSD
	}
	fmt.Printf("\n=== Refactor Complete ===\n")
	fmt.Printf("Session: %s\n", r.Session.ID)
	fmt.Printf("Steps:   %d\n", len(r.Steps))
	fmt.Printf("Cost:    $%.4f\n", totalCost)
	fmt.Printf("\nRun `devkit status %s` for details.\n", r.Session.ID)
}
