package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// approveNamePattern bounds the gate name to chars that are safe as a
// filename on every target filesystem and unambiguously non-traversal.
// Rejecting leading `.` also prevents shadowing dotfiles like `.gitignore`
// if someone ever puts the gates dir in a more sensitive location.
var approveNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

var approveCmd = &cobra.Command{
	Use:   "approve <name>",
	Short: "Approve a workflow gate",
	Long: `Approve a named gate so a polling gate step in a workflow can continue.

A gate step in a workflow is a command that blocks until a marker file
exists under .devkit/gates/. This command writes that marker with a
timestamp and approver so the gate step can exit cleanly.

The name must match [a-zA-Z0-9][a-zA-Z0-9_-]{0,63} — no path separators,
no leading dot, 1-64 chars.`,
	Example: `  devkit approve plan
  devkit approve deploy-prod`,
	Args: cobra.ExactArgs(1),
	// Skip the root PersistentPreRunE — approve only needs a repo root,
	// not the sessions DB. Avoids creating .devkit/devkit.db for users
	// who only want to approve a gate without ever running a workflow
	// in this directory (e.g. CI runners approving from a script).
	PersistentPreRunE:  func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error { return nil },
	RunE: func(cmd *cobra.Command, args []string) error {
		return runApprove(args[0])
	},
}

func init() {
	rootCmd.AddCommand(approveCmd)
}

func runApprove(name string) error {
	if !approveNamePattern.MatchString(name) {
		return fmt.Errorf("invalid gate name %q — must match [a-zA-Z0-9][a-zA-Z0-9_-]{0,63}", name)
	}

	root, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("not inside a git repo — run devkit approve from a project directory")
	}

	gatesDir := filepath.Join(root, ".devkit", "gates")
	if err := os.MkdirAll(gatesDir, 0o700); err != nil {
		return fmt.Errorf("create gates dir: %w", err)
	}

	markerPath := filepath.Join(gatesDir, name+".approved")

	if existing, err := os.ReadFile(markerPath); err == nil {
		fmt.Printf("gate %q already approved:\n%s", name, existing)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read existing marker: %w", err)
	}

	approver := approverIdentity()
	content := fmt.Sprintf("approved_at: %s\napproved_by: %s\n",
		time.Now().UTC().Format(time.RFC3339),
		approver)

	// O_EXCL so a concurrent `devkit approve` can't race us and end up
	// with a half-written marker. The earlier ReadFile check handles the
	// idempotent case; this handles the narrow race where two approvers
	// hit it between the stat and the create.
	f, err := os.OpenFile(markerPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			existing, rerr := os.ReadFile(markerPath)
			if rerr == nil {
				fmt.Printf("gate %q already approved:\n%s", name, existing)
				return nil
			}
		}
		return fmt.Errorf("write marker: %w", err)
	}
	if _, werr := f.WriteString(content); werr != nil {
		f.Close()
		os.Remove(markerPath)
		return fmt.Errorf("write marker: %w", werr)
	}
	if cerr := f.Close(); cerr != nil {
		return fmt.Errorf("close marker: %w", cerr)
	}

	fmt.Printf("gate %q approved by %s\nmarker: %s\n", name, approver, markerPath)
	return nil
}

// approverIdentity prefers git config so the marker reflects the repo's
// commit-author identity (which is what reviewers recognise). Falls back
// through $USER and finally "unknown" so the marker is always written —
// refusing to approve because we can't identify the user would be worse
// than a marker that says "unknown".
func approverIdentity() string {
	if name := gitConfig("user.name"); name != "" {
		if email := gitConfig("user.email"); email != "" {
			return fmt.Sprintf("%s <%s>", name, email)
		}
		return name
	}
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		return u
	}
	return "unknown"
}

func gitConfig(key string) string {
	out, err := exec.Command("git", "config", "--get", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
