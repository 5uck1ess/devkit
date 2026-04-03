package runners

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type ClaudeRunner struct{}

type claudeResponse struct {
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
	IsError   bool   `json:"is_error"`
	Usage     struct {
		InputTokens  int     `json:"input_tokens"`
		OutputTokens int     `json:"output_tokens"`
		CostUSD      float64 `json:"cost_usd"`
	} `json:"usage"`
}

func (r *ClaudeRunner) Name() string { return "claude" }

func (r *ClaudeRunner) Available() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (r *ClaudeRunner) Run(ctx context.Context, prompt string, opts RunOpts) (RunResult, error) {
	args := []string{
		"--bare",
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
			return RunResult{ExitCode: 1}, fmt.Errorf("claude failed to start: %w — is claude CLI installed?", err)
		}
	}

	var resp claudeResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		// If JSON parsing fails, return raw output
		return RunResult{
			Output:   stdout.String(),
			ExitCode: exitCode,
		}, nil
	}

	return RunResult{
		Output:    resp.Result,
		SessionID: resp.SessionID,
		TokensIn:  resp.Usage.InputTokens,
		TokensOut: resp.Usage.OutputTokens,
		CostUSD:   resp.Usage.CostUSD,
		ExitCode:  exitCode,
	}, nil
}
