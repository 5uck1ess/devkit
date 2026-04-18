package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestApproveNameValidation(t *testing.T) {
	tests := []struct {
		name    string
		gate    string
		wantErr bool
	}{
		{"simple", "plan", false},
		{"with dash", "deploy-prod", false},
		{"with underscore", "gate_1", false},
		{"alphanumeric", "Gate42", false},
		{"max length", strings.Repeat("a", 64), false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 65), true},
		{"leading dot", ".hidden", true},
		{"leading dash", "-flag", true},
		{"path traversal dotdot", "../escape", true},
		{"path separator unix", "a/b", true},
		{"path separator windows", "a\\b", true},
		{"space", "plan v2", true},
		{"null byte", "plan\x00", true},
		{"leading digit ok", "1plan", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := approveNamePattern.MatchString(tc.gate)
			if got == tc.wantErr {
				t.Fatalf("name %q: match=%v, wantErr=%v", tc.gate, got, tc.wantErr)
			}
		})
	}
}

func TestRunApproveWritesMarker(t *testing.T) {
	dir := newTestRepo(t)
	t.Chdir(dir)

	if err := runApprove("plan"); err != nil {
		t.Fatalf("runApprove: %v", err)
	}

	markerPath := filepath.Join(dir, ".devkit", "gates", "plan.approved")
	data, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if !strings.Contains(string(data), "approved_at:") {
		t.Errorf("marker missing approved_at: %q", data)
	}
	if !strings.Contains(string(data), "approved_by:") {
		t.Errorf("marker missing approved_by: %q", data)
	}
}

func TestRunApproveIdempotent(t *testing.T) {
	dir := newTestRepo(t)
	t.Chdir(dir)

	if err := runApprove("plan"); err != nil {
		t.Fatalf("first approve: %v", err)
	}
	markerPath := filepath.Join(dir, ".devkit", "gates", "plan.approved")
	first, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("read after first: %v", err)
	}

	if err := runApprove("plan"); err != nil {
		t.Fatalf("second approve: %v", err)
	}
	second, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("read after second: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("marker content changed on idempotent re-approve\nfirst:  %q\nsecond: %q", first, second)
	}
}

func TestRunApproveRejectsInvalidName(t *testing.T) {
	dir := newTestRepo(t)
	t.Chdir(dir)

	err := runApprove("../escape")
	if err == nil {
		t.Fatal("expected error for path traversal name")
	}
	if !strings.Contains(err.Error(), "invalid gate name") {
		t.Errorf("expected 'invalid gate name' in error, got: %v", err)
	}

	// Ensure no file was created at the traversal target.
	escaped := filepath.Join(dir, "..", "escape.approved")
	if _, err := os.Stat(escaped); err == nil {
		t.Errorf("traversal created %s", escaped)
	}
}

func TestRunApproveOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	err := runApprove("plan")
	if err == nil {
		t.Fatal("expected error outside git repo")
	}
	if !strings.Contains(err.Error(), "not inside a git repo") {
		t.Errorf("expected 'not inside a git repo' in error, got: %v", err)
	}
}

// newTestRepo creates a temp directory with a .git sentinel dir so
// findRepoRoot resolves to it. No real git history is needed — the repo
// root check is a single filepath.Stat for .git. We also pre-seed a
// minimal git config so approverIdentity doesn't reach up to the real
// user's ~/.gitconfig and leak their identity into a CI test log.
func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	// Confine git config lookups to this repo so the test is hermetic.
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, ".gitconfig-global"))
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("HOME", dir)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH — approver identity test needs git")
	}
	cmd := exec.Command("git", "-C", dir, "init", "--quiet")
	if err := cmd.Run(); err == nil {
		_ = exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()
		_ = exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run()
	}
	return dir
}
