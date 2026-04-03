package runners

import "context"

type RunResult struct {
	Output    string
	SessionID string
	TokensIn  int
	TokensOut int
	CostUSD   float64
	ExitCode  int
}

type RunOpts struct {
	WorkDir                string
	AllowedTools           string
	AppendSystemPromptFile string
	AppendSystemPrompt     string
	MaxTurns               int
}

type Runner interface {
	Name() string
	Available() bool
	Run(ctx context.Context, prompt string, opts RunOpts) (RunResult, error)
}

func DetectRunners() []Runner {
	all := []Runner{
		&ClaudeRunner{},
		&CodexRunner{},
		&GeminiRunner{},
	}
	var available []Runner
	for _, r := range all {
		if r.Available() {
			available = append(available, r)
		}
	}
	return available
}

func FindRunner(name string, runners []Runner) Runner {
	for _, r := range runners {
		if r.Name() == name {
			return r
		}
	}
	return nil
}
