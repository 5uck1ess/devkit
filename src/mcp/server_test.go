package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewServer(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, "workflows")
	os.MkdirAll(wfDir, 0o755)

	srv, err := NewServer(dir, dir, wfDir)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if srv == nil {
		t.Fatal("server is nil")
	}
}
