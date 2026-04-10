---
description: Run a YAML workflow by name via the devkit engine (feature, bugfix, refactor, self-*, audit, etc.).
---

# Workflow Runner

Generic entry point for any YAML workflow in `workflows/`. The devkit engine controls step order, loops, gates, and branches.

## Invoke

Use the `devkit_list` tool first to see available workflows, or pick one by name:

```
devkit_start(workflow: "<name>", input: "<description>")
```

Then call `devkit_advance(session: "<id>")` after completing each step the engine returns. The engine controls step order, gates, and loops. Do NOT skip steps.

## Available Workflows

| Workflow | Purpose |
|---|---|
| `feature` | Brainstorm, plan, implement, test, lint, review |
| `bugfix` | Reproduce, diagnose, fix, regression test, verify |
| `refactor` | Analyze smells, plan, restructure, verify nothing broke |
| `research` | Clarify, decompose, parallel search, synthesize |
| `deep-research` | ACH hypotheses, disconfirmation, evidence matrix |
| `self-test` | Run tests, fix failures, loop until passing |
| `self-lint` | Run linter, fix violations, loop until clean |
| `self-perf` | Benchmark, optimize, loop until target met |
| `self-improve` | Run metric, fix issues, loop until passing |
| `self-migrate` | Migrate code incrementally with test gate |
| `self-audit` | Measure codebase, rank improvements by evidence |
| `autoloop` | Autonomous audit/fix/measure/keep-or-revert loop |
| `audit` | Dependencies, vulnerabilities, licenses, lint, security |
| `pr-ready` | Full PR preparation pipeline |
| `tri-review` | Multi-agent code review |
| `tri-debug` | Multi-agent debugging |
| `tri-security` | Multi-agent security audit |
| `tri-dispatch` | Generic parallel dispatch to multiple agents |

## Examples

```
# List workflows
devkit_list()

# Run a feature workflow
devkit_start(workflow: "feature", input: "add JWT auth to src/auth/")

# Run self-test with the project test command
devkit_start(workflow: "self-test", input: "npm test")

# Autonomous improvement loop
devkit_start(workflow: "autoloop", input: "improve test coverage | metric: go test -cover ./... | direction: higher-is-better | iterations: 10")
```

## Rules

- The engine enforces step order — you cannot skip steps
- Command steps execute automatically when you call `devkit_advance`
- Loop steps repeat based on gate/until/max conditions
- Branch steps jump based on output content matching
- Parallel steps dispatch to subagents via the Agent tool + plugins
- Call `devkit_status` anytime to check current progress
