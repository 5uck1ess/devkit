package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/runners"
	"github.com/spf13/cobra"
)

var (
	db       *lib.DB
	repoRoot string
	Version  = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "devkit",
	Short: "Deterministic orchestration for AI agent workflows",
	Long:  "Go CLI harness for devkit — deterministic loop control, process management, and unattended runs.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		root, err := findRepoRoot()
		if err != nil {
			return fmt.Errorf("not inside a git repo — run devkit from a project directory")
		}
		repoRoot = root

		dbPath := filepath.Join(root, ".devkit", "devkit.db")
		db, err = lib.OpenDB(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open state database at %s: %w", dbPath, err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if db != nil {
			return db.Close()
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().String("agent", "claude", "AI agent to use (claude, codex, gemini)")
}

func Execute() error {
	rootCmd.Version = Version
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	rootCmd.SetContext(ctx)
	return rootCmd.ExecuteContext(ctx)
}

func resolveRunner(name string) (runners.Runner, error) {
	return resolveRunnerFrom(name, runners.DetectRunners())
}

func resolveRunnerFrom(name string, available []runners.Runner) (runners.Runner, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	r := runners.FindRunner(name, available)
	if r != nil {
		return r, nil
	}
	var names []string
	for _, a := range available {
		names = append(names, a.Name())
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no AI agents found in PATH — install claude, codex, or gemini")
	}
	return nil, fmt.Errorf("agent %q not found — available: %s", name, strings.Join(names, ", "))
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .git directory found")
		}
		dir = parent
	}
}
