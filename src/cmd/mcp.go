package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server on stdio",
	Long:  "Launch the devkit engine as an MCP server communicating via JSON-RPC over stdin/stdout.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "devkit MCP server starting...")
		// TODO: wire up in Task 11
		return fmt.Errorf("not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
