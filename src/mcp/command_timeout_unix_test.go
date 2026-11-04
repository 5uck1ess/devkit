//go:build !windows

package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/5uck1ess/devkit/lib"
)

// processGone reports whether pid has exited (missing or zombie).
func processGone(pid int) bool {
	if err := syscall.Kill(pid, 0); err != nil {
		return true
	}
	stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return true
	}
	// Format: pid (comm) state ...
	if i := strings.LastIndexByte(string(stat), ')'); i >= 0 && i+2 < len(stat) {
		return stat[i+2] == 'Z'
	}
	return false
}

// A timed-out command whose child inherited the output pipe must still
// return 124 promptly, and the child must be killed with it (issue #126).
func TestRunCommandWithTimeout_KillsChildrenOnTimeout(t *testing.T) {
	s := newTestServer(t, t.TempDir(), t.TempDir())
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	state := &lib.SessionState{Input: "", Outputs: map[string]string{}}

	start := time.Now()
	out, code, err := s.runCommandWithTimeout(context.Background(),
		"sleep 10 & echo $! > "+pidFile+"; wait", state, 300*time.Millisecond)
	elapsed := time.Since(start)

	if code != 124 || err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("want exit 124 timeout, got code=%d err=%v out=%q", code, err, out)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("timeout not enforced: returned after %s", elapsed)
	}

	raw, readErr := os.ReadFile(pidFile)
	if readErr != nil {
		t.Fatalf("read child pid: %v", readErr)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	if pid <= 0 {
		t.Fatalf("bad child pid %q", raw)
	}
	deadline := time.Now().Add(2 * time.Second)
	for !processGone(pid) {
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("child %d survived the timeout", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// A command that exits on its own but leaves a background child holding
// the output pipe must not block until the child exits.
func TestRunCommandWithTimeout_DoesNotWaitOnOrphanedPipe(t *testing.T) {
	old := commandWaitDelay
	commandWaitDelay = 200 * time.Millisecond
	t.Cleanup(func() { commandWaitDelay = old })

	s := newTestServer(t, t.TempDir(), t.TempDir())
	state := &lib.SessionState{Input: "", Outputs: map[string]string{}}

	start := time.Now()
	out, code, err := s.runCommandWithTimeout(context.Background(),
		"sleep 10 & echo done", state, time.Minute)
	elapsed := time.Since(start)

	if err != nil || code != 0 {
		t.Fatalf("want exit 0, got code=%d err=%v out=%q", code, err, out)
	}
	if !strings.Contains(out, "done") {
		t.Fatalf("output lost: %q", out)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("blocked on orphaned pipe for %s", elapsed)
	}
}
