package cmd

import (
	"context"
	"testing"

	"github.com/5uck1ess/devkit/runners"
)

type stubRunner struct {
	name string
}

func (s *stubRunner) Name() string      { return s.name }
func (s *stubRunner) Available() bool   { return true }
func (s *stubRunner) Run(_ context.Context, _ string, _ runners.RunOpts) (runners.RunResult, error) {
	return runners.RunResult{}, nil
}

func TestResolveRunnerFrom_Found(t *testing.T) {
	available := []runners.Runner{&stubRunner{"claude"}, &stubRunner{"codex"}}
	r, err := resolveRunnerFrom("claude", available)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name() != "claude" {
		t.Errorf("got %s, want claude", r.Name())
	}
}

func TestResolveRunnerFrom_CaseInsensitive(t *testing.T) {
	available := []runners.Runner{&stubRunner{"claude"}, &stubRunner{"gemini"}}
	r, err := resolveRunnerFrom("GEMINI", available)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name() != "gemini" {
		t.Errorf("got %s, want gemini", r.Name())
	}
}

func TestResolveRunnerFrom_Whitespace(t *testing.T) {
	available := []runners.Runner{&stubRunner{"codex"}}
	r, err := resolveRunnerFrom("  codex  ", available)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name() != "codex" {
		t.Errorf("got %s, want codex", r.Name())
	}
}

func TestResolveRunnerFrom_NotFound(t *testing.T) {
	available := []runners.Runner{&stubRunner{"claude"}}
	_, err := resolveRunnerFrom("gpt", available)
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if got := err.Error(); got != `agent "gpt" not found — available: claude` {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestResolveRunnerFrom_NoAgents(t *testing.T) {
	_, err := resolveRunnerFrom("claude", nil)
	if err == nil {
		t.Fatal("expected error when no agents available")
	}
	if got := err.Error(); got != "no AI agents found in PATH — install claude, codex, or gemini" {
		t.Errorf("unexpected error: %s", got)
	}
}
