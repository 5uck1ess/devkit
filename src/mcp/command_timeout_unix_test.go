//go:build !windows

package mcp

import (
	"context"
	"os"
	"os/exec"
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

// A command that exits 0 but leaves a background child holding the output
// pipe must not block until the child exits — and must not report success,
// because the captured output is incomplete: the child could still print
// FAIL after the PASS we saw, and a last-line check would wrongly pass.
//
// The shell's exit code must not matter: with `exit 1`, an `expect: failure`
// step would otherwise accept the truncated PASS.
func TestRunCommandWithTimeout_FailsClosedOnOrphanedPipe(t *testing.T) {
	for name, tail := range map[string]string{
		"exit0": "echo PASS",
		"exit1": "echo PASS; exit 1",
	} {
		t.Run(name, func(t *testing.T) { assertFailsClosedOnOrphanedPipe(t, tail) })
	}
}

func assertFailsClosedOnOrphanedPipe(t *testing.T, tail string) {
	old := commandWaitDelay
	commandWaitDelay = 200 * time.Millisecond
	t.Cleanup(func() { commandWaitDelay = old })

	s := newTestServer(t, t.TempDir(), t.TempDir())
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	state := &lib.SessionState{Input: "", Outputs: map[string]string{}}

	start := time.Now()
	out, code, err := s.runCommandWithTimeout(context.Background(),
		"(sleep 10; echo FAIL) & echo $! > "+pidFile+"; "+tail, state, time.Minute)
	elapsed := time.Since(start)

	if err == nil || code == 0 {
		t.Fatalf("truncated output reported as success: code=%d err=%v out=%q", code, err, out)
	}
	if !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("error should say output is incomplete: %v", err)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("blocked on orphaned pipe for %s", elapsed)
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
			t.Fatalf("leftover child %d survived the step", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// A command whose background job does not hold the output pipe is
// unaffected: it succeeds immediately with full output.
func TestRunCommandWithTimeout_RedirectedBackgroundJobSucceeds(t *testing.T) {
	s := newTestServer(t, t.TempDir(), t.TempDir())
	state := &lib.SessionState{Input: "", Outputs: map[string]string{}}

	start := time.Now()
	out, code, err := s.runCommandWithTimeout(context.Background(),
		"sleep 10 >/dev/null 2>&1 & echo PASS", state, time.Minute)
	elapsed := time.Since(start)

	if err != nil || code != 0 || !strings.Contains(out, "PASS") {
		t.Fatalf("want exit 0 with PASS, got code=%d err=%v out=%q", code, err, out)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("redirected background job delayed the step by %s", elapsed)
	}
}

// The direct process can leave its process group before the deadline;
// the group kill then misses it, so the timeout must also kill it
// directly or Wait blocks until it exits on its own.
func TestRunCommandWithTimeout_KillsProcessThatLeftGroup(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	s := newTestServer(t, t.TempDir(), t.TempDir())
	state := &lib.SessionState{Input: "", Outputs: map[string]string{}}

	start := time.Now()
	out, code, err := s.runCommandWithTimeout(context.Background(),
		`exec python3 -c 'import os,time; os.setpgid(0,os.getpgid(os.getppid())); time.sleep(10)'`,
		state, 500*time.Millisecond)
	elapsed := time.Since(start)

	if code != 124 || err == nil {
		t.Fatalf("want exit 124 timeout, got code=%d err=%v out=%q", code, err, out)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("timeout not enforced after group escape: returned after %s", elapsed)
	}
}
