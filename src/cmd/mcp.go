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
	// MCP must boot even when Claude Code's working directory is outside a
	// git repo — otherwise the JSON-RPC initialize handshake never completes
	// and the user only sees -32000. Tool calls that genuinely need git
	// state can check repoRoot themselves and return a structured error.
	Annotations: map[string]string{"allow_no_git": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, err := resolveMCPDataDir(repoRoot)
		if err != nil {
			return fmt.Errorf("resolving MCP data dir: %w", err)
		}

		pluginRoot := os.Getenv("CLAUDE_PLUGIN_ROOT")
		workflowDir := ""
		switch {
		case pluginRoot != "":
			workflowDir = filepath.Join(pluginRoot, "workflows")
		case repoRoot != "":
			workflowDir = filepath.Join(repoRoot, "workflows")
		}

		srv, err := devkitmcp.NewServer(repoRoot, dataDir, workflowDir)
		if err != nil {
			return fmt.Errorf("create MCP server: %w", err)
		}
		defer srv.Close()

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		fmt.Fprintln(os.Stderr, "devkit MCP server ready")
		return srv.Serve(ctx)
	},
}

// resolveMCPDataDir picks where the MCP server stores its state DB. Order:
//  1. CLAUDE_PLUGIN_DATA (explicit override from the harness)
//  2. <repoRoot>/.devkit (when launched inside a git project)
//  3. <user cache dir>/devkit (fallback for non-git cwds, e.g. running CC
//     from $HOME — the symptom that produced issue #105)
//
// The directory is created on demand so the caller can rely on it existing.
func resolveMCPDataDir(repoRoot string) (string, error) {
	dir := os.Getenv("CLAUDE_PLUGIN_DATA")
	if dir == "" {
		if repoRoot != "" {
			dir = filepath.Join(repoRoot, ".devkit")
		} else {
			cache, err := os.UserCacheDir()
			if err != nil {
				return "", fmt.Errorf("locating user cache dir: %w", err)
			}
			dir = filepath.Join(cache, "devkit")
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	return dir, nil
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
