//go:build !windows

package lib

import (
	"os"

	"golang.org/x/sys/unix"
)

// lockFile takes an exclusive advisory flock on f. Blocks until granted.
// Released automatically when the fd is closed, but callers should pair
// with unlockFile on exit to be explicit.
func lockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX)
}

// unlockFile releases an advisory flock held on f. Errors are ignored
// by callers because Close() would release the lock anyway; returning
// the error lets tests assert on it.
func unlockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
