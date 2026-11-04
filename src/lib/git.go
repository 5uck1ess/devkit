package lib

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Git struct {
	Dir string
}

func (g *Git) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (g *Git) CreateBranch(name string) error {
	_, err := g.run("checkout", "-b", name)
	return err
}

func (g *Git) CheckoutBranch(name string) error {
	_, err := g.run("checkout", name)
	return err
}

func (g *Git) CurrentBranch() (string, error) {
	return g.run("branch", "--show-current")
}

func (g *Git) CommitAll(msg string) error {
	if _, err := g.run("add", "-A"); err != nil {
		return err
	}
	// Check if there's anything to commit
	status, err := g.run("status", "--porcelain")
	if err != nil {
		return err
	}
	if status == "" {
		return nil // nothing to commit
	}
	_, err = g.run("commit", "-m", msg)
	return err
}

// Snapshot records the current working tree (tracked and untracked,
// honoring .gitignore) as a git tree object and returns its id. Unlike a
// commit it runs no hooks, so it works even when the tree would fail a
// pre-commit check.
func (g *Git) Snapshot() (string, error) {
	if _, err := g.run("add", "-A"); err != nil {
		return "", err
	}
	return g.run("write-tree")
}

// RestoreSnapshot makes the working tree match a tree from Snapshot:
// modified and deleted files come back, files created since are removed.
func (g *Git) RestoreSnapshot(tree string) error {
	if _, err := g.run("read-tree", tree); err != nil {
		return err
	}
	if _, err := g.run("checkout-index", "-a", "-f"); err != nil {
		return err
	}
	_, err := g.run("clean", "-fd")
	return err
}

func (g *Git) DiffStat() (string, error) {
	// Stage first so untracked files are included in the stat
	g.run("add", "-A")
	return g.run("diff", "--cached", "--stat")
}

func (g *Git) DefaultBranch() string {
	// Try to detect from remote
	if ref, err := g.run("symbolic-ref", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimPrefix(ref, "refs/remotes/origin/")
	}
	// Fallback: try main, then master
	if _, err := g.run("rev-parse", "--verify", "main"); err == nil {
		return "main"
	}
	if _, err := g.run("rev-parse", "--verify", "master"); err == nil {
		return "master"
	}
	return "main"
}

func (g *Git) DiffFromMain() (string, error) {
	base := g.DefaultBranch()
	diff, err := g.run("diff", base+"...HEAD")
	if err != nil {
		// fallback to cached + unstaged
		diff, err = g.run("diff", "HEAD")
	}
	if err != nil {
		diff, err = g.run("diff", "--cached")
	}
	return diff, err
}

func (g *Git) DiffSummary() (string, error) {
	return g.run("diff", "--stat", "HEAD~1")
}

func (g *Git) HasUncommittedChanges() (bool, error) {
	status, err := g.run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return status != "", nil
}
