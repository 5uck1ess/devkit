package runners

import "testing"

func TestAgyRunnerName(t *testing.T) {
	r := &AgyRunner{}
	if r.Name() != "agy" {
		t.Errorf("got %q, want %q", r.Name(), "agy")
	}
}

func TestAgyExecArgs(t *testing.T) {
	tests := []struct {
		name    string
		agyArgs string
		workDir string
		want    []string
	}{
		{
			name: "default no workdir",
			want: []string{"exec", "-"},
		},
		{
			name:    "default with workdir",
			workDir: "/tmp/repo",
			want:    []string{"exec", "-C", "/tmp/repo", "-"},
		},
		{
			name:    "AGY_ARGS override with workdir token",
			agyArgs: "run --cwd {workdir} -",
			workDir: "/tmp/repo",
			want:    []string{"run", "--cwd", "/tmp/repo", "-"},
		},
		{
			name:    "AGY_ARGS override drops workdir token when unset",
			agyArgs: "run --cwd {workdir} -",
			want:    []string{"run", "--cwd", "-"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv with "" forces the default branch (TrimSpace == "").
			t.Setenv("AGY_ARGS", tt.agyArgs)
			got := agyExecArgs(tt.workDir)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d (%v)", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("arg[%d] = %q, want %q (all args: %v)", i, got[i], tt.want[i], got)
				}
			}
		})
	}
}

func TestAgyCmdOverride(t *testing.T) {
	t.Setenv("AGY_CMD", "my-custom-agy")
	if got := agyCmd(); got != "my-custom-agy" {
		t.Errorf("agyCmd() = %q, want %q", got, "my-custom-agy")
	}
	// Empty override falls back to the default binary name.
	t.Setenv("AGY_CMD", "")
	if got := agyCmd(); got != "agy" {
		t.Errorf("agyCmd() default = %q, want %q", got, "agy")
	}
}

func TestAgyAvailableFalseWhenBinaryMissing(t *testing.T) {
	t.Setenv("AGY_CMD", "definitely-not-a-real-binary-xyz123")
	r := &AgyRunner{}
	if r.Available() {
		t.Error("Available() = true for a nonexistent binary, want false")
	}
}
