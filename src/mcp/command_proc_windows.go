//go:build windows

package mcp

import (
	"os"
	"os/exec"
)

// configureCommandProcess is a no-op on Windows: there is no process
// group equivalent wired up here, so the default Cancel (Kill the shell)
// applies and commandWaitDelay bounds any wait on inherited pipes.
func configureCommandProcess(cmd *exec.Cmd) {}

// killCommandProcessGroup is a no-op on Windows (see configureCommandProcess).
func killCommandProcessGroup(cmd *exec.Cmd) error { return os.ErrProcessDone }
