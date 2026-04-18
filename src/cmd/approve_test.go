package cmd

import (
	"bytes"
	"context"
	"io"
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

	var out bytes.Buffer
	if err := runApprove(context.Background(), &out, "plan"); err != nil {
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
	if !strings.Contains(out.String(), "approved by") {
		t.Errorf("stdout missing approval confirmation: %q", out.String())
	}
}

func TestRunApproveIdempotent(t *testing.T) {
	dir := newTestRepo(t)
	t.Chdir(dir)

	if err := runApprove(context.Background(), io.Discard, "plan"); err != nil {
		t.Fatalf("first approve: %v", err)
	}
	markerPath := filepath.Join(dir, ".devkit", "gates", "plan.approved")
	first, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("read after first: %v", err)
	}
	firstStat, err := os.Stat(markerPath)
	if err != nil {
		t.Fatalf("stat after first: %v", err)
	}

	if err := runApprove(context.Background(), io.Discard, "plan"); err != nil {
		t.Fatalf("second approve: %v", err)
	}
	second, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("read after second: %v", err)
	}
	secondStat, err := os.Stat(markerPath)
	if err != nil {
		t.Fatalf("stat after second: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("marker content changed on idempotent re-approve\nfirst:  %q\nsecond: %q", first, second)
	}
	// mtime must be preserved on a no-op re-approve — if it changed we
	// know the file was rewritten with identical content, which a naive
	// content-equality check would miss.
	if !firstStat.ModTime().Equal(secondStat.ModTime()) {
		t.Errorf("marker mtime changed on idempotent re-approve: %v -> %v", firstStat.ModTime(), secondStat.ModTime())
	}
}

func TestRunApproveRejectsInvalidName(t *testing.T) {
	dir := newTestRepo(t)
	t.Chdir(dir)

	err := runApprove(context.Background(), io.Discard, "../escape")
	if err == nil {
		t.Fatal("expected error for path traversal name")
	}
	if !strings.Contains(err.Error(), "invalid gate name") {
		t.Errorf("expected 'invalid gate name' in error, got: %v", err)
	}
	// Error message should include the regex anchors so users don't
	// read the allowlist as a partial-match pattern.
	if !strings.Contains(err.Error(), "^") || !strings.Contains(err.Error(), "$") {
		t.Errorf("expected regex anchors in error message, got: %v", err)
	}

	// Ensure no file was created at the traversal target.
	escaped := filepath.Join(dir, "..", "escape.approved")
	if _, err := os.Stat(escaped); err == nil {
		t.Errorf("traversal created %s", escaped)
	}
}

func TestRunApproveRejectsNullByteAtRegex(t *testing.T) {
	// Null-byte rejection must happen at the regex layer, not via the
	// filesystem refusing the name later. Otherwise the same input
	// could sneak past on an OS with lax filename validation.
	dir := newTestRepo(t)
	t.Chdir(dir)

	err := runApprove(context.Background(), io.Discard, "plan\x00")
	if err == nil {
		t.Fatal("expected error for null-byte name")
	}
	if !strings.Contains(err.Error(), "invalid gate name") {
		t.Errorf("expected regex-layer rejection, got: %v", err)
	}
}

func TestRunApproveOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	err := runApprove(context.Background(), io.Discard, "plan")
	if err == nil {
		t.Fatal("expected error outside git repo")
	}
	if !strings.Contains(err.Error(), "not inside a git repo") {
		t.Errorf("expected 'not inside a git repo' in error, got: %v", err)
	}
}

func TestParseGitUserRegexp(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantEmail string
	}{
		{"both present", "user.name Alice Example\nuser.email alice@example.com\n", "Alice Example", "alice@example.com"},
		{"name only", "user.name Bob\n", "Bob", ""},
		{"email only", "user.email c@d.io\n", "", "c@d.io"},
		{"empty", "", "", ""},
		{"crlf", "user.name Dana\r\nuser.email d@d.io\r\n", "Dana", "d@d.io"},
		{"blank lines", "\n\nuser.name Eve\n\n", "Eve", ""},
		{"name with spaces", "user.name  Two  Spaces\n", " Two  Spaces", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n, e := parseGitUserRegexp(tc.input)
			if n != tc.wantName {
				t.Errorf("name: got %q, want %q", n, tc.wantName)
			}
			if e != tc.wantEmail {
				t.Errorf("email: got %q, want %q", e, tc.wantEmail)
			}
		})
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
	// os.DevNull is portable across Unix and Windows ("/dev/null" vs
	// "NUL"); hardcoding a Unix path would break the test on Windows.
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, ".gitconfig-global"))
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
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
