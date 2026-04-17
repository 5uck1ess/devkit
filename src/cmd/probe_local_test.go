package cmd

import (
	"context"
	"testing"
	"time"
)

func TestRunProbe_Disabled(t *testing.T) {
	cfg := ProbeConfig{
		Enabled:  false,
		Endpoint: "http://localhost:8080/v1",
		Model:    "test-model",
		Timeout:  1 * time.Second,
	}
	got := runProbe(context.Background(), cfg)

	if got.Enabled {
		t.Errorf("Enabled: got true, want false")
	}
	if got.Reachable {
		t.Errorf("Reachable: got true, want false (disabled should not probe)")
	}
	if got.HTTPStatus != 0 {
		t.Errorf("HTTPStatus: got %d, want 0 (no request should fire)", got.HTTPStatus)
	}
}
