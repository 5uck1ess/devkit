package mcp

import (
	"strings"
	"testing"
)

func TestAskMissingArgs(t *testing.T) {
	srv := newTestServer(t, t.TempDir(), t.TempDir())
	_, handler := srv.askTool()

	out, isErr := callToolHandler(t, handler, map[string]interface{}{})
	if !isErr {
		t.Fatalf("expected IsError=true for missing args, got: %s", out)
	}
	if !strings.Contains(out, "missing argument") {
		t.Errorf("expected 'missing argument' in output, got: %s", out)
	}
}

func TestAskUnknownPeer(t *testing.T) {
	srv := newTestServer(t, t.TempDir(), t.TempDir())
	_, handler := srv.askTool()

	out, isErr := callToolHandler(t, handler, map[string]interface{}{
		"to":     "gemini",
		"prompt": "hi",
	})
	if !isErr {
		t.Fatalf("expected IsError=true for unknown peer, got: %s", out)
	}
	if !strings.Contains(out, "unknown peer") {
		t.Errorf("expected 'unknown peer' in output, got: %s", out)
	}
}

func TestAskPeerUnavailable(t *testing.T) {
	// Force the agy binary to be undiscoverable so DetectRunners filters it
	// out and FindRunner returns nil — deterministic regardless of host.
	t.Setenv("AGY_CMD", "definitely-not-a-real-binary-xyz123")

	srv := newTestServer(t, t.TempDir(), t.TempDir())
	_, handler := srv.askTool()

	out, isErr := callToolHandler(t, handler, map[string]interface{}{
		"to":     "agy",
		"prompt": "hi",
	})
	if !isErr {
		t.Fatalf("expected IsError=true for unavailable peer, got: %s", out)
	}
	if !strings.Contains(out, "not available") {
		t.Errorf("expected 'not available' in output, got: %s", out)
	}
}
