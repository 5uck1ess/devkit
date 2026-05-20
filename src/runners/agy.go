package runners

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type AgyRunner struct{}

func (r *AgyRunner) Name() string { return "agy" }

func (r *AgyRunner) Available() bool {
	_, err := exec.LookPath("agy")
	return err == nil
}

func (r *AgyRunner) Run(ctx context.Context, prompt string, opts RunOpts) (RunResult, error) {
	// agy reads the prompt from stdin in --print mode; passing the prompt as
	// argv after --print is not used and causes a print-timeout error.
	args := []string{"--print", "--dangerously-skip-permissions"}

	cmd := exec.CommandContext(ctx, "agy", args...)
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
			return RunResult{ExitCode: 1}, fmt.Errorf("agy failed to run: %w", err)
		}
	}

	output := stdout.String()
	result := RunResult{Output: output, ExitCode: exitCode}
	// agy exits 0 on internal errors (e.g. print timeout) and prints
	// "Error: ..." on stdout — surface those as failures rather than
	// returning the error string as a successful response.
	if exitCode == 0 && strings.HasPrefix(strings.TrimSpace(output), "Error:") {
		return result, fmt.Errorf("agy reported error: %s", TruncStr(strings.TrimSpace(output), 200))
	}
	if exitCode != 0 {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = output
		}
		return result, fmt.Errorf("agy exited %d: %s", exitCode, TruncStr(errMsg, 200))
	}
	return result, nil
}
