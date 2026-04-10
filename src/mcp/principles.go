package mcp

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadPrinciples reads the condensed principles index.
// Looks in the plugin's skills/ directory for _principles.yml.
func LoadPrinciples(workflowDir string) (map[string][]string, error) {
	// Try relative to workflow dir (plugin root/skills/_principles.yml)
	candidates := []string{
		filepath.Join(filepath.Dir(workflowDir), "skills", "_principles.yml"),
	}

	// Also check CLAUDE_PLUGIN_ROOT
	if root := os.Getenv("CLAUDE_PLUGIN_ROOT"); root != "" {
		candidates = append([]string{filepath.Join(root, "skills", "_principles.yml")}, candidates...)
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var principles map[string][]string
		if err := yaml.Unmarshal(data, &principles); err != nil {
			return nil, fmt.Errorf("parse principles: %w", err)
		}
		return principles, nil
	}

	return nil, fmt.Errorf("_principles.yml not found")
}
