package cmd

import (
	"fmt"
	"strings"

	"github.com/5uck1ess/devkit/loops"
	"github.com/5uck1ess/devkit/runners"
	"github.com/spf13/cobra"
)

var dispatchCmd = &cobra.Command{
	Use:   "dispatch [prompt]",
	Short: "Send a task to multiple agents and compare outputs",
	Long:  "Dispatches the same prompt to all available agents in parallel, collects and compares results.",
	Example: `  devkit dispatch "compare caching approaches"
  devkit dispatch --agents claude,gemini "review the API design"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentList, _ := cmd.Flags().GetString("agents")

		prompt := strings.Join(args, " ")

		var agents []string
		if agentList != "" {
			agents = strings.Split(agentList, ",")
		}

		available := runners.DetectRunners()

		cfg := loops.DispatchConfig{
			Prompt:   prompt,
			Agents:   agents,
			RepoRoot: repoRoot,
		}

		result, err := loops.RunDispatch(cmd.Context(), db, available, cfg)
		if err != nil {
			return err
		}

		printDispatchResult(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dispatchCmd)
	dispatchCmd.Flags().String("agents", "", "Comma-separated list of agents (default: all available)")
}

func printDispatchResult(r *loops.DispatchResult) {
	fmt.Printf("\n=== Dispatch: %s ===\n\n", r.Session.ID)

	for _, res := range r.Results {
		fmt.Printf("### %s\n", strings.ToUpper(res.Agent))
		if res.Error != nil {
			fmt.Printf("Error: %s\n\n", res.Error)
			continue
		}
		fmt.Printf("%s\n\n", res.Output)
	}

	var totalCost float64
	for _, res := range r.Results {
		totalCost += res.Cost
	}
	fmt.Printf("---\nAgents: %d/%d responded | Total cost: $%.4f\n", countSuccess(r.Results), len(r.Results), totalCost)
}

func countSuccess(results []loops.AgentResult) int {
	n := 0
	for _, r := range results {
		if r.Error == nil {
			n++
		}
	}
	return n
}
