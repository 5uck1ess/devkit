package runners

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// AgyRunner dispatches to the Google Antigravity CLI ("agy") for a single
// headless request/response. It mirrors CodexRunner: the prompt is piped via
// stdin, stdout is buffered and returned verbatim, and a non-zero exit is
// surfaced with truncated stderr.
//
// The agy CLI's non-interactive invocation is NOT verified in this repo (agy
// is not installed on the build host). Both the binary name and the argv are
// therefore overridable via environment so an operator can wire in the real
// flags without recompiling:
//
//	AGY_CMD   — binary name/path (default: "agy"), used by both Available()
//	            and Run().
//	AGY_ARGS  — space-separated argv template that REPLACES the built-in
//	            default. Use the literal token {workdir} where the working
//	            directory should be substituted; the token is dropped when no
//	            WorkDir is set. cmd.Dir is still applied from WorkDir
//	            regardless. Example:
//	              AGY_ARGS="exec --cwd {workdir} -"
type AgyRunner struct{}

func (r *AgyRunner) Name() string { return "agy" }

// agyCmd resolves the agy binary name (AGY_CMD override, default "agy").
func agyCmd() string { return envDefault("AGY_CMD", "agy") }

func (r *AgyRunner) Available() bool {
	_, err := exec.LookPath(agyCmd())
	return err == nil
}

// agyExecArgs builds the argv for a one-shot, stdin-fed request.
//
// TODO(agy): the default below is a PLACEHOLDER modeled on the codex shape
// (`agy exec [-C dir] -`, prompt on stdin). It has NOT been validated against
// a real agy CLI. Before relying on agy as a bridge target, run these probes
// on a host with agy installed and either update this default or set AGY_ARGS:
//
//	agy --version
//	agy --help          # find the non-interactive / exec / run subcommand
//	agy exec --help     # or `agy run --help`: how to read the prompt from
//	                    # stdin, set the working dir, force headless mode, and
//	                    # emit plain (non-streaming) text
//	agy models          # confirm the default model / whether --model is required
func agyExecArgs(workDir string) []string {
	if raw := strings.TrimSpace(os.Getenv("AGY_ARGS")); raw != "" {
		fields := strings.Fields(raw)
		args := make([]string, 0, len(fields))
		for _, f := range fields {
			if f == "{workdir}" {
				if workDir != "" {
					args = append(args, workDir)
				}
				continue
			}
			args = append(args, f)
		}
		return args
	}
	// TODO(agy): PLACEHOLDER default — verify against `agy --help`.
	args := []string{"exec"}
	if workDir != "" {
		args = append(args, "-C", workDir)
	}
	return append(args, "-")
}

func (r *AgyRunner) Run(ctx context.Context, prompt string, opts RunOpts) (RunResult, error) {
	// `agy` reads the prompt from stdin (via the trailing "-"); piping avoids
	// arg-length limits on large prompts/diffs, matching codex/gemini.
	cmd := exec.CommandContext(ctx, agyCmd(), agyExecArgs(opts.WorkDir)...)
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

	result := RunResult{
		Output:   stdout.String(),
		ExitCode: exitCode,
	}
	if exitCode != 0 {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		return result, fmt.Errorf("agy exited %d: %s", exitCode, TruncStr(errMsg, 200))
	}
	return result, nil
}
