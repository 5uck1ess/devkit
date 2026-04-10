package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	devkitmcp "github.com/5uck1ess/devkit/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server on stdio",
	Long:  "Launch the devkit engine as an MCP server communicating via JSON-RPC over stdin/stdout.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir := os.Getenv("CLAUDE_PLUGIN_DATA")
		if dataDir == "" {
			dataDir = filepath.Join(repoRoot, ".devkit")
		}

		pluginRoot := os.Getenv("CLAUDE_PLUGIN_ROOT")
		workflowDir := filepath.Join(repoRoot, "workflows")
		if pluginRoot != "" {
			workflowDir = filepath.Join(pluginRoot, "workflows")
		}

		srv, err := devkitmcp.NewServer(repoRoot, dataDir, workflowDir)
		if err != nil {
			return fmt.Errorf("create MCP server: %w", err)
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		fmt.Fprintln(os.Stderr, "devkit MCP server ready")
		return srv.Serve(ctx)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
