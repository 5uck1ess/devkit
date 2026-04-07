package runners

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type GeminiRunner struct{}

func (r *GeminiRunner) Name() string { return "gemini" }

func (r *GeminiRunner) Available() bool {
	_, err := exec.LookPath("gemini")
	return err == nil
}

func (r *GeminiRunner) Run(ctx context.Context, prompt string, opts RunOpts) (RunResult, error) {
	// Gemini reads from stdin and -p appends to it. To avoid duplicate
	// instructions, pipe the full prompt via stdin only — no -p flag.
	args := []string{
		"-y",
		"--output-format", "text",
	}

	cmd := exec.CommandContext(ctx, "gemini", args...)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return RunResult{ExitCode: 1}, fmt.Errorf("gemini failed to run: %w", err)
		}
	}

	result := RunResult{
		Output:   stdout.String(),
		ExitCode: exitCode,
	}
	if exitCode != 0 {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		return result, fmt.Errorf("gemini exited %d: %s", exitCode, TruncStr(errMsg, 200))
	}
	return result, nil
}
