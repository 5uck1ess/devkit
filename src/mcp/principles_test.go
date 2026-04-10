package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrinciples(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillsDir, 0o755)
	wfDir := filepath.Join(dir, "workflows")
	os.MkdirAll(wfDir, 0o755)

	content := []byte("dry:\n  - Don't abstract until 3rd duplication\nyagni:\n  - Build what's needed now\n")
	os.WriteFile(filepath.Join(skillsDir, "_principles.yml"), content, 0o644)

	p, err := LoadPrinciples(wfDir)
	if err != nil {
		t.Fatalf("LoadPrinciples: %v", err)
	}
	if len(p["dry"]) != 1 {
		t.Errorf("expected 1 dry principle, got %d", len(p["dry"]))
	}
	if len(p["yagni"]) != 1 {
		t.Errorf("expected 1 yagni principle, got %d", len(p["yagni"]))
	}
}
