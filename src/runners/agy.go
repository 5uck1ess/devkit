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
	// agy --print reads the prompt from stdin; supplying it as a trailing
	// argv leaves stdin empty and the process hangs until --print-timeout
	// (5m default) fires.
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
	// Observed against agy v1.0.0: print-timeout and similar failures exit
	// 0 with a short single-line "Error: ..." on stdout. Treat that as
	// failure; see agyOutputIsError for the heuristic and its bounds (model
	// responses that discuss errors are longer/multi-paragraph and pass).
	if exitCode == 0 && agyOutputIsError(output) {
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

// agyOutputIsError decides whether an exit-0 stdout payload from `agy --print`
// is actually an agy-emitted error rather than a model response.
//
// Genuine agy errors are short, single-line, case-sensitive "Error: ..."
// strings. Model outputs that legitimately discuss errors are typically
// multi-paragraph (contain a blank line) or substantially longer, so those
// pass through.
func agyOutputIsError(output string) bool {
	trimmed := strings.TrimSpace(output)
	if !strings.HasPrefix(trimmed, "Error:") {
		return false
	}
	if len(trimmed) > 300 {
		return false
	}
	if strings.Contains(trimmed, "\n\n") {
		return false
	}
	return true
}
