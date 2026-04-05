package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/5uck1ess/devkit/engine"
	"github.com/5uck1ess/devkit/lib"
	"github.com/spf13/cobra"
)

var validWorkflowName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var workflowCmd = &cobra.Command{
	Use:   "workflow [name] [description...]",
	Short: "Run a YAML workflow by name",
	Long:  "Execute a workflow from the workflows/ directory. The engine handles step sequencing, branching, loops, and parallel dispatch deterministically.",
	Example: `  devkit workflow feature "add JWT authentication"
  devkit workflow bugfix "fix null pointer in handler"
  devkit workflow list`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Handle "list" subcommand
		if name == "list" {
			return listWorkflows()
		}

		if len(args) < 2 {
			return fmt.Errorf("usage: devkit workflow <name> <description>")
		}

		// Validate workflow name to prevent path traversal
		if !validWorkflowName.MatchString(name) {
			return fmt.Errorf("invalid workflow name %q — use only letters, numbers, hyphens, underscores", name)
		}

		dirty, err := (&lib.Git{Dir: repoRoot}).HasUncommittedChanges()
		if err != nil {
			return fmt.Errorf("check git status: %w", err)
		}
		if dirty {
			return fmt.Errorf("working tree has uncommitted changes — commit or stash first")
		}

		// Find workflow file
		wfPath := findWorkflowFile(name)
		if wfPath == "" {
			return fmt.Errorf("workflow %q not found — run `devkit workflow list`", name)
		}

		wf, err := engine.ParseFile(wfPath)
		if err != nil {
			return fmt.Errorf("parse workflow: %w", err)
		}

		agentName, _ := cmd.Flags().GetString("agent")
		runner, err := resolveRunner(agentName)
		if err != nil {
			return err
		}

		budget, _ := cmd.Flags().GetFloat64("budget")
		// CLI flag overrides YAML budget; fall back to YAML if flag not set
		if budget == 0 && wf.Budget.Limit > 0 {
			// Convert token budget to rough USD estimate ($0.01 per 1K tokens)
			budget = float64(wf.Budget.Limit) / 1000.0 * 0.01
		}

		eng, err := engine.NewEngine(db, &lib.Git{Dir: repoRoot}, runner, repoRoot)
		if err != nil {
			return err
		}

		description := strings.Join(args[1:], " ")
		result, err := eng.RunWorkflow(cmd.Context(), wf, engine.RunConfig{
			Input:     description,
			BudgetUSD: budget,
		})
		if err != nil {
			return err
		}

		printWorkflowResult(wf.Name, result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(workflowCmd)
	workflowCmd.Flags().Float64("budget", 0, "Maximum spend in USD (0 = unlimited)")
}

func findWorkflowFile(name string) string {
	// Search in repo workflows/ directory, then plugin workflows/
	candidates := []string{
		filepath.Join(repoRoot, "workflows", name+".yml"),
		filepath.Join(repoRoot, "workflows", name+".yaml"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func listWorkflows() error {
	dir := filepath.Join(repoRoot, "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("no workflows/ directory found in %s", repoRoot)
	}

	fmt.Println("Available workflows:")
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}
		wfName := strings.TrimSuffix(strings.TrimSuffix(name, ".yml"), ".yaml")
		path := filepath.Join(dir, name)
		wf, err := engine.ParseFile(path)
		if err != nil {
			fmt.Printf("  %-20s  (parse error: %v)\n", wfName, err)
			continue
		}
		fmt.Printf("  %-20s  %s\n", wfName, wf.Description)
	}
	return nil
}

func printWorkflowResult(name string, r *engine.Result) {
	fmt.Printf("\n=== %s Complete ===\n", name)
	fmt.Printf("Session: %s\n", r.Session.ID)
	fmt.Printf("Steps:   %d\n", len(r.Steps))
	fmt.Printf("Cost:    $%.4f\n", r.TotalUSD)
	fmt.Printf("\nRun `devkit status %s` for details.\n", r.Session.ID)
}
