package runners

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type CodexRunner struct{}

func (r *CodexRunner) Name() string { return "codex" }

func (r *CodexRunner) Available() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

func (r *CodexRunner) Run(ctx context.Context, prompt string, opts RunOpts) (RunResult, error) {
	// Codex reads additional input from stdin automatically.
	// Pass a short instruction as the argument, pipe the full prompt via stdin.
	args := []string{"exec", "--full-auto", "Follow the instructions provided on stdin."}

	cmd := exec.CommandContext(ctx, "codex", args...)
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
			return RunResult{ExitCode: 1}, fmt.Errorf("codex failed to start: %w", err)
		}
	}

	return RunResult{
		Output:   stdout.String(),
		ExitCode: exitCode,
	}, nil
}
