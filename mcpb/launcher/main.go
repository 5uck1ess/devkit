// Package main is the devkit MCPB Windows launcher.
//
// This binary is what plugin.json's mcpb platform_overrides.win32 points at,
// i.e. it's what Claude Code CreateProcess()'s on Windows when spawning the
// devkit MCP server. Same contract as the POSIX bin/devkit shell wrapper —
// read plugin.json, fetch the engine from the matching GitHub release,
// verify SHA-256, exec — but the Windows path is simpler: single-shot HTTP
// fetch, no resume, no downloader fallback chain. A killed launcher
// mid-download leaves no state; the next run starts over.
//
// Go stdlib only, so the static build stays small and reproducible. Go's
// crypto/tls handles the HTTPS fetch, not Windows schannel, which sidesteps
// the renegotiation abort (CURLE_WRITE_ERROR / exit 23) that curl.exe hits
// mid-stream on release-assets.githubusercontent.com.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	releaseOwner = "5uck1ess"
	releaseRepo  = "devkit"

	// Cap a single HTTP fetch; past this ceiling a slow connection is a
	// stuck one.
	httpTimeout = 5 * time.Minute

	// Cap plugin.json reads. A legitimate manifest is under a kilobyte;
	// anything above this ceiling is either corrupted or hostile.
	maxPluginJSONBytes = 64 * 1024
)

func main() {
	// Stdout is reserved for the MCP stdio transport once execEngine runs;
	// pin the stdlib log package to stderr so a future refactor can't leak
	// diagnostics into the JSON-RPC framing.
	log.SetOutput(os.Stderr)

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "devkit launcher: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	pluginRoot := os.Getenv("CLAUDE_PLUGIN_ROOT")
	if pluginRoot == "" {
		return errors.New("CLAUDE_PLUGIN_ROOT is not set; this launcher must be invoked by Claude Code as an MCP server")
	}

	// CLAUDE_PLUGIN_ROOT must be absolute. CC always sets it that way;
	// anything else is a corrupted env or a spoofed launch.
	if !filepath.IsAbs(pluginRoot) {
		return fmt.Errorf("CLAUDE_PLUGIN_ROOT is not absolute: %q", pluginRoot)
	}

	pluginJSON := filepath.Join(pluginRoot, ".claude-plugin", "plugin.json")
	version, err := readPluginVersion(pluginJSON)
	if err != nil {
		return fmt.Errorf("reading plugin version from %s: %w", pluginJSON, err)
	}
	// version is interpolated into a filename joined to binDir below. Reject
	// anything outside a narrow charset so a corrupted or spoofed plugin.json
	// can't produce a path that escapes binDir via "..", a path separator,
	// or shell metacharacters.
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("invalid plugin version %q: %w", version, err)
	}

	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	engineName := fmt.Sprintf("devkit-engine-v%s-%s.exe", version, platform)
	binDir := filepath.Join(pluginRoot, "bin")
	enginePath := filepath.Join(binDir, engineName)

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", binDir, err)
	}

	cached, err := engineLooksCached(enginePath)
	if err != nil {
		return fmt.Errorf("checking engine cache at %s: %w", enginePath, err)
	}
	if !cached {
		logf("first-run: downloading engine v%s (%s)...", version, platform)
		if err := ensureEngine(version, platform, enginePath); err != nil {
			return fmt.Errorf("downloading engine: %w", err)
		}
		logf("installed engine at %s", enginePath)
	}

	// Best-effort sweep of engines from other versions; never fatal.
	if err := sweepStaleEngines(binDir, engineName); err != nil {
		logf("warning: could not sweep stale engines: %v", err)
	}

	return execEngine(enginePath, os.Args[1:])
}

// logf writes to stderr with a "devkit launcher:" prefix. Stdout is
// reserved for the MCP stdio transport once the engine takes over; the
// launcher must never write to it.
func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "devkit launcher: "+format+"\n", args...)
}

// versionPattern accepts digits + dots + an optional single pre-release
// identifier. Narrower than semver on purpose — no "+build" metadata, no
// hyphens inside the pre-release — so the result is safe to join onto a
// filename.
var versionPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+){0,3}(-[A-Za-z0-9.]+)?$`)

func validateVersion(version string) error {
	if version == "" {
		return errors.New("empty")
	}
	if !versionPattern.MatchString(version) {
		return errors.New("must match ^[0-9]+(\\.[0-9]+){0,3}(-[A-Za-z0-9.]+)?$")
	}
	// Defense in depth: even inside the charset, reject ".." and path
	// separators. If a future refactor widens versionPattern these guards
	// still hold — they are load-bearing for the filesystem boundary and are
	// enforced by table tests in main_test.go.
	if strings.Contains(version, "..") || strings.ContainsAny(version, `/\`) {
		return errors.New("contains path traversal sequence or separator")
	}
	return nil
}

func readPluginVersion(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	// Reject oversize before reading. io.LimitReader would silently
	// truncate, leaving a prefix that may happen to parse as valid JSON
	// and return the wrong version — stat-first makes truncation a hard
	// fail instead of a silent prefix parse.
	if info.Size() > maxPluginJSONBytes {
		return "", fmt.Errorf("plugin.json is %d bytes, max %d", info.Size(), maxPluginJSONBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", err
	}
	if manifest.Version == "" {
		return "", errors.New("version field missing or empty")
	}
	return manifest.Version, nil
}

// engineLooksCached reports whether enginePath is present and non-empty.
// Size-only check matches bin/devkit's POSIX semantics; stat errors other
// than IsNotExist are surfaced so ACL/permission issues don't masquerade
// as a cache miss and trigger spurious redownloads on every invocation.
func engineLooksCached(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() || info.Size() == 0 {
		return false, nil
	}
	return true, nil
}

// ensureEngine fetches the engine asset from the matching GitHub release,
// verifies SHA-256, and atomic-installs the binary at enginePath. The
// .sums.tmp and .partial staging files are cleaned up on every exit path.
func ensureEngine(version, platform, enginePath string) error {
	tag := "v" + version
	baseURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s", releaseOwner, releaseRepo, tag)

	sumsPath := enginePath + ".sums.tmp"
	defer os.Remove(sumsPath)
	if err := downloadTo(baseURL+"/checksums.txt", sumsPath); err != nil {
		return fmt.Errorf("fetching checksums.txt: %w", err)
	}

	assetName := fmt.Sprintf("devkit-%s.exe", platform)
	expected, err := findChecksum(sumsPath, assetName)
	if err != nil {
		return fmt.Errorf("finding checksum for %s: %w", assetName, err)
	}

	stagingPath := enginePath + ".partial"
	defer os.Remove(stagingPath)
	if err := downloadTo(baseURL+"/"+assetName, stagingPath); err != nil {
		return fmt.Errorf("fetching %s: %w", assetName, err)
	}

	actual, err := sha256File(stagingPath)
	if err != nil {
		return fmt.Errorf("checksumming %s: %w", assetName, err)
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", assetName, expected, actual)
	}

	// os.Rename is atomic on Windows within the same volume. enginePath and
	// stagingPath are both in binDir, so always same volume.
	if err := os.Rename(stagingPath, enginePath); err != nil {
		return fmt.Errorf("installing %s -> %s: %w", stagingPath, enginePath, err)
	}
	return nil
}

func downloadTo(url, destPath string) (retErr error) {
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	// Capture a late Close error so antivirus quarantine-on-close, disk
	// quota, and SMB sync-on-close failures don't manifest only as a
	// mysterious checksum mismatch later. Must join with an existing
	// retErr so we don't mask the primary failure — io.Copy or Sync
	// errors win the "root cause" slot, Close errors are appended.
	defer func() {
		if cerr := out.Close(); cerr != nil {
			cerr = fmt.Errorf("closing %s: %w", destPath, cerr)
			if retErr == nil {
				retErr = cerr
			} else {
				retErr = errors.Join(retErr, cerr)
			}
		}
	}()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	// A server advertising Content-Length must deliver it; a silent
	// short read would otherwise only surface as a SHA-256 mismatch
	// downstream (and for checksums.txt there is no SHA to verify
	// against, so truncation would masquerade as "no entry").
	if resp.ContentLength >= 0 && n != resp.ContentLength {
		return fmt.Errorf("short read from %s: got %d bytes, expected %d", url, n, resp.ContentLength)
	}
	// Chunked / identity responses with no Content-Length fall through
	// to the engine's SHA-256 verify, but an empty body from either is
	// always wrong — reject it up front.
	if n == 0 {
		return fmt.Errorf("empty response from %s", url)
	}
	if err := out.Sync(); err != nil {
		return err
	}
	return nil
}

// findChecksum parses a sha256sum-format file and returns the hash for
// assetName. Two columns per line: hash, filename. The filename may have
// a leading "*" for binary mode (per GNU coreutils sha256sum), which we
// trim before comparing.
func findChecksum(sumsFile, assetName string) (string, error) {
	data, err := os.ReadFile(sumsFile)
	if err != nil {
		return "", err
	}
	// Normalize CRLF so a Windows-produced checksums.txt doesn't leave a
	// trailing \r on the filename field and silently miss the match.
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum entry for %s in checksums.txt", assetName)
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// sweepStaleEngines removes engine binaries from other versions that are
// sitting in binDir. Best-effort: a few stale megabytes on disk is strictly
// better than failing to start the engine. The prefix match deliberately
// skips the current engine and ignores any name that doesn't start with
// "devkit-engine-v" or "devkit-checksums-v".
func sweepStaleEngines(binDir, currentEngineName string) error {
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == currentEngineName {
			continue
		}
		if strings.HasPrefix(name, "devkit-engine-v") || strings.HasPrefix(name, "devkit-checksums-v") {
			_ = os.Remove(filepath.Join(binDir, name))
		}
	}
	return nil
}

// execEngine runs the engine binary with inherited stdio and forwards the
// exit code. Windows has no execve, so the launcher stays alive as the
// parent process until the engine exits. stdin/stdout/stderr are assigned
// as *os.File values, which os/exec passes directly to CreateProcess via
// STARTUPINFO handles — no pipe, no goroutine copy, no buffering. MCP
// JSON-RPC framing is untouched by the launcher.
//
// On a non-ExitError failure (CreateProcess rejected the cached binary —
// wrong architecture, truncated PE, missing dependent DLL), the cached
// engine is removed so the next launch self-heals by re-downloading
// instead of getting stuck in a loop. A failing os.Remove is surfaced in
// the error so "cache purged" is never a lie.
func execEngine(enginePath string, args []string) error {
	cmd := exec.Command(enginePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		if code < 0 {
			// Abnormal termination (signal, job kill). Windows maps -1 to
			// 0xFFFFFFFF via os.Exit, which CC's MCP manager can't read.
			code = 1
		}
		os.Exit(code)
	}
	if rmErr := os.Remove(enginePath); rmErr != nil {
		return fmt.Errorf("executing engine %s (cache purge also failed: %v): %w", enginePath, rmErr, err)
	}
	return fmt.Errorf("executing engine %s (cache purged for next run): %w", enginePath, err)
}
