//go:build !windows

package runners

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// fakeClaude puts a `claude` script on PATH that prints stdout and exits with code.
func fakeClaude(t *testing.T, stdout string, code int) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\ncat <<'JSON'\n" + stdout + "\nJSON\nexit " + itoa(code) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestClaudeRun_IsErrorReturnsError(t *testing.T) {
	fakeClaude(t, `{"is_error":true,"result":"Credit balance is too low","total_cost_usd":0.01}`, 0)
	res, err := (&ClaudeRunner{}).Run(context.Background(), "hi", RunOpts{})
	if err == nil {
		t.Fatalf("expected error for is_error=true, got nil (output %q)", res.Output)
	}
	if !strings.Contains(err.Error(), "Credit balance is too low") {
		t.Errorf("error should carry the result text, got %v", err)
	}
	if res.CostUSD != 0.01 {
		t.Errorf("cost should be preserved on error, got %v", res.CostUSD)
	}
}

func TestClaudeRun_NonZeroExitWithJSONReturnsError(t *testing.T) {
	fakeClaude(t, `{"is_error":false,"result":"partial","total_cost_usd":0.02}`, 1)
	res, err := (&ClaudeRunner{}).Run(context.Background(), "hi", RunOpts{})
	if err == nil {
		t.Fatal("expected error for non-zero exit, got nil")
	}
	if res.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", res.ExitCode)
	}
	if res.Output != "partial" {
		t.Errorf("output should be preserved, got %q", res.Output)
	}
}

func TestClaudeRun_Success(t *testing.T) {
	fakeClaude(t, `{"is_error":false,"result":"ok","total_cost_usd":0.03}`, 0)
	res, err := (&ClaudeRunner{}).Run(context.Background(), "hi", RunOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Output != "ok" || res.CostUSD != 0.03 {
		t.Errorf("got %+v", res)
	}
}

func TestClaudeRun_MaxTurnsUsesErrorsField(t *testing.T) {
	fakeClaude(t, `{"subtype":"error_max_turns","is_error":true,"total_cost_usd":0.8,"errors":["Reached maximum number of turns (30)"]}`, 1)
	_, err := (&ClaudeRunner{}).Run(context.Background(), "hi", RunOpts{})
	if err == nil || !strings.Contains(err.Error(), "Reached maximum number of turns") {
		t.Fatalf("error should carry the errors[] text, got %v", err)
	}
}
