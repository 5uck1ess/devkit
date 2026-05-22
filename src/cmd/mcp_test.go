package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// devkit mcp speaks JSON-RPC over stdout. A single byte of non-JSON on
// that stream breaks the handshake (Codex saw "connection closed:
// initialize response"). This test runs the real binary, sends an
// initialize request, and asserts stdout carries only the JSON-RPC
// response.

var (
	binBuildOnce sync.Once
	binBuildPath string
	binBuildErr  error
)

func buildDevkitBinary(t *testing.T) string {
	t.Helper()
	binBuildOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "devkit-mcp-test-*")
		if err != nil {
			binBuildErr = err
			return
		}
		name := "devkit-test"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		binBuildPath = filepath.Join(tmpDir, name)
		// Build from src/ (parent of cmd/). The test's working
		// directory is the package dir, so "../" resolves to src/.
		cmd := exec.Command("go", "build", "-o", binBuildPath, ".")
		cmd.Dir = ".."
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			binBuildErr = errors.New("go build failed: " + err.Error() + ": " + stderr.String())
		}
	})
	if binBuildErr != nil {
		t.Fatalf("build devkit: %v", binBuildErr)
	}
	return binBuildPath
}

// TestMCPStdoutIsCleanJSONRPC is a regression test for the stdout
// contamination bug. Any banner, log line, or stray Println in a code
// path reachable from `devkit mcp` will fail this test by appearing on
// stdout before the JSON-RPC response.
func TestMCPStdoutIsCleanJSONRPC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess build in short mode")
	}

	bin := buildDevkitBinary(t)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":` +
		`{"protocolVersion":"2024-11-05","capabilities":{},` +
		`"clientInfo":{"name":"regression-test","version":"0.0.0"}}}` + "\n"

	// CLAUDE_PLUGIN_ROOT points the server at this repo's workflows
	// dir; without it, NewServer would fall back to repoRoot and the
	// process needs to be inside a git repo (it is — tests run from
	// the package dir which is inside the devkit checkout).
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	cmd := exec.Command(bin, "mcp")
	cmd.Env = append(os.Environ(),
		"CLAUDE_PLUGIN_ROOT="+repoRoot,
		"CLAUDE_PLUGIN_DATA="+t.TempDir(),
	)
	cmd.Stdin = strings.NewReader(initReq)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start devkit mcp: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// The server keeps reading stdin after responding to initialize.
	// Closing stdin (already exhausted) and giving it a bounded wait
	// is enough — if it doesn't exit, kill it.
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("devkit mcp did not exit within 10s\nstdout: %q\nstderr: %q", stdout.String(), stderr.String())
	}

	out := stdout.Bytes()
	if len(out) == 0 {
		t.Fatalf("devkit mcp wrote nothing to stdout\nstderr: %q", stderr.String())
	}
	if out[0] != '{' {
		t.Fatalf("stdout does not start with JSON-RPC object — leading bytes: %q\nfull stdout: %q\nstderr: %q",
			leadingBytes(out, 80), out, stderr.String())
	}

	// Every line on stdout must parse as JSON. A stray Println would
	// land on its own line and fail decoding here.
	dec := json.NewDecoder(bytes.NewReader(out))
	sawInitResp := false
	for {
		var msg map[string]any
		err := dec.Decode(&msg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("stdout contains non-JSON content: %v\nstdout: %q\nstderr: %q", err, out, stderr.String())
		}
		if id, ok := msg["id"]; ok {
			// JSON numbers decode as float64; the initialize id is 1.
			if n, ok := id.(float64); ok && n == 1 {
				sawInitResp = true
			}
		}
	}
	if !sawInitResp {
		t.Fatalf("did not see initialize response on stdout\nstdout: %q\nstderr: %q", out, stderr.String())
	}

	// The diagnostic banner belongs on stderr. This is the affirmative
	// half of the contract — if someone deletes the banner, that's
	// fine; if they move it back to stdout, the JSON check above
	// fails first.
	if strings.Contains(stdout.String(), "devkit MCP server ready") {
		t.Fatalf("ready banner leaked to stdout — must go to stderr\nstdout: %q", stdout.String())
	}
}

func leadingBytes(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}

// TestMCPInitializeOutsideGitRepo is a regression test for issue #105.
// Before the fix, devkit mcp aborted at startup whenever the cwd was not
// inside a git repo — the JSON-RPC initialize handshake never completed
// and Claude Code only surfaced the opaque `-32000` server error. The MCP
// server must boot regardless of cwd; tools that genuinely need git state
// return structured errors on call, not by exiting at boot.
func TestMCPInitializeOutsideGitRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess build in short mode")
	}

	bin := buildDevkitBinary(t)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":` +
		`{"protocolVersion":"2024-11-05","capabilities":{},` +
		`"clientInfo":{"name":"regression-test","version":"0.0.0"}}}` + "\n"

	// Resolve CLAUDE_PLUGIN_ROOT to this repo so workflows still load —
	// in production the launcher always sets this env var. The point of
	// this test is that the cwd is OUTSIDE any git repo.
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	nonGitDir := t.TempDir()
	if _, err := os.Stat(filepath.Join(nonGitDir, ".git")); err == nil {
		t.Fatalf("tempdir unexpectedly contained .git — test premise broken")
	}

	cmd := exec.Command(bin, "mcp")
	cmd.Dir = nonGitDir
	cmd.Env = append(os.Environ(),
		"CLAUDE_PLUGIN_ROOT="+repoRoot,
		"CLAUDE_PLUGIN_DATA="+t.TempDir(),
	)
	cmd.Stdin = strings.NewReader(initReq)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start devkit mcp: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("devkit mcp did not exit within 10s\nstdout: %q\nstderr: %q", stdout.String(), stderr.String())
	}

	if strings.Contains(stderr.String(), "not inside a git repo") {
		t.Fatalf("devkit mcp aborted with git-repo check — boot must tolerate non-git cwds\nstderr: %q", stderr.String())
	}

	out := stdout.Bytes()
	if len(out) == 0 {
		t.Fatalf("devkit mcp wrote nothing to stdout — initialize handshake never completed\nstderr: %q", stderr.String())
	}

	dec := json.NewDecoder(bytes.NewReader(out))
	sawInitResp := false
	for {
		var msg map[string]any
		err := dec.Decode(&msg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("stdout contains non-JSON content: %v\nstdout: %q\nstderr: %q", err, out, stderr.String())
		}
		if id, ok := msg["id"]; ok {
			if n, ok := id.(float64); ok && n == 1 {
				sawInitResp = true
			}
		}
	}
	if !sawInitResp {
		t.Fatalf("did not see initialize response on stdout — handshake failed outside a git repo\nstdout: %q\nstderr: %q", out, stderr.String())
	}
}
