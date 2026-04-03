package lib

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

type MetricResult struct {
	ExitCode int
	Output   string
	Duration time.Duration
}

func RunMetric(ctx context.Context, command string, dir string) MetricResult {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	output := stdout.String()
	if output == "" {
		output = stderr.String()
	}

	// Truncate to avoid bloating state
	const maxOutput = 4096
	if len(output) > maxOutput {
		output = output[:maxOutput] + "\n... (truncated)"
	}

	return MetricResult{
		ExitCode: exitCode,
		Output:   output,
		Duration: duration,
	}
}
