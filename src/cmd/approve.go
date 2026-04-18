package cmd

import (
	"context"
	"fmt"
	"io"
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
		return runApprove(cmd.Context(), cmd.OutOrStdout(), args[0])
	},
}

func init() {
	rootCmd.AddCommand(approveCmd)
}

func runApprove(ctx context.Context, out io.Writer, name string) error {
	if !approveNamePattern.MatchString(name) {
		return fmt.Errorf("invalid gate name %q — must match ^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$", name)
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
		fmt.Fprintf(out, "gate %q already approved:\n%s", name, existing)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read existing marker: %w", err)
	}

	approver := approverIdentity(ctx)
	content := fmt.Sprintf("approved_at: %s\napproved_by: %s\n",
		time.Now().UTC().Format(time.RFC3339),
		approver)

	// Atomic publish: write to a temp file in the same directory, then
	// rename. A polling gate does `[ -f ... ]`, which sees the marker
	// only after rename makes the directory entry visible — so a crash
	// or write error mid-approve can no longer unblock the workflow on
	// a partially-written file. The O_EXCL-on-the-final-path approach
	// was unsafe because the marker became visible before WriteString
	// and Close had returned.
	tmp, err := os.CreateTemp(gatesDir, name+".approved.tmp-*")
	if err != nil {
		return fmt.Errorf("create temp marker: %w", err)
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp marker: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp marker: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp marker: %w", err)
	}

	if err := os.Rename(tmpPath, markerPath); err != nil {
		return fmt.Errorf("publish marker: %w", err)
	}
	committed = true

	fmt.Fprintf(out, "gate %q approved by %s\nmarker: %s\n", name, approver, markerPath)
	return nil
}

// approverIdentity prefers git config so the marker reflects the repo's
// commit-author identity (which is what reviewers recognise). Falls back
// through $USER and finally "unknown" so the marker is always written —
// refusing to approve because we can't identify the user would be worse
// than a marker that says "unknown".
//
// Runs git under a 2s timeout because `git config --get` has been
// observed to hang on locked-index or misconfigured repos, which would
// deadlock the CLI indefinitely. A single --get-regexp call fetches
// both user.name and user.email in one subprocess instead of two.
func approverIdentity(ctx context.Context) string {
	tctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(tctx, "git", "config", "--get-regexp", `^user\.(name|email)$`).Output()
	if err == nil {
		name, email := parseGitUserRegexp(string(out))
		switch {
		case name != "" && email != "":
			return fmt.Sprintf("%s <%s>", name, email)
		case name != "":
			return name
		}
	}

	fallback := "unknown"
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		fallback = u
	}
	// Surface git failures so the user can tell "git timed out" from
	// "git wasn't configured" from "git config is corrupt" — without
	// this, every such case silently produces approver=<USER-or-unknown>
	// with no diagnostic trail. We only log when git actually failed
	// (err != nil); a git run that succeeded but returned no user.* keys
	// is a legitimate unconfigured-repo case and doesn't need noise.
	if err != nil {
		fmt.Fprintf(os.Stderr, "devkit approve: git identity unavailable (%v); approver=%s\n", err, fallback)
	}
	return fallback
}

// parseGitUserRegexp extracts user.name and user.email from the output
// of `git config --get-regexp`. Each line is "<key> <value>" separated
// by a single space; values may themselves contain spaces (e.g. a full
// name) so we split on the first space only.
func parseGitUserRegexp(out string) (name, email string) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		switch key {
		case "user.name":
			name = val
		case "user.email":
			email = val
		}
	}
	return
}
