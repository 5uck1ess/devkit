package mcp

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// lockedBuffer is a bytes.Buffer safe for one writer and concurrent
// snapshot reads.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// runDrained starts cmd with stdout+stderr on a pipe we own, waits for the
// shell, then waits up to drainDelay for the pipe to reach EOF.
//
// truncated is true when the pipe was still open drainDelay after the
// shell exited: some leftover descendant inherited it, so the captured
// output is incomplete regardless of the shell's exit code. The caller
// must not judge that output. On truncation the leftover process group
// is killed and the pipe is closed so nothing blocks.
//
// startErr is non-nil only when the command could not be started.
func runDrained(cmd *exec.Cmd, drainDelay time.Duration) (output string, waitErr, startErr error, truncated bool) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return "", nil, err, false
	}
	defer pr.Close()
	cmd.Stdout = pw
	cmd.Stderr = pw

	err = cmd.Start()
	// The child has its own copy of the write end; drop ours so EOF
	// arrives once every holder in the command's tree has exited.
	pw.Close()
	if err != nil {
		return "", nil, err, false
	}

	var out lockedBuffer
	readDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&out, pr)
		close(readDone)
	}()

	waitErr = cmd.Wait()

	timer := time.NewTimer(drainDelay)
	defer timer.Stop()
	select {
	case <-readDone:
		return out.String(), waitErr, nil, false
	case <-timer.C:
	}

	_ = killCommandProcessGroup(cmd)
	// Snapshot before closing: on platforms where Close does not
	// interrupt a blocked Read, the copy goroutine may linger until
	// the remaining writer dies, but we never wait on it.
	output = out.String()
	_ = pr.Close()
	return output, waitErr, nil, true
}
