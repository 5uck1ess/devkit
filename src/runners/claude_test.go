package runners

import (
	"encoding/json"
	"testing"
)

func TestClaudeRunnerName(t *testing.T) {
	r := &ClaudeRunner{}
	if r.Name() != "claude" {
		t.Errorf("got %q, want %q", r.Name(), "claude")
	}
}

func TestCodexRunnerName(t *testing.T) {
	r := &CodexRunner{}
	if r.Name() != "codex" {
		t.Errorf("got %q, want %q", r.Name(), "codex")
	}
}

func TestGeminiRunnerName(t *testing.T) {
	r := &GeminiRunner{}
	if r.Name() != "gemini" {
		t.Errorf("got %q, want %q", r.Name(), "gemini")
	}
}

func TestClaudeResponseParsing(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantOut  string
		wantCost float64
		wantIn   int
		wantSess string
		wantErr  bool
	}{
		{
			name:     "full response",
			json:     `{"result":"hello world","session_id":"sess-123","is_error":false,"total_cost_usd":0.05,"usage":{"input_tokens":100,"output_tokens":50}}`,
			wantOut:  "hello world",
			wantCost: 0.05,
			wantIn:   100,
			wantSess: "sess-123",
		},
		{
			name:     "error response",
			json:     `{"result":"something went wrong","is_error":true,"total_cost_usd":0.01,"usage":{"input_tokens":10,"output_tokens":5}}`,
			wantOut:  "something went wrong",
			wantCost: 0.01,
			wantIn:   10,
		},
		{
			name:     "empty response",
			json:     `{"result":"","session_id":"","total_cost_usd":0,"usage":{"input_tokens":0,"output_tokens":0}}`,
			wantOut:  "",
			wantCost: 0,
			wantIn:   0,
		},
		{
			name:    "invalid json",
			json:    `not json at all`,
			wantErr: true,
		},
		{
			name:     "missing fields",
			json:     `{"result":"partial"}`,
			wantOut:  "partial",
			wantCost: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp claudeResponse
			err := json.Unmarshal([]byte(tt.json), &resp)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected parse error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Result != tt.wantOut {
				t.Errorf("result = %q, want %q", resp.Result, tt.wantOut)
			}
			if resp.TotalCostUSD != tt.wantCost {
				t.Errorf("cost = %f, want %f", resp.TotalCostUSD, tt.wantCost)
			}
			if resp.Usage.InputTokens != tt.wantIn {
				t.Errorf("input tokens = %d, want %d", resp.Usage.InputTokens, tt.wantIn)
			}
			if resp.SessionID != tt.wantSess {
				t.Errorf("session = %q, want %q", resp.SessionID, tt.wantSess)
			}
		})
	}
}

func TestTruncStr_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"empty", "", 10, ""},
		{"exact length", "abcde", 5, "abcde"},
		{"one over", "abcdef", 5, "abcde..."},
		{"zero limit", "abc", 0, "..."},
		{"unicode", "hello 世界！", 7, "hello 世..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncStr(tt.s, tt.n)
			if got != tt.want {
				t.Errorf("TruncStr(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
		})
	}
}

func TestRunOptsDefaults(t *testing.T) {
	opts := RunOpts{}
	if opts.WorkDir != "" {
		t.Error("default WorkDir should be empty")
	}
	if opts.MaxTurns != 0 {
		t.Error("default MaxTurns should be 0")
	}
	if opts.AllowedTools != "" {
		t.Error("default AllowedTools should be empty")
	}
}

func TestRunResultDefaults(t *testing.T) {
	r := RunResult{}
	if r.Output != "" || r.CostUSD != 0 || r.ExitCode != 0 {
		t.Error("default RunResult should have zero values")
	}
}
