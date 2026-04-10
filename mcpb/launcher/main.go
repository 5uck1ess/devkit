// devkit MCPB Windows launcher — real v2.1.7 launcher.
//
// Replaces the probe stub. This binary is what plugin.json's mcpb
// platform_overrides.win32 points at, i.e. it's what Claude Code actually
// CreateProcess()'s on Windows when spawning the devkit MCP server. It
// duplicates the responsibilities of the POSIX bin/devkit shell wrapper,
// so on Windows the download/verify/exec flow is identical in behavior
// even though the code is Go instead of /bin/sh.
//
// Flow on every invocation:
//
//  1. Read CLAUDE_PLUGIN_ROOT from the environment — CC exports it to every
//     MCP server child, and it points at the installed devkit plugin dir
//     (not the .mcpb-cache subdir this binary physically lives in).
//  2. Parse <CLAUDE_PLUGIN_ROOT>/.claude-plugin/plugin.json for the plugin
//     version. Authoritative source of truth — matches what bin/devkit reads
//     on POSIX.
//  3. Compute the engine path at <CLAUDE_PLUGIN_ROOT>/bin/
//     devkit-engine-v<version>-windows-amd64.exe. Same cache location as
//     bin/devkit, so nothing gets re-downloaded if the user happens to have
//     run the engine through any other path before.
//  4. If the engine isn't cached, fetch checksums.txt and the engine asset
//     from the matching GitHub release, verify SHA-256, and atomic-rename
//     into place. Stale binaries from other versions are swept.
//  5. exec (well, cmd.Start + cmd.Wait — Windows has no execve) the engine
//     with this process's args and inherited stdio. MCP JSON-RPC flows
//     directly through the launcher's parent pipes into the engine.
//
// Go stdlib only — no external modules — so the static build stays small
// and reproducible. Crucially, Go's crypto/tls is used for the HTTPS fetch,
// not Windows schannel, which sidesteps the CDN renegotiation bug issue #58
// / PR #59 had to work around for curl.exe.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	// Cap a single HTTP fetch at 5 minutes. The engine binary is ~8 MB and
	// checksums.txt is a few KB — anything past this window is a stuck
	// connection, not a slow one.
	httpTimeout = 5 * time.Minute
)

func main() {
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

	// CLAUDE_PLUGIN_ROOT must be an absolute path. CC always sets it that
	// way; anything else is either a corrupted env or a spoofed launch and
	// we refuse to touch the filesystem from there.
	if !filepath.IsAbs(pluginRoot) {
		return fmt.Errorf("CLAUDE_PLUGIN_ROOT is not absolute: %q", pluginRoot)
	}

	pluginJSON := filepath.Join(pluginRoot, ".claude-plugin", "plugin.json")
	version, err := readPluginVersion(pluginJSON)
	if err != nil {
		return fmt.Errorf("reading plugin version from %s: %w", pluginJSON, err)
	}
	// version is interpolated into a filename joined to binDir below. Reject
	// anything outside a narrow semver-ish charset so a corrupted or spoofed
	// plugin.json can't produce a path that escapes binDir via "..", a path
	// separator, or shell metacharacters.
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

	if !fileIsExecutable(enginePath) {
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
// reserved for the MCP stdio transport once the engine takes over — we
// must never write to it from the launcher side.
func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "devkit launcher: "+format+"\n", args...)
}

// versionPattern matches a conservative superset of semver: digits, dots,
// and hyphen-separated alphanumeric pre-release/build tags. Explicitly
// excludes "/", "\", "..", and anything that could escape a directory join.
var versionPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+){0,3}(-[A-Za-z0-9.]+)?$`)

func validateVersion(version string) error {
	if version == "" {
		return errors.New("empty")
	}
	if !versionPattern.MatchString(version) {
		return errors.New("must match ^[0-9]+(\\.[0-9]+){0,3}(-[A-Za-z0-9.]+)?$")
	}
	// Defense in depth: even inside the charset, reject ".." and path
	// separators. The regex already excludes them, but if a future refactor
	// widens versionPattern these guards still hold.
	if strings.Contains(version, "..") || strings.ContainsAny(version, `/\`) {
		return errors.New("contains path traversal sequence or separator")
	}
	return nil
}

func readPluginVersion(path string) (string, error) {
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

func fileIsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}

// ensureEngine fetches checksums.txt and the engine asset from the matching
// GitHub release, verifies SHA-256, and atomic-installs the binary at
// enginePath. Intermediate files (.sums, .partial) are cleaned up on both
// success and failure.
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
		return err
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

func downloadTo(url, destPath string) error {
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
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return out.Sync()
}

// findChecksum parses a sha256sum-format file (two columns: hash, filename;
// filename may have a leading "*" for binary mode) and returns the hash for
// assetName. Matches bin/devkit's awk logic.
func findChecksum(sumsFile, assetName string) (string, error) {
	data, err := os.ReadFile(sumsFile)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
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
// sitting in binDir. Matches bin/devkit's find-based sweep. Errors are
// non-fatal — worst case we leave a few MB on disk.
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
// by reference, so MCP JSON-RPC flows straight through Claude Code's pipes
// into the engine with no buffering or framing on the launcher's part.
func execEngine(enginePath string, args []string) error {
	cmd := exec.Command(enginePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}
