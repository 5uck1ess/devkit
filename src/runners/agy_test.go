package runners

import "testing"

func TestAgyRunnerName(t *testing.T) {
	r := &AgyRunner{}
	if r.Name() != "agy" {
		t.Errorf("Name() = %q, want %q", r.Name(), "agy")
	}
}

func TestAgyOutputIsError(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"observed print-timeout", "Error: timed out waiting for response", true},
		{"hypothetical print timeout", "Error: print timeout", true},
		{"leading and trailing whitespace", "  Error: rate limited\n", true},

		{"model discussing errors (multi-paragraph)", "Error: NullPointerException at line 42\n\nThe root cause is the nil dereference in handleRequest()...", false},
		{"long model response starting with Error:", "Error: " + longString(400), false},
		{"Error: mid-body, not at prefix", "Here's a review.\n\nError: this is a discussion of error handling.", false},
		{"empty output", "", false},
		{"whitespace only", "   \n  \t  ", false},
		{"lowercase error: is not an agy error", "error: lowercase from a model response", false},
		{"uppercase ERROR: is not matched (case-sensitive)", "ERROR: yelling", false},
		{"Error- with dash, not colon", "Error- not the right delimiter", false},
		{"plain success output", "ok", false},
		{"multi-line model output", "Here is my review:\n- point 1\n- point 2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := agyOutputIsError(tt.output); got != tt.want {
				t.Errorf("agyOutputIsError(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func longString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'x'
	}
	return string(b)
}
