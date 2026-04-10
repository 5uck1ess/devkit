package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func readDirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

func mkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func TestValidateVersion(t *testing.T) {
	cases := []struct {
		name    string
		version string
		wantErr bool
	}{
		// Valid — these must stay accepted.
		{"plain", "1.2.3", false},
		{"four parts", "1.2.3.4", false},
		{"pre-release", "2.1.7-rc.1", false},
		{"two digits", "10.20", false},
		{"single major", "1", false},
		{"three-digit components", "10.20.30", false},
		{"alpha dot num", "1.0.0-alpha.1", false},
		// Invalid — these are the security contract.
		{"empty", "", true},
		{"dotdot", "..", true},
		{"traversal forward", "1.2.3/../x", true},
		{"traversal back", `1.2.3\..\x`, true},
		{"leading traversal", "../1.2.3", true},
		{"semicolon injection", "1.2.3;rm", true},
		{"newline", "1.2.3\n", true},
		{"double dot inside pre", "1.2.3-foo..bar", true},
		{"forward slash", "1/2/3", true},
		{"backslash", `1\2\3`, true},
		{"space", "1.2 .3", true},
		{"build metadata not supported", "1.0.0+build.1", true},
		{"dangling hyphen", "1.2.3-", true},
		{"null byte", "1.2.3\x00", true},
		{"backtick", "1.2.3`pwd`", true},
		{"var expansion", "${PATH}", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateVersion(tc.version)
			if tc.wantErr && err == nil {
				t.Fatalf("validateVersion(%q) = nil, want error", tc.version)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("validateVersion(%q) = %v, want nil", tc.version, err)
			}
		})
	}
}

func TestFindChecksum(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		asset    string
		wantHash string
		wantErr  bool
	}{
		{
			name:     "plain single entry",
			body:     "abc123  devkit-windows-amd64.exe\n",
			asset:    "devkit-windows-amd64.exe",
			wantHash: "abc123",
		},
		{
			name:     "binary-mode star prefix",
			body:     "abc123 *devkit-windows-amd64.exe\n",
			asset:    "devkit-windows-amd64.exe",
			wantHash: "abc123",
		},
		{
			name: "target is second entry",
			body: "deadbeef  other.exe\n" +
				"cafef00d  devkit-windows-amd64.exe\n",
			asset:    "devkit-windows-amd64.exe",
			wantHash: "cafef00d",
		},
		{
			name:     "missing trailing newline",
			body:     "abc123  devkit-windows-amd64.exe",
			asset:    "devkit-windows-amd64.exe",
			wantHash: "abc123",
		},
		{
			name:     "CRLF line endings from Windows builder",
			body:     "abc123  devkit-windows-amd64.exe\r\n",
			asset:    "devkit-windows-amd64.exe",
			wantHash: "abc123",
		},
		{
			name: "CRLF mixed multi-line",
			body: "deadbeef  other.exe\r\n" +
				"cafef00d  devkit-windows-amd64.exe\r\n",
			asset:    "devkit-windows-amd64.exe",
			wantHash: "cafef00d",
		},
		{
			name:    "no match",
			body:    "abc123  different.exe\n",
			asset:   "devkit-windows-amd64.exe",
			wantErr: true,
		},
		{
			name:    "empty file",
			body:    "",
			asset:   "devkit-windows-amd64.exe",
			wantErr: true,
		},
		{
			name:    "single-column garbage",
			body:    "notachecksum\n",
			asset:   "devkit-windows-amd64.exe",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "sums.txt")
			if err := writeFile(path, tc.body); err != nil {
				t.Fatal(err)
			}
			got, err := findChecksum(path, tc.asset)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("findChecksum() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("findChecksum() error = %v", err)
			}
			if got != tc.wantHash {
				t.Fatalf("findChecksum() = %q, want %q", got, tc.wantHash)
			}
		})
	}
}

func TestSweepStaleEngines(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"devkit-engine-v2.1.6-windows-amd64.exe", // current
		"devkit-engine-v2.1.5-windows-amd64.exe", // stale
		"devkit-engine-v2.0.0-darwin-arm64",      // stale
		"devkit-checksums-v2.1.5.txt",            // stale
		"README.md",                              // must be preserved
		"devkit",                                 // must be preserved (no -engine-v prefix)
		"unrelated-binary.exe",                   // must be preserved
	}
	for _, name := range files {
		if err := writeFile(filepath.Join(dir, name), "x"); err != nil {
			t.Fatal(err)
		}
	}

	current := "devkit-engine-v2.1.6-windows-amd64.exe"
	if err := sweepStaleEngines(dir, current); err != nil {
		t.Fatalf("sweepStaleEngines() error = %v", err)
	}

	want := map[string]bool{
		"devkit-engine-v2.1.6-windows-amd64.exe": true,
		"README.md":                              true,
		"devkit":                                 true,
		"unrelated-binary.exe":                   true,
	}
	entries, err := readDirNames(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, name := range entries {
		got[name] = true
	}
	for name := range want {
		if !got[name] {
			t.Errorf("file %q was deleted but should have been preserved", name)
		}
	}
	for name := range got {
		if !want[name] {
			t.Errorf("file %q was preserved but should have been swept", name)
		}
	}
}

// TestExecEngineRemovesCorruptCache covers the self-heal path in
// execEngine: when cmd.Run returns a non-ExitError (CreateProcess /
// execve rejection — wrong architecture, truncated PE, ENOEXEC),
// the cached binary must be removed so the next launch re-downloads.
// A regression that moves the os.Remove inside the ExitError branch
// would quietly recreate the stuck-cache class this fix exists for.
func TestExecEngineRemovesCorruptCache(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.exe")
	// Non-binary contents with exec permission. execve/CreateProcess
	// reject this before the child ever runs, which is the non-ExitError
	// path execEngine must self-heal.
	if err := os.WriteFile(bad, []byte("not an executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := execEngine(bad, nil)
	if err == nil {
		t.Fatal("execEngine(non-binary) = nil, want error")
	}
	if _, statErr := os.Stat(bad); !os.IsNotExist(statErr) {
		t.Fatalf("cached binary not removed: stat err = %v", statErr)
	}
}

func TestEngineLooksCached(t *testing.T) {
	dir := t.TempDir()

	missing := filepath.Join(dir, "nope.exe")
	ok, err := engineLooksCached(missing)
	if err != nil || ok {
		t.Fatalf("missing: got ok=%v err=%v, want ok=false err=nil", ok, err)
	}

	empty := filepath.Join(dir, "empty.exe")
	if err := writeFile(empty, ""); err != nil {
		t.Fatal(err)
	}
	ok, err = engineLooksCached(empty)
	if err != nil || ok {
		t.Fatalf("empty: got ok=%v err=%v, want ok=false err=nil", ok, err)
	}

	present := filepath.Join(dir, "present.exe")
	if err := writeFile(present, "MZ"); err != nil {
		t.Fatal(err)
	}
	ok, err = engineLooksCached(present)
	if err != nil || !ok {
		t.Fatalf("real: got ok=%v err=%v, want ok=true err=nil", ok, err)
	}

	subdir := filepath.Join(dir, "adir")
	if err := mkdirAll(subdir); err != nil {
		t.Fatal(err)
	}
	ok, err = engineLooksCached(subdir)
	if err != nil || ok {
		t.Fatalf("dir: got ok=%v err=%v, want ok=false err=nil", ok, err)
	}
}

func TestReadPluginVersion(t *testing.T) {
	dir := t.TempDir()

	good := filepath.Join(dir, "good.json")
	if err := writeFile(good, `{"name":"devkit","version":"2.1.6"}`); err != nil {
		t.Fatal(err)
	}
	v, err := readPluginVersion(good)
	if err != nil || v != "2.1.6" {
		t.Fatalf("good: got %q err=%v, want 2.1.6 nil", v, err)
	}

	noVersion := filepath.Join(dir, "nover.json")
	if err := writeFile(noVersion, `{"name":"devkit"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := readPluginVersion(noVersion); err == nil {
		t.Fatal("no version field: want error, got nil")
	}

	empty := filepath.Join(dir, "empty.json")
	if err := writeFile(empty, `{"version":""}`); err != nil {
		t.Fatal(err)
	}
	if _, err := readPluginVersion(empty); err == nil {
		t.Fatal("empty version: want error, got nil")
	}

	garbage := filepath.Join(dir, "garbage.json")
	if err := writeFile(garbage, `not json at all`); err != nil {
		t.Fatal(err)
	}
	if _, err := readPluginVersion(garbage); err == nil {
		t.Fatal("garbage: want error, got nil")
	}

	// Oversize rejection must catch a file that would parse correctly if
	// we only read the first maxPluginJSONBytes. Use a legitimately large
	// JSON: a valid object at the head, then whitespace padding that
	// pushes total size past the cap. A buggy size guard (e.g. a naive
	// LimitReader) would silently truncate the padding and parse the
	// head, returning "2.1.6" instead of an error.
	oversized := filepath.Join(dir, "huge.json")
	head := `{"name":"devkit","version":"2.1.6"}`
	padding := strings.Repeat(" ", maxPluginJSONBytes)
	if err := writeFile(oversized, head+padding); err != nil {
		t.Fatal(err)
	}
	v, err = readPluginVersion(oversized)
	if err == nil {
		t.Fatalf("oversized: got %q, want error", v)
	}

	// Boundary: a manifest under the cap with some slack must still parse.
	slack := 4096
	boundary := filepath.Join(dir, "boundary.json")
	extra := strings.Repeat(" ", maxPluginJSONBytes-len(head)-slack)
	if err := writeFile(boundary, head+extra); err != nil {
		t.Fatal(err)
	}
	v, err = readPluginVersion(boundary)
	if err != nil || v != "2.1.6" {
		t.Fatalf("boundary: got %q err=%v, want 2.1.6 nil", v, err)
	}
}

// TestValidateVersionDefenseInDepth exercises the explicit Contains("..")
// and ContainsAny(`/\`) guards independently of the regex. The current
// regex already rejects these inputs, so a refactor that widens the regex
// must still hit the guards — this test lets that change fail fast.
func TestValidateVersionDefenseInDepth(t *testing.T) {
	// Temporarily widen the pattern to one that admits any printable
	// ASCII so we can prove the guards are doing load-bearing work. We
	// restore the original at the end.
	original := versionPattern
	defer func() { versionPattern = original }()
	versionPattern = regexp.MustCompile(`^[ -~]+$`)

	badInputs := []string{
		"1.2..3",
		"1/2/3",
		`1\2\3`,
		"..",
		"../etc/passwd",
		`..\windows\system32`,
	}
	for _, in := range badInputs {
		if err := validateVersion(in); err == nil {
			t.Errorf("widened-regex + guard: validateVersion(%q) = nil, want error", in)
		}
	}

	// Sanity: plain versions still pass under the widened regex.
	for _, in := range []string{"1.2.3", "2.1.7-rc.1"} {
		if err := validateVersion(in); err != nil {
			t.Errorf("widened-regex: validateVersion(%q) = %v, want nil", in, err)
		}
	}
}
