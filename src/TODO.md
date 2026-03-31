# Devkit CLI Harness — TODO

> Go binary (Cobra CLI) for deterministic orchestration that skills/workflows can't guarantee.
> Complements both pi workflows and Claude Code skills.

## When to build

- Self-improve loop stops early because the LLM decides to
- Multi-agent dispatch doesn't wait for all agents
- You want to run iterations unattended (overnight, CI, cron)
- You need deterministic behavior every time
- Pi workflow conditions aren't reliable enough for production loops

## Architecture

```
devkit (Go binary, Cobra CLI)
├── cmd/
│   ├── root.go              # Cobra root command
│   ├── review.go            # devkit review "prompt"
│   ├── improve.go           # devkit improve --target --metric --objective --iterations
│   └── dispatch.go          # devkit dispatch --agent claude|codex|gemini|pi|all "prompt"
├── runners/
│   ├── claude.go            # Spawn claude -p with agent config flags
│   ├── codex.go             # Spawn codex exec --full-auto
│   ├── gemini.go            # Spawn gemini -p -y
│   └── pi.go                # Spawn pi -p with workflow flags
├── loops/
│   └── improve.go           # Baseline → iterate → measure → keep/discard → repeat
├── lib/
│   ├── git.go               # Deterministic git ops (branch, commit, revert, diff)
│   ├── metric.go            # Run metric command, parse result, compare
│   └── report.go            # Consolidated output (stdout, markdown, JSON)
├── go.mod
└── main.go
```

## Key dependencies

- github.com/spf13/cobra — CLI framework
- os/exec — spawn agent processes
- Standard library for everything else (git, file I/O, JSON)

## Commands

```bash
devkit review "check for DRY violations"
devkit review --security
devkit improve --target src/ --metric "npm test" --objective "fix failures" --iterations 20
devkit dispatch --agent all "compare caching approaches"
devkit dispatch --agent pi "analyze with pi workflow"
```

## What the harness handles (that skills/workflows can't)

- Deterministic loop control (exact N iterations)
- Process management (spawn, timeout, kill)
- Exit code-based metric evaluation
- Git branching/committing/reverting without LLM involvement
- Parallel process orchestration with proper wait/collect
- Crash recovery (state on disk, resume where left off)
- Budget tracking across iterations
- Structured JSON reporting

## What stays as skills/workflows

- The prompts (what to tell each agent)
- Agent configs (model, effort, maxTurns)
- Pi workflow definitions (YAML)
- Claude Code plugin for users who don't need the binary
