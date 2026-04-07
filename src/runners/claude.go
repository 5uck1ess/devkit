package runners

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type ClaudeRunner struct{}

type claudeResponse struct {
	Result       string  `json:"result"`
	SessionID    string  `json:"session_id"`
	IsError      bool    `json:"is_error"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (r *ClaudeRunner) Name() string { return "claude" }

func (r *ClaudeRunner) Available() bool {
	if _, err := exec.LookPath("claude"); err != nil {
		return false
	}
	// OAuth tokens (sk-ant-oat*) don't work for subprocess claude -p calls.
	// Only real API keys (sk-ant-api*) or no key (keychain auth) work.
	key := os.Getenv("ANTHROPIC_API_KEY")
	if strings.HasPrefix(key, "sk-ant-oat") {
		fmt.Fprintf(os.Stderr, "claude: skipping — ANTHROPIC_API_KEY is an OAuth token (sk-ant-oat*), which doesn't work for subprocess calls\n")
		return false
	}
	return true
}

func (r *ClaudeRunner) Run(ctx context.Context, prompt string, opts RunOpts) (RunResult, error) {
	// claude -p is "print mode" — the prompt is a positional argument.
	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--no-session-persistence",
	}
	if opts.AllowedTools != "" {
		args = append(args, "--allowedTools", opts.AllowedTools)
	}
	if opts.AppendSystemPromptFile != "" {
		args = append(args, "--append-system-prompt-file", opts.AppendSystemPromptFile)
	}
	if opts.AppendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", opts.AppendSystemPrompt)
	}
	if opts.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(opts.MaxTurns))
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return RunResult{ExitCode: 1}, fmt.Errorf("claude failed to run: %w", err)
		}
	}

	var resp claudeResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return RunResult{Output: stdout.String(), ExitCode: exitCode},
			fmt.Errorf("claude returned non-JSON output: %s", TruncStr(stdout.String(), 200))
	}

	return RunResult{
		Output:    resp.Result,
		SessionID: resp.SessionID,
		TokensIn:  resp.Usage.InputTokens,
		TokensOut: resp.Usage.OutputTokens,
		CostUSD:   resp.TotalCostUSD,
		ExitCode:  exitCode,
	}, nil
}
