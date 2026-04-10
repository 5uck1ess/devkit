package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/5uck1ess/devkit/lib"
	"github.com/5uck1ess/devkit/runners"
)

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

func tempDB(t *testing.T) *lib.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := lib.OpenDB(filepath.Join(dir, ".devkit", "devkit.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func initGitRepo(t *testing.T) (string, *lib.Git) {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup %v: %s", args, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"git", "add", "-A"},
		{"git", "commit", "-m", "initial"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup %v: %s", args, out)
		}
	}
	return dir, &lib.Git{Dir: dir}
}

type mockRunner struct {
	name      string
	responses []runners.RunResult
	errors    []error
	callIdx   int
	prompts   []string
	mu        sync.Mutex
}

func newMockRunner(responses []runners.RunResult, errs []error) *mockRunner {
	return &mockRunner{name: "mock", responses: responses, errors: errs}
}

func (m *mockRunner) Name() string    { return m.name }
func (m *mockRunner) Available() bool { return true }

func (m *mockRunner) Run(ctx context.Context, prompt string, opts runners.RunOpts) (runners.RunResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prompts = append(m.prompts, prompt)
	idx := m.callIdx
	m.callIdx++
	// Check errors first — if error is set, return zero result + error
	if idx < len(m.errors) && m.errors[idx] != nil {
		return runners.RunResult{}, m.errors[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return runners.RunResult{Output: "mock exhausted"}, nil
}

func result(output string) runners.RunResult {
	return runners.RunResult{Output: output, CostUSD: 0.01}
}

func mustEngine(t *testing.T, db *lib.DB, git *lib.Git, runner runners.Runner, repoRoot string) *Engine {
	t.Helper()
	eng, err := NewEngine(db, git, runner, repoRoot)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return eng
}

// ---------------------------------------------------------------------------
// Parse tests
// ---------------------------------------------------------------------------

func TestParseMinimal(t *testing.T) {
	yaml := `
name: Test
description: A test workflow
steps:
  - id: step1
    model: fast
    prompt: "Do something"
`
	wf, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if wf.Name != "Test" {
		t.Errorf("name = %q, want Test", wf.Name)
	}
	if len(wf.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(wf.Steps))
	}
	if wf.Steps[0].ID != "step1" {
		t.Errorf("step id = %q, want step1", wf.Steps[0].ID)
	}
}

func TestParseWithLoop(t *testing.T) {
	yaml := `
name: Looper
description: test
steps:
  - id: fix
    model: smart
    prompt: "Fix it"
    loop:
      max: 5
      until: ALL_DONE
`
	wf, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if wf.Steps[0].Loop == nil {
		t.Fatal("expected loop to be set")
	}
	if wf.Steps[0].Loop.Max != 5 {
		t.Errorf("loop max = %d, want 5", wf.Steps[0].Loop.Max)
	}
	if wf.Steps[0].Loop.Until != "ALL_DONE" {
		t.Errorf("loop until = %q, want ALL_DONE", wf.Steps[0].Loop.Until)
	}
}

func TestParseWithBranch(t *testing.T) {
	yaml := `
name: Brancher
description: test
steps:
  - id: classify
    model: fast
    prompt: "Classify"
    branch:
      - when: "TINY"
        goto: quick
      - when: "LARGE"
        goto: full
  - id: full
    model: smart
    prompt: "Full pipeline"
  - id: quick
    model: fast
    prompt: "Quick fix"
`
	wf, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(wf.Steps[0].Branch) != 2 {
		t.Fatalf("branches = %d, want 2", len(wf.Steps[0].Branch))
	}
	if wf.Steps[0].Branch[0].Goto != "quick" {
		t.Errorf("branch[0].goto = %q, want quick", wf.Steps[0].Branch[0].Goto)
	}
}

func TestParseValidation(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{"missing name", `steps: [{id: s, prompt: x}]`, "missing name"},
		{"no steps", `name: T`, "no steps"},
		{"duplicate id", `name: T
steps:
  - {id: a, prompt: x}
  - {id: a, prompt: y}`, "duplicate step id"},
		{"bad branch target", `name: T
steps:
  - id: a
    prompt: x
    branch: [{when: "x", goto: missing}]`, "branch target"},
		{"negative budget", `name: T
budget: {limit: -100}
steps: [{id: a, prompt: x}]`, "negative budget"},
		{"parallel with prompt", `name: T
steps:
  - id: a
    prompt: "do something"
    parallel: [b]
  - id: b
    prompt: "other"`, "mutually exclusive"},
		{"parallel with loop", `name: T
steps:
  - id: a
    parallel: [b]
    loop: {max: 3, until: DONE}
  - id: b
    prompt: "other"`, "mutually exclusive"},
		{"command with prompt", `name: T
steps:
  - id: a
    command: "echo hi"
    prompt: "do thing"`, "mutually exclusive"},
		{"parallel with command", `name: T
steps:
  - id: a
    command: "echo hi"
    parallel: [b]
  - id: b
    prompt: "other"`, "mutually exclusive"},
		{"command with loop", `name: T
steps:
  - id: a
    command: "echo hi"
    loop: {max: 3, until: DONE}`, "mutually exclusive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q doesn't contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestParseBudget(t *testing.T) {
	yaml := `
name: Budgeted
description: test
budget:
  limit: 300000
  downgrade: fast
steps:
  - id: s1
    model: smart
    prompt: "Do"
`
	wf, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if wf.Budget.Limit != 300000 {
		t.Errorf("budget limit = %d, want 300000", wf.Budget.Limit)
	}
	if wf.Budget.Downgrade != "fast" {
		t.Errorf("budget downgrade = %q, want fast", wf.Budget.Downgrade)
	}
}

// ---------------------------------------------------------------------------
// Interpolation tests
// ---------------------------------------------------------------------------

func TestInterpolate(t *testing.T) {
	outputs := map[string]string{
		"plan":  "1. Do X\n2. Do Y",
		"build": "compiled OK",
	}
	prompt := "Input: {{input}}\nPlan: {{plan}}\nBuild: {{build}}"
	got := Interpolate(prompt, "add auth", outputs)

	if !strings.Contains(got, "Input: add auth") {
		t.Error("input not interpolated")
	}
	if !strings.Contains(got, "Plan: 1. Do X") {
		t.Error("plan not interpolated")
	}
	if !strings.Contains(got, "Build: compiled OK") {
		t.Error("build not interpolated")
	}
}

func TestInterpolateMissing(t *testing.T) {
	got := Interpolate("ref: {{missing}}", "input", map[string]string{})
	if !strings.Contains(got, "{{missing}}") {
		t.Error("missing variable should be left as-is")
	}
}

// ---------------------------------------------------------------------------
// Branch evaluation tests
// ---------------------------------------------------------------------------

func TestEvalBranch(t *testing.T) {
	branches := []Branch{
		{When: "TINY", Goto: "quick"},
		{When: "SMALL", Goto: "plan"},
	}

	tests := []struct {
		output string
		want   string
	}{
		{"TINY: just a typo fix", "quick"},
		{"tiny change", "quick"}, // case insensitive
		{"SMALL: one function", "plan"},
		{"MEDIUM: multiple files", ""}, // no match
		{"LARGE: new subsystem", ""},
	}

	for _, tt := range tests {
		got := EvalBranch(tt.output, branches)
		if got != tt.want {
			t.Errorf("EvalBranch(%q) = %q, want %q", tt.output, got, tt.want)
		}
	}
}

func TestEvalBranchFirstMatchWins(t *testing.T) {
	branches := []Branch{
		{When: "error", Goto: "retry"},
		{When: "error", Goto: "fail"},
	}
	got := EvalBranch("got an error", branches)
	if got != "retry" {
		t.Errorf("first match should win, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Engine execution tests
// ---------------------------------------------------------------------------

func TestNewEngineValidation(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)
	runner := newMockRunner(nil, nil)

	if _, err := NewEngine(nil, git, runner, dir); err == nil {
		t.Error("expected error for nil db")
	}
	if _, err := NewEngine(db, nil, runner, dir); err == nil {
		t.Error("expected error for nil git")
	}
	if _, err := NewEngine(db, git, nil, dir); err == nil {
		t.Error("expected error for nil runner")
	}
	if _, err := NewEngine(db, git, runner, ""); err == nil {
		t.Error("expected error for empty repoRoot")
	}
	if _, err := NewEngine(db, git, runner, dir); err != nil {
		t.Errorf("valid args should succeed: %v", err)
	}
}

func TestRunWorkflowNegativeBudget(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)
	runner := newMockRunner([]runners.RunResult{result("ok")}, nil)
	eng := mustEngine(t, db, git, runner, dir)

	wf := &Workflow{Name: "test", Steps: []WfStep{{ID: "s1", Prompt: "Do"}}}
	_, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test", BudgetUSD: -1.0})
	if err == nil {
		t.Fatal("expected error for negative budget")
	}
	if !strings.Contains(err.Error(), "invalid budget") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunWorkflowSimple(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner([]runners.RunResult{
		result("planned: do A then B"),
		result("implemented A and B"),
	}, nil)

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "plan", Model: "smart", Prompt: "Plan: {{input}}"},
			{ID: "impl", Model: "smart", Prompt: "Implement: {{plan}}"},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "add auth"})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}
	if res.TotalUSD != 0.02 {
		t.Errorf("total cost = %f, want 0.02", res.TotalUSD)
	}
	if len(res.Steps) != 2 {
		t.Errorf("steps = %d, want 2", len(res.Steps))
	}

	// Verify interpolation happened
	if !strings.Contains(runner.prompts[0], "add auth") {
		t.Error("input not interpolated in plan prompt")
	}
	if !strings.Contains(runner.prompts[1], "planned: do A then B") {
		t.Error("plan output not interpolated in impl prompt")
	}
}

func TestRunWorkflowBranch(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner([]runners.RunResult{
		result("TINY: just a typo"), // triage output
		result("fixed the typo"),    // quick-fix output
	}, nil)

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "triage", Model: "fast", Prompt: "Classify: {{input}}", Branch: []Branch{
				{When: "TINY", Goto: "quick"},
				{When: "SMALL", Goto: "plan"},
			}},
			{ID: "brainstorm", Model: "smart", Prompt: "Think about {{input}}"},
			{ID: "plan", Model: "smart", Prompt: "Plan {{input}}"},
			{ID: "quick", Model: "fast", Prompt: "Quick fix: {{input}}"},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "fix typo"})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}

	// Should have skipped brainstorm and plan, jumped to quick
	if runner.callIdx != 2 {
		t.Errorf("runner called %d times, want 2 (triage + quick)", runner.callIdx)
	}
	if _, ok := res.Outputs["brainstorm"]; ok {
		t.Error("brainstorm should have been skipped")
	}
	if _, ok := res.Outputs["quick"]; !ok {
		t.Error("quick-fix should have been executed")
	}
}

func TestRunWorkflowLoop(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner([]runners.RunResult{
		result("attempt 1: still failing"),
		result("attempt 2: ALL_PASSING"),
	}, nil)

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "fix", Model: "smart", Prompt: "Fix tests", Loop: &Loop{Max: 5, Until: "ALL_PASSING"}},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "fix"})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}

	// Should have run 2 iterations (found ALL_PASSING on second)
	if runner.callIdx != 2 {
		t.Errorf("runner called %d times, want 2", runner.callIdx)
	}
	if res.TotalUSD != 0.02 {
		t.Errorf("total cost = %f, want 0.02", res.TotalUSD)
	}
}

func TestRunWorkflowLoopMaxIterations(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner([]runners.RunResult{
		result("still broken"),
		result("still broken"),
		result("still broken"),
	}, nil)

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "fix", Model: "smart", Prompt: "Fix", Loop: &Loop{Max: 3, Until: "DONE"}},
		},
	}

	_, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "fix"})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}

	if runner.callIdx != 3 {
		t.Errorf("runner called %d times, want 3 (max)", runner.callIdx)
	}
}

func TestRunWorkflowBudget(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner([]runners.RunResult{
		{Output: "step 1", CostUSD: 0.50},
		{Output: "step 2", CostUSD: 0.50},
		{Output: "step 3", CostUSD: 0.50}, // should not be reached
	}, nil)

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "s1", Model: "smart", Prompt: "Step 1"},
			{ID: "s2", Model: "smart", Prompt: "Step 2"},
			{ID: "s3", Model: "smart", Prompt: "Step 3"},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test", BudgetUSD: 1.00})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}

	// s1 costs 0.50, s2 costs 0.50, total = 1.00 >= budget, s3 skipped
	if runner.callIdx != 2 {
		t.Errorf("runner called %d times, want 2 (budget hit)", runner.callIdx)
	}
	if res.TotalUSD != 1.00 {
		t.Errorf("total cost = %f, want 1.00", res.TotalUSD)
	}
}

func TestRunWorkflowParallel(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner([]runners.RunResult{
		result("review A findings"),
		result("review B findings"),
	}, nil)

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "review-a", Model: "smart", Prompt: "Review A"},
			{ID: "review-b", Model: "fast", Prompt: "Review B"},
			{ID: "dispatch", Parallel: []string{"review-a", "review-b"}},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "review"})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}

	if _, ok := res.Outputs["review-a"]; !ok {
		t.Error("review-a output missing")
	}
	if _, ok := res.Outputs["review-b"]; !ok {
		t.Error("review-b output missing")
	}
}

func TestRunWorkflowContextCancelled(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner([]runners.RunResult{
		result("step 1 done"),
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "s1", Model: "smart", Prompt: "Step 1"},
		},
	}

	_, err := eng.RunWorkflow(ctx, wf, RunConfig{Input: "test"})
	if err != nil {
		t.Fatalf("RunWorkflow should not error on cancel: %v", err)
	}
	if runner.callIdx != 0 {
		t.Errorf("runner called %d times, want 0 (cancelled)", runner.callIdx)
	}
}

func TestRunWorkflowLoopAllFail(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner(nil, []error{
		fmt.Errorf("runner error 1"),
		fmt.Errorf("runner error 2"),
	})

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "fix", Model: "smart", Prompt: "Fix", Loop: &Loop{Max: 2, Until: "DONE"}},
		},
	}

	_, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "fix"})
	if err == nil {
		t.Fatal("expected error when all loop iterations fail")
	}
	if !strings.Contains(err.Error(), "all 2 iterations failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunWorkflowBranchCycleLimit(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	// Every step output contains "LOOP" which branches back to itself
	responses := make([]runners.RunResult, 150)
	for i := range responses {
		responses[i] = result("LOOP back")
	}
	runner := newMockRunner(responses, nil)

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "start", Model: "fast", Prompt: "Do", Branch: []Branch{
				{When: "LOOP", Goto: "start"},
			}},
		},
	}

	_, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	// Engine should complete (possibly with failed status) but not hang
	_ = err
	// Should have stopped at maxBranches (100), not run forever
	if runner.callIdx > 101 {
		t.Errorf("runner called %d times, expected <= 101 (branch limit)", runner.callIdx)
	}
}

func TestRunWorkflowStepFailure(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner(
		[]runners.RunResult{result("plan done")},
		[]error{nil, fmt.Errorf("implement failed")},
	)

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "plan", Model: "smart", Prompt: "Plan"},
			{ID: "impl", Model: "smart", Prompt: "Implement"},
		},
	}

	_, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	if err == nil {
		t.Fatal("expected error when step fails")
	}
	if !strings.Contains(err.Error(), "impl failed") {
		t.Errorf("error should reference step: %v", err)
	}
}

func TestRunWorkflowParallelPartialFailure(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	// One succeeds, one fails — order depends on goroutine scheduling
	runner := newMockRunner(
		[]runners.RunResult{result("review ok")},
		[]error{nil, fmt.Errorf("review crashed")},
	)

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "ra", Model: "smart", Prompt: "Review A"},
			{ID: "rb", Model: "fast", Prompt: "Review B"},
			{ID: "dispatch", Parallel: []string{"ra", "rb"}},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "review"})
	if err != nil {
		t.Fatalf("partial failure should not error: %v", err)
	}
	// At least one step should have output (we don't know which got the success)
	hasOutput := len(res.Outputs["ra"]) > 0 || len(res.Outputs["rb"]) > 0
	if !hasOutput {
		t.Error("expected at least one parallel step to have output")
	}
}

func TestRunWorkflowParallelAllFail(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner(nil, []error{
		fmt.Errorf("review A failed"),
		fmt.Errorf("review B failed"),
	})

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "ra", Model: "smart", Prompt: "Review A"},
			{ID: "rb", Model: "fast", Prompt: "Review B"},
			{ID: "dispatch", Parallel: []string{"ra", "rb"}},
		},
	}

	_, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "review"})
	if err == nil {
		t.Fatal("expected error when all parallel steps fail")
	}
	if !strings.Contains(err.Error(), "all parallel steps failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunWorkflowBudgetInLoop(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	// Each iteration costs 0.50, budget is 1.00 — should stop after 2
	responses := make([]runners.RunResult, 10)
	for i := range responses {
		responses[i] = runners.RunResult{Output: "still broken", CostUSD: 0.50}
	}
	runner := newMockRunner(responses, nil)

	eng := mustEngine(t, db, git, runner, dir)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "fix", Model: "smart", Prompt: "Fix", Loop: &Loop{Max: 10, Until: "DONE"}},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "fix", BudgetUSD: 1.00})
	_ = err
	// At $0.50/iter with $1.00 budget: 2 iterations run ($1.00), iteration 3 blocked
	if runner.callIdx > 3 {
		t.Errorf("runner called %d times, expected <= 3 (budget should stop loop)", runner.callIdx)
	}
	if res.TotalUSD > 1.50 {
		t.Errorf("total cost $%.2f, expected <= $1.50", res.TotalUSD)
	}
}

// ---------------------------------------------------------------------------
// Parse real workflow files
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Command step tests
// ---------------------------------------------------------------------------

func TestRunWorkflowCommandStep(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	// Runner should NOT be called for command steps
	runner := newMockRunner(nil, nil)
	eng := mustEngine(t, db, git, runner, dir)

	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "check", Command: "echo hello world"},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}

	output, ok := res.Outputs["check"]
	if !ok {
		t.Fatal("command step output missing")
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("output = %q, want it to contain 'hello world'", output)
	}
	if !strings.Contains(output, "exit code: 0") {
		t.Errorf("output should contain exit code, got %q", output)
	}
	// No LLM cost for command steps
	if res.TotalUSD != 0 {
		t.Errorf("total cost = %f, want 0 for command-only workflow", res.TotalUSD)
	}
	// Runner should not have been called
	if runner.callIdx != 0 {
		t.Errorf("runner called %d times, want 0 for command step", runner.callIdx)
	}
}

func TestRunWorkflowCommandInterpolation(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)
	runner := newMockRunner(nil, nil)
	eng := mustEngine(t, db, git, runner, dir)

	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "greet", Command: "echo {{input}}"},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "howdy"})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}
	if !strings.Contains(res.Outputs["greet"], "howdy") {
		t.Errorf("input not interpolated in command output: %q", res.Outputs["greet"])
	}
}

func TestRunWorkflowCommandChainedWithPrompt(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner([]runners.RunResult{
		result("analyzed: found 3 issues"),
	}, nil)
	eng := mustEngine(t, db, git, runner, dir)

	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "lint", Command: "echo 'error: unused var x'"},
			{ID: "fix", Model: "smart", Prompt: "Fix these issues: {{lint}}"},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}

	// Command output should be interpolated into the prompt
	if !strings.Contains(runner.prompts[0], "unused var x") {
		t.Errorf("command output not interpolated into prompt: %q", runner.prompts[0])
	}
	// Only the prompt step should cost money
	if res.TotalUSD != 0.01 {
		t.Errorf("total cost = %f, want 0.01", res.TotalUSD)
	}
}

func TestRunWorkflowCommandNonZeroExit(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)
	runner := newMockRunner(nil, nil)
	eng := mustEngine(t, db, git, runner, dir)

	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "fail", Command: "echo 'errors found'; exit 1"},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}

	// Non-zero exit should NOT be a fatal error — output is still captured
	output := res.Outputs["fail"]
	if !strings.Contains(output, "errors found") {
		t.Errorf("output missing, got %q", output)
	}
	if !strings.Contains(output, "exit code: 1") {
		t.Errorf("exit code not captured, got %q", output)
	}
}

// ---------------------------------------------------------------------------
// Expect field tests
// ---------------------------------------------------------------------------

func TestRunWorkflowExpectFailure(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)
	runner := newMockRunner(nil, nil)
	eng := mustEngine(t, db, git, runner, dir)

	// expect: failure — non-zero exit should succeed (bug repro)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "repro", Command: "exit 1", Expect: "failure"},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	if err != nil {
		t.Fatalf("expected success for non-zero exit with expect:failure, got: %v", err)
	}
	output, ok := res.Outputs["repro"]
	if !ok {
		t.Fatal("repro output missing")
	}
	if !strings.Contains(output, "exit code: 1") {
		t.Errorf("output should contain exit code, got %q", output)
	}
}

func TestRunWorkflowExpectFailureButGotZero(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)
	runner := newMockRunner(nil, nil)
	eng := mustEngine(t, db, git, runner, dir)

	// expect: failure — zero exit should FAIL (bug not reproducible)
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "repro", Command: "exit 0", Expect: "failure"},
		},
	}

	_, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	if err == nil {
		t.Fatal("expected error for exit 0 with expect:failure, got nil")
	}
	if !strings.Contains(err.Error(), "expected failure") {
		t.Errorf("error = %q, want it to contain 'expected failure'", err.Error())
	}
}

func TestRunWorkflowExpectSuccessPass(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)
	runner := newMockRunner(nil, nil)
	eng := mustEngine(t, db, git, runner, dir)

	// expect: success — exit 0 should succeed
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "check", Command: "exit 0", Expect: "success"},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	if err != nil {
		t.Fatalf("expected success for exit 0 with expect:success, got: %v", err)
	}
	if !strings.Contains(res.Outputs["check"], "exit code: 0") {
		t.Error("exit code not in output")
	}
}

func TestRunWorkflowExpectSuccessButGotNonZero(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)
	runner := newMockRunner(nil, nil)
	eng := mustEngine(t, db, git, runner, dir)

	// expect: success — non-zero exit should FAIL
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "check", Command: "exit 1", Expect: "success"},
		},
	}

	_, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	if err == nil {
		t.Fatal("expected error for exit 1 with expect:success, got nil")
	}
	if !strings.Contains(err.Error(), "expected success") {
		t.Errorf("error = %q, want it to contain 'expected success'", err.Error())
	}
}

func TestRunWorkflowNoExpectDefault(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)
	runner := newMockRunner(nil, nil)
	eng := mustEngine(t, db, git, runner, dir)

	// No expect field (default) — non-zero exit is informational, not fatal
	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "check", Command: "exit 1"},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	if err != nil {
		t.Fatalf("default (no expect) should not fail on non-zero exit: %v", err)
	}
	if !strings.Contains(res.Outputs["check"], "exit code: 1") {
		t.Error("exit code not in output")
	}
}

func TestParseExpectField(t *testing.T) {
	yaml := `
name: Expect Test
description: test

steps:
  - id: repro
    command: "npm test"
    expect: failure
  - id: verify
    command: "npm test"
    expect: success
`
	wf, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if wf.Steps[0].Expect != "failure" {
		t.Errorf("step 0 expect = %q, want failure", wf.Steps[0].Expect)
	}
	if wf.Steps[1].Expect != "success" {
		t.Errorf("step 1 expect = %q, want success", wf.Steps[1].Expect)
	}
}

func TestValidateExpectOnPromptStep(t *testing.T) {
	yaml := `
name: Bad
description: test

steps:
  - id: broken
    model: smart
    prompt: "do something"
    expect: failure
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected validation error for expect on prompt step")
	}
	if !strings.Contains(err.Error(), "expect without command") {
		t.Errorf("error = %q, want 'expect without command'", err.Error())
	}
}

func TestValidateExpectInvalidValue(t *testing.T) {
	yaml := `
name: Bad
description: test

steps:
  - id: broken
    command: "echo hi"
    expect: maybe
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected validation error for invalid expect value")
	}
	if !strings.Contains(err.Error(), "invalid expect") {
		t.Errorf("error = %q, want 'invalid expect'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Gate tests
// ---------------------------------------------------------------------------

func TestRunWorkflowLoopGatePass(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner([]runners.RunResult{
		result("fixed something ALL_DONE"),
	}, nil)
	eng := mustEngine(t, db, git, runner, dir)

	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "fix", Model: "smart", Prompt: "Fix", Loop: &Loop{
				Max:   5,
				Until: "ALL_DONE",
				Gate:  "true", // always passes
			}},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}
	// Gate passes, until found on first iteration → 1 call
	if runner.callIdx != 1 {
		t.Errorf("runner called %d times, want 1", runner.callIdx)
	}
	if len(res.Steps) == 0 {
		t.Fatal("expected at least one step")
	}
	if res.Steps[0].Status != "kept" {
		t.Errorf("step status = %q, want kept", res.Steps[0].Status)
	}
}

func TestRunWorkflowLoopGateFail(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	// 3 attempts, all gate-fail, then stuck detection kicks in
	runner := newMockRunner([]runners.RunResult{
		result("attempt 1"),
		result("attempt 2"),
		result("attempt 3"),
	}, nil)
	eng := mustEngine(t, db, git, runner, dir)

	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "fix", Model: "smart", Prompt: "Fix", Loop: &Loop{
				Max:   10,
				Until: "DONE",
				Gate:  "false", // always fails
			}},
		},
	}

	_, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	// All iterations reverted → all failed
	if err == nil {
		t.Fatal("expected error when all iterations are gate-reverted")
	}
	// Should have stopped after 3 consecutive failures
	if runner.callIdx != 3 {
		t.Errorf("runner called %d times, want 3 (stuck detection)", runner.callIdx)
	}
}

func TestRunWorkflowLoopGateRecovery(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	// First attempt fails gate, second passes and hits until
	runner := newMockRunner([]runners.RunResult{
		result("attempt 1"),
		result("attempt 2 ALL_DONE"),
	}, nil)
	eng := mustEngine(t, db, git, runner, dir)

	// Gate: exit 1 on first call, exit 0 on second.
	// Counter file must be OUTSIDE the repo so git revert doesn't reset it.
	counterDir := t.TempDir()
	counterFile := filepath.Join(counterDir, "gate-counter")
	if err := os.WriteFile(counterFile, []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}
	gateScript := fmt.Sprintf(
		`count=$(cat %q); count=$((count + 1)); printf '%%s' "$count" > %q; test "$count" -ge 2`,
		counterFile, counterFile,
	)

	wf := &Workflow{
		Name: "test",
		Steps: []WfStep{
			{ID: "fix", Model: "smart", Prompt: "Fix", Loop: &Loop{
				Max:   5,
				Until: "ALL_DONE",
				Gate:  gateScript,
			}},
		},
	}

	res, err := eng.RunWorkflow(context.Background(), wf, RunConfig{Input: "test"})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}
	// 2 runner calls: first reverted, second kept
	if runner.callIdx != 2 {
		t.Errorf("runner called %d times, want 2", runner.callIdx)
	}

	// Check that we have both reverted and kept steps
	var reverted, kept int
	for _, s := range res.Steps {
		switch s.Status {
		case "reverted":
			reverted++
		case "kept":
			kept++
		}
	}
	if reverted != 1 {
		t.Errorf("reverted steps = %d, want 1", reverted)
	}
	if kept != 1 {
		t.Errorf("kept steps = %d, want 1", kept)
	}
}

func TestParseCommandStep(t *testing.T) {
	yaml := `
name: CmdTest
description: test
steps:
  - id: run
    command: "echo hello"
`
	wf, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if wf.Steps[0].Command != "echo hello" {
		t.Errorf("command = %q, want 'echo hello'", wf.Steps[0].Command)
	}
}

func TestParseLoopGate(t *testing.T) {
	yaml := `
name: GateTest
description: test
steps:
  - id: fix
    model: smart
    prompt: "Fix"
    loop:
      max: 5
      until: DONE
      gate: "npm test"
`
	wf, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if wf.Steps[0].Loop.Gate != "npm test" {
		t.Errorf("gate = %q, want 'npm test'", wf.Steps[0].Loop.Gate)
	}
}

func TestParseRealWorkflows(t *testing.T) {
	workflowDir := filepath.Join("..", "..", "workflows")
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		t.Skip("workflows directory not found:", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(workflowDir, entry.Name())
			wf, err := ParseFile(path)
			if err != nil {
				t.Fatalf("parse %s: %v", entry.Name(), err)
			}
			if wf.Name == "" {
				t.Error("workflow name is empty")
			}
			if len(wf.Steps) == 0 {
				t.Error("workflow has no steps")
			}
		})
	}
}

func TestParseWorkflowNewFields(t *testing.T) {
	yaml := []byte(`
name: test-new-fields
enforce: soft
branch: true
principles: [dry, yagni]
steps:
  - id: step1
    prompt: "do something"
    principles: [clean-code]
`)
	wf, err := Parse(yaml)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if wf.Enforce != "soft" {
		t.Errorf("enforce = %q, want soft", wf.Enforce)
	}
	if !wf.BranchMode {
		t.Error("branch should be true")
	}
	if len(wf.Principles) != 2 || wf.Principles[0] != "dry" {
		t.Errorf("principles = %v, want [dry yagni]", wf.Principles)
	}
	if len(wf.Steps[0].Principles) != 1 || wf.Steps[0].Principles[0] != "clean-code" {
		t.Errorf("step principles = %v, want [clean-code]", wf.Steps[0].Principles)
	}
}
