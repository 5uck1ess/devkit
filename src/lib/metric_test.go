package lib

import (
	"context"
	"testing"
)

func TestRunMetricSuccess(t *testing.T) {
	result := RunMetric(context.Background(), "echo hello", t.TempDir())
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if result.Output == "" {
		t.Error("output should not be empty")
	}
}

func TestRunMetricFailure(t *testing.T) {
	result := RunMetric(context.Background(), "exit 1", t.TempDir())
	if result.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", result.ExitCode)
	}
}

func TestRunMetricTruncation(t *testing.T) {
	// Generate output larger than 4096 bytes
	result := RunMetric(context.Background(), "head -c 5000 /dev/zero | tr '\\0' 'a'", t.TempDir())
	if len(result.Output) > 4200 {
		t.Errorf("output should be truncated, got %d bytes", len(result.Output))
	}
}

func TestRunMetricCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := RunMetric(ctx, "sleep 10", t.TempDir())
	if result.ExitCode == 0 {
		t.Error("cancelled command should have non-zero exit code")
	}
}
