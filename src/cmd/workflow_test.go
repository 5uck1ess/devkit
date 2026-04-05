package cmd

import (
	"testing"
	"time"

	"github.com/5uck1ess/devkit/runners"
)

func TestWorkflowNameValidation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"simple", "feature", true},
		{"with-dash", "self-improve", true},
		{"with-underscore", "my_workflow", true},
		{"with-numbers", "v2-test", true},
		{"path-traversal", "../etc/passwd", false},
		{"absolute-path", "/etc/passwd", false},
		{"spaces", "my workflow", false},
		{"dots", "my.workflow", false},
		{"empty", "", false},
		{"shell-injection", "foo;rm -rf /", false},
		{"backtick", "foo`id`", false},
		{"dollar", "foo$HOME", false},
		{"null-byte", "foo\x00bar", false},
		{"pipe", "foo|cat", false},
		{"newline", "foo\nbar", false},
		{"lone-dot", ".", false},
		{"double-dot", "..", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validWorkflowName.MatchString(tt.input)
			if got != tt.valid {
				t.Errorf("validWorkflowName.MatchString(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name string
		age  time.Duration
		want string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"days", 48 * time.Hour, "2d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAge(time.Now().Add(-tt.age))
			if got != tt.want {
				t.Errorf("formatAge(-%v) = %q, want %q", tt.age, got, tt.want)
			}
		})
	}
}

func TestFormatAge_Boundaries(t *testing.T) {
	// At exactly 59s — still "just now" (d < time.Minute)
	got := formatAge(time.Now().Add(-59 * time.Second))
	if got != "just now" {
		t.Errorf("formatAge(-59s) = %q, want %q", got, "just now")
	}

	// At 61s — crosses minute boundary, should be "1m ago"
	got = formatAge(time.Now().Add(-61 * time.Second))
	if got != "1m ago" {
		t.Errorf("formatAge(-61s) = %q, want %q", got, "1m ago")
	}

	// At 59m — still minutes, should be "59m ago"
	got = formatAge(time.Now().Add(-59 * time.Minute))
	if got != "59m ago" {
		t.Errorf("formatAge(-59m) = %q, want %q", got, "59m ago")
	}

	// At 61m — crosses hour boundary, should be "1h ago"
	got = formatAge(time.Now().Add(-61 * time.Minute))
	if got != "1h ago" {
		t.Errorf("formatAge(-61m) = %q, want %q", got, "1h ago")
	}

	// At 23h — still hours
	got = formatAge(time.Now().Add(-23 * time.Hour))
	if got != "23h ago" {
		t.Errorf("formatAge(-23h) = %q, want %q", got, "23h ago")
	}

	// At 25h — crosses day boundary, should be "1d ago"
	got = formatAge(time.Now().Add(-25 * time.Hour))
	if got != "1d ago" {
		t.Errorf("formatAge(-25h) = %q, want %q", got, "1d ago")
	}
}

func TestResolveRunnerFrom_EmptyName(t *testing.T) {
	available := []runners.Runner{&stubRunner{"claude"}}
	_, err := resolveRunnerFrom("", available)
	if err == nil {
		t.Fatal("expected error for empty agent name")
	}
}
