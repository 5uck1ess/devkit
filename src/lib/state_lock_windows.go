//go:build windows

package lib

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockFile takes an exclusive lock on the first byte of f via
// LockFileEx, which is Windows' mandatory equivalent of Unix advisory
// flock for the purposes we use it (serializing session.json writers
// across processes). Blocks until granted.
func lockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0, 1, 0, ol,
	)
}

// unlockFile releases a lock previously taken by lockFile.
func unlockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.UnlockFileEx(
		windows.Handle(f.Fd()),
		0, 1, 0, ol,
	)
}
