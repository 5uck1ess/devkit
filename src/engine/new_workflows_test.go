package engine

import (
	"path/filepath"
	"testing"
)

// TestParseNewWorkflows is a temporary sanity test added while landing
// feat/deterministic-skill-dispatch. It parses every YAML under
// ../../workflows/ through the engine's own parser so a malformed new
// workflow (test-gen / doc-gen / onboard / etc.) fails loudly in CI
// instead of at first `devkit_start` call on the user's box.
//
// Safe to remove once the parse is exercised by an integration harness
// that walks the workflows directory natively.
func TestParseAllShippedWorkflows(t *testing.T) {
	matches, err := filepath.Glob("../../workflows/*.yml")
	if err != nil {
		t.Fatalf("glob workflows: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no workflows found under ../../workflows/*.yml")
	}
	for _, path := range matches {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			wf, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile(%s): %v", path, err)
			}
			if err := wf.Validate(); err != nil {
				t.Fatalf("Validate(%s): %v", path, err)
			}
		})
	}
}
