package cmd

import (
	"fmt"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/loops"
	"github.com/5uck1ess/devkit/runners"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume <session-id>",
	Short: "Resume a paused or crashed session",
	Long:  "Picks up an improve session from where it left off, using the SQLite state and handoff file.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]

		session, err := db.GetSession(sessionID)
		if err != nil {
			return err
		}

		if session.Status != "paused" && session.Status != "failed" && session.Status != "running" {
			return fmt.Errorf("session %s has status %q — only paused, failed, or interrupted sessions can be resumed", sessionID, session.Status)
		}

		if session.Workflow != "improve" {
			return fmt.Errorf("resume only supports improve sessions — session %s is %q", sessionID, session.Workflow)
		}

		available := runners.DetectRunners()
		runner := runners.FindRunner("claude", available)
		if runner == nil {
			return fmt.Errorf("claude CLI not found in PATH")
		}

		// Check we're on the right branch
		git := &lib.Git{Dir: repoRoot}
		branch, err := git.CurrentBranch()
		if err != nil {
			return fmt.Errorf("get current branch: %w", err)
		}

		expectedBranch := fmt.Sprintf("self-improve/%s", sessionID)
		if branch != expectedBranch {
			fmt.Printf("Switching to branch %s...\n", expectedBranch)
			if err := git.CheckoutBranch(expectedBranch); err != nil {
				return fmt.Errorf("checkout branch %s: %w — does the branch still exist?", expectedBranch, err)
			}
		}

		result, err := loops.ResumeImproveLoop(cmd.Context(), db, runner, git, session, repoRoot)
		if err != nil {
			return err
		}

		printImproveResult(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
