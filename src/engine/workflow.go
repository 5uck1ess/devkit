// Package engine provides a generic YAML workflow execution engine.
// It replaces hardcoded Go workflow implementations with a single engine
// that reads and executes workflow YAML files deterministically.
package engine

import (
	"bytes"
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
	Enforce     string   `yaml:"enforce"`    // "hard" (default) | "soft"
	BranchMode  bool     `yaml:"branch"`     // create git branch per session
	Principles  []string `yaml:"principles"` // principle keys to inject
}

// Budget controls token spending limits.
type Budget struct {
	Limit     int    `yaml:"limit"`
	Downgrade string `yaml:"downgrade"`
}

// WfStep is a single step in a workflow.
type WfStep struct {
	ID         string   `yaml:"id"`
	Model      string   `yaml:"model"`
	Prompt     string   `yaml:"prompt"`
	Command    string   `yaml:"command"`
	Expect     string   `yaml:"expect"`
	Parallel   []string `yaml:"parallel"`
	Loop       *Loop    `yaml:"loop"`
	Branch     []Branch `yaml:"branch"`
	Principles []string `yaml:"principles"` // per-step override
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

// Parse parses workflow YAML bytes with strict field checking so typos
// like "commnd:" fail loudly instead of silently producing a step that
// never runs the intended command.
func Parse(data []byte) (*Workflow, error) {
	var wf Workflow
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&wf); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if err := validate(&wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

// validate checks the workflow for structural errors.
func validate(wf *Workflow) error {
	// Apply defaults before validation so directly-constructed Workflow values
	// (not via Parse) also get sensible defaults.
	if wf.Enforce == "" {
		wf.Enforce = "hard"
	}

	if wf.Name == "" {
		return fmt.Errorf("workflow missing name")
	}
	if wf.Enforce != "hard" && wf.Enforce != "soft" {
		return fmt.Errorf("workflow %q has invalid enforce %q — must be \"hard\" or \"soft\"", wf.Name, wf.Enforce)
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
		if s.Expect != "" && s.Command == "" {
			return fmt.Errorf("step %q has expect without command — expect only applies to command steps", s.ID)
		}
		if s.Expect != "" && s.Expect != "success" && s.Expect != "failure" {
			return fmt.Errorf("step %q has invalid expect %q — must be \"success\" or \"failure\"", s.ID, s.Expect)
		}
		if s.Command != "" && s.Loop != nil {
			return fmt.Errorf("step %q has both command and loop — these are mutually exclusive", s.ID)
		}
		if len(s.Parallel) > 0 && s.Loop != nil {
			return fmt.Errorf("step %q has both parallel and loop — these are mutually exclusive", s.ID)
		}
		// Command strings are never interpolated — values are passed
		// through env vars ($DEVKIT_INPUT, $DEVKIT_OUT_<id>) to avoid
		// shell injection. Reject {{...}} in command/gate strings so
		// the author gets a clear error instead of a silently broken
		// step or, worse, a shell injection.
		if s.Command != "" && strings.Contains(s.Command, "{{") {
			return fmt.Errorf("step %q command must not use {{...}} — pass values via $DEVKIT_INPUT or $DEVKIT_OUT_<step_id> instead (shell injection mitigation)", s.ID)
		}
		if s.Loop != nil && s.Loop.Gate != "" && strings.Contains(s.Loop.Gate, "{{") {
			return fmt.Errorf("step %q loop.gate must not use {{...}} — pass values via $DEVKIT_INPUT or $DEVKIT_OUT_<step_id> instead (shell injection mitigation)", s.ID)
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
//
// Matching is word-boundary (case-insensitive): the sentinel must
// appear as a whole word in the output, bounded on both sides by a
// non-alphanumeric character or string edge. This is the same
// semantics as grep -w. It accepts idiomatic patterns like "TINY: short
// fix" and "attempt 2: ALL_PASSING" while rejecting accidental
// substrings — `fail` won't match inside `failures`, and `small` won't
// match inside `smaller`.
//
// Note: workflow authors should still pick distinctive sentinels.
// `until: done` will match any sentence containing the standalone word
// "done" (e.g. a prose reply "I'm done reviewing"). Prefer sentinels
// like `ALL_DONE`, `DONE_FIXING`, or `===DONE===` for robustness.
func EvalBranch(output string, branches []Branch) string {
	for _, b := range branches {
		want := strings.ToLower(strings.TrimSpace(b.When))
		if want == "" {
			continue
		}
		if containsWord(output, want) {
			return b.Goto
		}
	}
	return ""
}

// MatchUntil checks whether a step's output satisfies its loop `until`
// sentinel. Same word-boundary semantics as EvalBranch.
func MatchUntil(output, sentinel string) bool {
	want := strings.ToLower(strings.TrimSpace(sentinel))
	if want == "" {
		return false
	}
	return containsWord(output, want)
}

// containsWord returns true when `want` (already lowercased and
// trimmed) appears in `text` as a whole word — bounded on both sides
// by a non-alphanumeric character, underscore, or string edge. Matches
// grep -w semantics.
func containsWord(text, want string) bool {
	lower := strings.ToLower(text)
	for i := 0; ; {
		idx := strings.Index(lower[i:], want)
		if idx < 0 {
			return false
		}
		start := i + idx
		end := start + len(want)
		if isWordBoundary(lower, start, end) {
			return true
		}
		i = start + 1
		if i >= len(lower) {
			return false
		}
	}
}

// isWordBoundary returns true when the characters just outside [start,
// end) in s are not alphanumeric/underscore (or the edge of the string).
func isWordBoundary(s string, start, end int) bool {
	leftOK := start == 0 || !isWordChar(s[start-1])
	rightOK := end == len(s) || !isWordChar(s[end])
	return leftOK && rightOK
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}
