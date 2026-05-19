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

func codexExecArgs(workDir string) []string {
	args := []string{"exec", "--sandbox", "workspace-write"}
	if workDir != "" {
		args = append(args, "-C", workDir)
	}
	return append(args, "-")
}

func (r *CodexRunner) Run(ctx context.Context, prompt string, opts RunOpts) (RunResult, error) {
	// `codex exec -` reads initial instructions from stdin.
	// Pipe the full prompt via stdin to handle large diffs safely.
	cmd := exec.CommandContext(ctx, "codex", codexExecArgs(opts.WorkDir)...)
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
			return RunResult{ExitCode: 1}, fmt.Errorf("codex failed to run: %w", err)
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
		return result, fmt.Errorf("codex exited %d: %s", exitCode, TruncStr(errMsg, 200))
	}
	return result, nil
}
