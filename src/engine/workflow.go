// Package engine provides a generic YAML workflow execution engine.
// It replaces hardcoded Go workflow implementations with a single engine
// that reads and executes workflow YAML files deterministically.
package engine

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/5uck1ess/devkit/lib"
	"gopkg.in/yaml.v3"
)

// EnforceMode aliases lib.EnforceMode so call sites in this package
// don't need a second import. The canonical definition lives in lib
// because SessionState (in lib) also needs it and lib cannot import
// engine (engine already imports lib).
type EnforceMode = lib.EnforceMode

const (
	EnforceInherit = lib.EnforceInherit
	EnforceHard    = lib.EnforceHard
	EnforceSoft    = lib.EnforceSoft
)

// Workflow is the top-level YAML structure.
type Workflow struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Budget      Budget      `yaml:"budget"`
	Steps       []WfStep    `yaml:"steps"`
	Enforce     EnforceMode `yaml:"enforce"`    // "hard" (default) | "soft"
	BranchMode  bool        `yaml:"branch"`     // create git branch per session
	Principles  []string    `yaml:"principles"` // principle keys to inject
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
	// Enforce overrides the workflow-level enforce for this step only.
	// Empty inherits from Workflow.Enforce. Lets a workflow keep most
	// prompt steps under hard (mid-step tool block) while allowing
	// specific steps whose body needs tools the hard mode blocks to
	// run under soft. The Stop-hook still blocks session end on soft
	// steps, so end-of-turn drift is still caught.
	Enforce EnforceMode `yaml:"enforce,omitempty"`
}

// EffectiveEnforce returns the enforcement mode for a step, falling back
// to the workflow-level setting when the step does not override it, and
// to EnforceHard when neither is set. Callers must use this instead of
// reading step.Enforce directly so the fall-through is consistent at
// every state transition. Takes values (not pointers) so the compiler
// enforces that both fields exist at the call site — every current
// caller owns concrete structs by the time they reach a transition.
// The return type is guaranteed concrete (never EnforceInherit), so
// callers storing the result into SessionState.StepEnforce can rely on
// the type-level invariant.
func EffectiveEnforce(wf Workflow, step WfStep) EnforceMode {
	if step.Enforce != EnforceInherit {
		return step.Enforce
	}
	if wf.Enforce != EnforceInherit {
		return wf.Enforce
	}
	return EnforceHard
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
	if wf.Enforce == EnforceInherit {
		wf.Enforce = EnforceHard
	}

	if wf.Name == "" {
		return fmt.Errorf("workflow missing name")
	}
	if !wf.Enforce.IsValid() {
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
	envKeys := make(map[string]string) // canonical env key → first step id that produced it
	for _, s := range wf.Steps {
		if s.ID == "" {
			return fmt.Errorf("step missing id in workflow %q", wf.Name)
		}
		if ids[s.ID] {
			return fmt.Errorf("duplicate step id %q in workflow %q", s.ID, wf.Name)
		}
		ids[s.ID] = true

		// Reject step IDs whose canonical env-key collides with an
		// earlier step. Without this, "fetch-data" and "fetch_data"
		// would both map to DEVKIT_OUT_FETCH_DATA and silently
		// overwrite each other depending on map iteration order.
		key := EnvKey(s.ID)
		if prior, clash := envKeys[key]; clash {
			return fmt.Errorf("step ids %q and %q collide under env key %q in workflow %q — rename one", prior, s.ID, key, wf.Name)
		}
		envKeys[key] = s.ID

		// Validate step mode mutual exclusion — exactly one of
		// prompt | command | parallel must be set. A step with only
		// metadata (id/model/principles but no executable body) would
		// otherwise parse fine and silently do nothing.
		modes := 0
		if s.Prompt != "" {
			modes++
		}
		if s.Command != "" {
			modes++
		}
		if len(s.Parallel) > 0 {
			modes++
		}
		if modes == 0 {
			return fmt.Errorf("step %q has no body — set exactly one of prompt, command, or parallel", s.ID)
		}
		if modes > 1 {
			return fmt.Errorf("step %q has multiple bodies — prompt, command, and parallel are mutually exclusive, set exactly one", s.ID)
		}
		if s.Expect != "" && s.Command == "" {
			return fmt.Errorf("step %q has expect without command — expect only applies to command steps", s.ID)
		}
		if s.Expect != "" && s.Expect != "success" && s.Expect != "failure" {
			return fmt.Errorf("step %q has invalid expect %q — must be \"success\" or \"failure\"", s.ID, s.Expect)
		}
		// Step-level enforce override: empty inherits from workflow,
		// otherwise must be hard|soft. Reject on command steps — the
		// guard honors SessionState.StepEnforce uniformly (see guard.go's
		// command branch), so marking a command step `soft` would let
		// arbitrary agent tool calls slip through while the engine is
		// executing that step. Since command steps are engine-owned
		// and never need per-step overrides, fail loudly at parse time
		// instead of producing a sharp edge at runtime.
		if s.Enforce != EnforceInherit {
			if !s.Enforce.IsValid() {
				return fmt.Errorf("step %q has invalid enforce %q — must be \"hard\" or \"soft\"", s.ID, s.Enforce)
			}
			if s.Command != "" {
				return fmt.Errorf("step %q has enforce on a command step — enforce is only meaningful for prompt steps (the engine executes command steps directly)", s.ID)
			}
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
			// Reject empty when — strings.Contains(x, "") is
			// always true, so an empty when: matches every step
			// and silently hijacks execution.
			if strings.TrimSpace(b.When) == "" {
				return fmt.Errorf("branch in step %q has empty when: — use a non-empty sentinel", s.ID)
			}
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

// EnvKey maps a workflow step ID to a POSIX env var suffix used in
// DEVKIT_OUT_<key>. POSIX allows [A-Za-z_][A-Za-z0-9_]*, so any
// non-alphanumeric byte is mapped to underscore and lowercase is
// upcased. Note the collision risk: "a-b", "a_b", "a.b", "A B" all
// produce "A_B". The validator rejects workflows whose step IDs
// collide under this mapping so two outputs can never silently shadow
// each other in the env.
func EnvKey(id string) string {
	b := make([]byte, 0, len(id))
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z':
			b = append(b, c-32)
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	return string(b)
}

// Interpolate replaces {{step-id}} and {{input}} placeholders in a prompt.
// Keys are iterated in sorted order so rendering is deterministic when one
// step's output itself contains a {{another-id}} placeholder.
func Interpolate(prompt string, input string, outputs map[string]string) string {
	result := strings.ReplaceAll(prompt, "{{input}}", input)
	ids := make([]string, 0, len(outputs))
	for id := range outputs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		result = strings.ReplaceAll(result, "{{"+id+"}}", outputs[id])
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
