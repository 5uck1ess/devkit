package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
}

func newMockRunner(responses []runners.RunResult, errs []error) *mockRunner {
	return &mockRunner{name: "mock", responses: responses, errors: errs}
}

func (m *mockRunner) Name() string    { return m.name }
func (m *mockRunner) Available() bool { return true }

func (m *mockRunner) Run(ctx context.Context, prompt string, opts runners.RunOpts) (runners.RunResult, error) {
	m.prompts = append(m.prompts, prompt)
	idx := m.callIdx
	m.callIdx++
	if idx >= len(m.responses) {
		return runners.RunResult{Output: "mock exhausted"}, nil
	}
	var err error
	if idx < len(m.errors) {
		err = m.errors[idx]
	}
	return m.responses[idx], err
}

func result(output string) runners.RunResult {
	return runners.RunResult{Output: output, CostUSD: 0.01}
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
		{"tiny change", "quick"},      // case insensitive
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

func TestRunWorkflowSimple(t *testing.T) {
	db := tempDB(t)
	dir, git := initGitRepo(t)

	runner := newMockRunner([]runners.RunResult{
		result("planned: do A then B"),
		result("implemented A and B"),
	}, nil)

	eng := &Engine{DB: db, Git: git, Runner: runner, RepoRoot: dir}
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
		result("TINY: just a typo"),  // triage output
		result("fixed the typo"),     // quick-fix output
	}, nil)

	eng := &Engine{DB: db, Git: git, Runner: runner, RepoRoot: dir}
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

	eng := &Engine{DB: db, Git: git, Runner: runner, RepoRoot: dir}
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

	eng := &Engine{DB: db, Git: git, Runner: runner, RepoRoot: dir}
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

	eng := &Engine{DB: db, Git: git, Runner: runner, RepoRoot: dir}
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

	eng := &Engine{DB: db, Git: git, Runner: runner, RepoRoot: dir}
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

	eng := &Engine{DB: db, Git: git, Runner: runner, RepoRoot: dir}
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

// ---------------------------------------------------------------------------
// Parse real workflow files
// ---------------------------------------------------------------------------

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
