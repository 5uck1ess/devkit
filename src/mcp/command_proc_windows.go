//go:build windows

package mcp

import "os/exec"

// configureCommandProcess is a no-op on Windows: there is no process
// group equivalent wired up here, so the default Cancel (Kill the shell)
// applies and commandWaitDelay bounds any wait on inherited pipes.
func configureCommandProcess(cmd *exec.Cmd) {}
