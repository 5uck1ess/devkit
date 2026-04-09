// Package engine provides a generic YAML workflow execution engine.
// It replaces hardcoded Go workflow implementations with a single engine
// that reads and executes workflow YAML files deterministically.
package engine

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Workflow is the top-level YAML structure.
type Workflow struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Budget      Budget   `yaml:"budget"`
	Steps       []WfStep `yaml:"steps"`
}

// Budget controls token spending limits.
type Budget struct {
	Limit     int    `yaml:"limit"`
	Downgrade string `yaml:"downgrade"`
}

// WfStep is a single step in a workflow.
type WfStep struct {
	ID       string   `yaml:"id"`
	Model    string   `yaml:"model"`
	Prompt   string   `yaml:"prompt"`
	Command  string   `yaml:"command"`
	Parallel []string `yaml:"parallel"`
	Loop     *Loop    `yaml:"loop"`
	Branch   []Branch `yaml:"branch"`
}

// Loop controls step repetition.
type Loop struct {
	Max   int    `yaml:"max"`
	Until string `yaml:"until"`
	Gate  string `yaml:"gate"`
}

// Branch routes execution based on step output.
type Branch struct {
	When string `yaml:"when"`
	Goto string `yaml:"goto"`
}

// ParseFile reads and parses a workflow YAML file.
func ParseFile(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow: %w", err)
	}
	return Parse(data)
}

// Parse parses workflow YAML bytes.
func Parse(data []byte) (*Workflow, error) {
	var wf Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if err := validate(&wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

// validate checks the workflow for structural errors.
func validate(wf *Workflow) error {
	if wf.Name == "" {
		return fmt.Errorf("workflow missing name")
	}
	if len(wf.Steps) == 0 {
		return fmt.Errorf("workflow %q has no steps", wf.Name)
	}

	// Validate budget
	if wf.Budget.Limit < 0 {
		return fmt.Errorf("workflow %q has negative budget limit", wf.Name)
	}

	ids := make(map[string]bool)
	for _, s := range wf.Steps {
		if s.ID == "" {
			return fmt.Errorf("step missing id in workflow %q", wf.Name)
		}
		if ids[s.ID] {
			return fmt.Errorf("duplicate step id %q in workflow %q", s.ID, wf.Name)
		}
		ids[s.ID] = true

		// Validate step mode mutual exclusion
		if s.Command != "" && s.Prompt != "" {
			return fmt.Errorf("step %q has both command and prompt — these are mutually exclusive", s.ID)
		}
		if len(s.Parallel) > 0 && (s.Prompt != "" || s.Command != "") {
			return fmt.Errorf("step %q has both parallel and prompt/command — these are mutually exclusive", s.ID)
		}
		if s.Command != "" && s.Loop != nil {
			return fmt.Errorf("step %q has both command and loop — these are mutually exclusive", s.ID)
		}
		if len(s.Parallel) > 0 && s.Loop != nil {
			return fmt.Errorf("step %q has both parallel and loop — these are mutually exclusive", s.ID)
		}
	}

	// Validate branch targets exist
	for _, s := range wf.Steps {
		for _, b := range s.Branch {
			if !ids[b.Goto] {
				return fmt.Errorf("branch target %q not found (step %q)", b.Goto, s.ID)
			}
		}
		// Validate parallel references exist
		for _, pid := range s.Parallel {
			if !ids[pid] {
				return fmt.Errorf("parallel step %q not found (step %q)", pid, s.ID)
			}
		}
	}

	return nil
}

// Validate re-validates a workflow that may have been constructed directly
// (not via Parse). Call this at the engine boundary for safety.
func (wf *Workflow) Validate() error {
	return validate(wf)
}

// Interpolate replaces {{step-id}} and {{input}} placeholders in a prompt.
func Interpolate(prompt string, input string, outputs map[string]string) string {
	result := strings.ReplaceAll(prompt, "{{input}}", input)
	for id, output := range outputs {
		result = strings.ReplaceAll(result, "{{"+id+"}}", output)
	}
	return result
}

// EvalBranch checks step output against branch conditions.
// Returns the goto target step ID, or "" if no match.
func EvalBranch(output string, branches []Branch) string {
	lower := strings.ToLower(output)
	for _, b := range branches {
		if strings.Contains(lower, strings.ToLower(b.When)) {
			return b.Goto
		}
	}
	return ""
}
