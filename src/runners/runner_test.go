package runners

import (
	"context"
	"testing"
)

// MockRunner for testing loops without real CLI calls
type MockRunner struct {
	name      string
	responses []RunResult
	errors    []error
	callIdx   int
}

func NewMockRunner(name string, responses []RunResult, errors []error) *MockRunner {
	return &MockRunner{name: name, responses: responses, errors: errors}
}

func (m *MockRunner) Name() string     { return m.name }
func (m *MockRunner) Available() bool  { return true }

func (m *MockRunner) Run(ctx context.Context, prompt string, opts RunOpts) (RunResult, error) {
	idx := m.callIdx
	m.callIdx++
	if idx >= len(m.responses) {
		return RunResult{Output: "mock exhausted"}, nil
	}
	var err error
	if idx < len(m.errors) {
		err = m.errors[idx]
	}
	return m.responses[idx], err
}

func (m *MockRunner) CallCount() int { return m.callIdx }

func TestDetectRunners(t *testing.T) {
	available := DetectRunners()
	for _, r := range available {
		if r.Name() == "" {
			t.Error("runner has empty name")
		}
		if !r.Available() {
			t.Errorf("runner %s reported as available but Available() returns false", r.Name())
		}
	}
}

func TestFindRunner(t *testing.T) {
	runners := []Runner{
		NewMockRunner("claude", nil, nil),
		NewMockRunner("codex", nil, nil),
	}

	found := FindRunner("claude", runners)
	if found == nil || found.Name() != "claude" {
		t.Error("should find claude runner")
	}

	notFound := FindRunner("gemini", runners)
	if notFound != nil {
		t.Error("should not find gemini runner")
	}
}

func TestTruncStr(t *testing.T) {
	if TruncStr("short", 10) != "short" {
		t.Error("short string should not be truncated")
	}
	result := TruncStr("this is a long string", 10)
	if result != "this is a ..." {
		t.Errorf("got %q, want %q", result, "this is a ...")
	}
}

func TestMockRunner(t *testing.T) {
	mock := NewMockRunner("test", []RunResult{
		{Output: "first", CostUSD: 0.01},
		{Output: "second", CostUSD: 0.02},
	}, nil)

	r1, _ := mock.Run(context.Background(), "p1", RunOpts{})
	if r1.Output != "first" {
		t.Errorf("first call output = %q", r1.Output)
	}

	r2, _ := mock.Run(context.Background(), "p2", RunOpts{})
	if r2.Output != "second" {
		t.Errorf("second call output = %q", r2.Output)
	}

	if mock.CallCount() != 2 {
		t.Errorf("call count = %d, want 2", mock.CallCount())
	}
}
