package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/5uck1ess/devkit/lib"
	mcpmcp "github.com/mark3labs/mcp-go/mcp"
)

// newTestServer builds a minimal Server with the given dirs — no DB needed for tool tests.
func newTestServer(t *testing.T, dataDir, workflowDir string) *Server {
	t.Helper()
	return &Server{
		dataDir:     dataDir,
		workflowDir: workflowDir,
		repoRoot:    t.TempDir(),
		db:          nil,
		git:         nil,
		principles:  map[string][]string{},
	}
}

func callTool(t *testing.T, handler func(context.Context, mcpmcp.CallToolRequest) (*mcpmcp.CallToolResult, error)) string {
	t.Helper()
	result, err := handler(context.Background(), mcpmcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil || len(result.Content) == 0 {
		t.Fatal("handler returned empty result")
	}
	// Extract text from first content item.
	if tc, ok := result.Content[0].(mcpmcp.TextContent); ok {
		return tc.Text
	}
	t.Fatalf("unexpected content type: %T", result.Content[0])
	return ""
}

func TestList(t *testing.T) {
	dir := t.TempDir()

	// Write two valid workflow YAMLs and one non-YAML file.
	writeFile(t, filepath.Join(dir, "alpha.yml"), `name: alpha
description: Alpha workflow
steps:
  - id: step-one
    prompt: do something
`)
	writeFile(t, filepath.Join(dir, "beta.yaml"), `name: beta
description: Beta workflow
steps:
  - id: step-one
    prompt: do something
`)
	writeFile(t, filepath.Join(dir, "readme.txt"), "ignore me")

	srv := newTestServer(t, t.TempDir(), dir)
	_, handler := srv.listTool()
	out := callTool(t, handler)

	if !strings.Contains(out, "alpha") {
		t.Errorf("expected 'alpha' in output, got: %s", out)
	}
	if !strings.Contains(out, "Alpha workflow") {
		t.Errorf("expected description 'Alpha workflow' in output, got: %s", out)
	}
	if !strings.Contains(out, "beta") {
		t.Errorf("expected 'beta' in output, got: %s", out)
	}
	if strings.Contains(out, "readme") {
		t.Errorf("non-YAML file should not appear in output, got: %s", out)
	}
}

func TestListParseError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "broken.yml"), "{{{{ not valid yaml ~~~~")

	srv := newTestServer(t, t.TempDir(), dir)
	_, handler := srv.listTool()
	out := callTool(t, handler)

	if !strings.Contains(out, "broken") {
		t.Errorf("expected 'broken' in output, got: %s", out)
	}
	if !strings.Contains(out, "parse error") {
		t.Errorf("expected 'parse error' in output, got: %s", out)
	}
}

func TestListMissingDir(t *testing.T) {
	srv := newTestServer(t, t.TempDir(), "/nonexistent/path/workflows")
	_, handler := srv.listTool()
	result, err := handler(context.Background(), mcpmcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for missing workflow dir")
	}
}

func TestStatusNoSession(t *testing.T) {
	dataDir := t.TempDir()
	srv := newTestServer(t, dataDir, t.TempDir())
	_, handler := srv.statusTool()
	out := callTool(t, handler)

	if !strings.Contains(out, "No active workflow session") {
		t.Errorf("expected no-session message, got: %s", out)
	}
}

func TestStatusWithSession(t *testing.T) {
	dataDir := t.TempDir()

	state := &lib.SessionState{
		ID:           "sess-123",
		Workflow:     "my-workflow",
		CurrentStep:  "step-review",
		CurrentIndex: 2,
		TotalSteps:   5,
		Enforce:      "hard",
		Status:       "running",
		StartedAt:    time.Now(),
		Outputs:      map[string]string{},
	}
	if err := lib.WriteSessionJSON(dataDir, state); err != nil {
		t.Fatalf("write session: %v", err)
	}

	srv := newTestServer(t, dataDir, t.TempDir())
	_, handler := srv.statusTool()
	out := callTool(t, handler)

	checks := []struct {
		field string
		value string
	}{
		{"workflow name", "my-workflow"},
		{"session ID", "sess-123"},
		{"step name", "step-review"},
		{"step progress", "3/5"},
		{"enforce", "hard"},
		{"status", "running"},
	}
	for _, c := range checks {
		if !strings.Contains(out, c.value) {
			t.Errorf("expected %s (%q) in output, got:\n%s", c.field, c.value, out)
		}
	}
}

// writeFile is a test helper that creates a file with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile %s: %v", path, err)
	}
}
