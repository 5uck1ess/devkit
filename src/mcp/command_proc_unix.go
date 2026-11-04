//go:build !windows

package mcp

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// configureCommandProcess runs cmd in its own process group and makes
// context cancellation SIGKILL the whole group, so children that
// inherited the output pipe (npm test workers, `cmd &`, dev servers)
// die with the shell instead of keeping cmd.Wait blocked past the
// deadline.
func configureCommandProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killCommandProcessGroup(cmd) }
}

// killCommandProcessGroup SIGKILLs every process left in cmd's group.
// Returns os.ErrProcessDone when the group is already empty.
func killCommandProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
