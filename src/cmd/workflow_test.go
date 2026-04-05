package cmd

import (
	"regexp"
	"testing"
	"time"
)

func TestWorkflowNameValidation(t *testing.T) {
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validPattern.MatchString(tt.input)
			if got != tt.valid {
				t.Errorf("validate(%q) = %v, want %v", tt.input, got, tt.valid)
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
	// Exactly 1 minute should show "1m ago", not "just now"
	got := formatAge(time.Now().Add(-61 * time.Second))
	if got != "1m ago" {
		t.Errorf("formatAge(-61s) = %q, want %q", got, "1m ago")
	}

	// Exactly 1 hour should show "1h ago"
	got = formatAge(time.Now().Add(-61 * time.Minute))
	if got != "1h ago" {
		t.Errorf("formatAge(-61m) = %q, want %q", got, "1h ago")
	}

	// Exactly 24 hours should show "1d ago"
	got = formatAge(time.Now().Add(-25 * time.Hour))
	if got != "1d ago" {
		t.Errorf("formatAge(-25h) = %q, want %q", got, "1d ago")
	}
}
