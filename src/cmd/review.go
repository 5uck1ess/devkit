package cmd

import (
	"fmt"
	"strings"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/loops"
	"github.com/5uck1ess/devkit/runners"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review [prompt]",
	Short: "Multi-agent code review of current branch diff",
	Long:  "Dispatches the diff to all available agents in parallel, consolidates findings.",
	Example: `  devkit review
  devkit review "check for DRY violations"
  devkit review --security
  devkit review --agents claude,codex`,
	RunE: func(cmd *cobra.Command, args []string) error {
		security, _ := cmd.Flags().GetBool("security")
		agentList, _ := cmd.Flags().GetString("agents")

		prompt := strings.Join(args, " ")

		var agents []string
		if agentList != "" {
			for _, a := range strings.Split(agentList, ",") {
				a = strings.TrimSpace(a)
				if a != "" {
					agents = append(agents, a)
				}
			}
		}

		available := runners.DetectRunners()
		git := &lib.Git{Dir: repoRoot}

		cfg := loops.ReviewConfig{
			Prompt:   prompt,
			Security: security,
			Agents:   agents,
			RepoRoot: repoRoot,
		}

		result, err := loops.RunReview(cmd.Context(), db, available, git, cfg)
		if err != nil {
			return err
		}

		printReviewResult(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reviewCmd)
	reviewCmd.Flags().Bool("security", false, "Focus on security issues")
	reviewCmd.Flags().String("agents", "", "Comma-separated list of agents (default: all available)")
}

func printReviewResult(r *loops.ReviewResult) {
	fmt.Printf("\n=== Review: %s ===\n\n", r.Session.ID)

	for _, res := range r.Results {
		fmt.Printf("### %s\n", strings.ToUpper(res.Agent))
		if res.Error != nil {
			fmt.Printf("Error: %s\n\n", res.Error)
			continue
		}
		fmt.Printf("%s\n\n", res.Output)
	}

	fmt.Printf("---\nAgents: %d/%d responded\n", countSuccessful(r.Results), len(r.Results))
}

func countSuccessful(results []loops.AgentResult) int {
	n := 0
	for _, r := range results {
		if r.Error == nil {
			n++
		}
	}
	return n
}
