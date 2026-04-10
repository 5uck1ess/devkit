package lib

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func NewSessionID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		// Fallback: fixed-length 12-char hex from nanosecond timestamp
		return fmt.Sprintf("%012x", time.Now().UnixNano())[:12]
	}
	return hex.EncodeToString(b)
}

func SessionDir(repoRoot, sessionID string) string {
	return filepath.Join(repoRoot, ".devkit", "sessions", sessionID)
}

func EnsureSessionDir(repoRoot, sessionID string) error {
	dir := SessionDir(repoRoot, sessionID)
	return os.MkdirAll(dir, 0o700)
}
