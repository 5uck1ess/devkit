# Devkit

A deterministic development harness for AI agents. The Go engine controls orchestration (loops, branches, gates, budgets). The agent handles creativity. Every change is measured, gated, and auditable.

Works with just Claude. Optionally adds Codex and Gemini for multi-agent consensus.

---

## Install

### 1. Devkit (required)

```bash
/plugin marketplace add 5uck1ess/marketplace
/plugin install devkit@5uck1ess-plugins
```

Auto-updates are enabled by default. Devkit updates itself when you restart Claude Code.

### 2. Multi-agent plugins (optional)

These enable `tri:*` commands (tri-review, tri-debug, tri-security, etc.) to run Claude + Codex + Gemini in parallel.

```bash
# Codex plugin
/plugin marketplace add openai/codex-plugin-cc
/plugin install codex@openai-codex

# Gemini plugin
/plugin marketplace add abiswas97/gemini-plugin-cc
/plugin install gemini@abiswas97-gemini
```

If plugins aren't installed, the CLI fallbacks work too:
```bash
brew install codex gemini-cli
```

### 3. Companion plugins (optional)

These handle concerns devkit doesn't — methodology, specialized reviews, and context management. No overlap.

```bash
# Methodology — brainstorming, planning, TDD, verification, debugging
/plugin install superpowers@claude-plugins-official

# Specialized review agents — comment accuracy, type design, silent failures
/plugin install pr-review-toolkit@claude-plugins-official

# Deep feature exploration — parallel codebase analysis, architecture proposals
/plugin install feature-dev@claude-plugins-official

# Quick commits — /commit, /commit-push-pr, /clean_gone
/plugin install commit-commands@claude-plugins-official

# Hook creation — markdown rules, hot reload, conversation analysis
/plugin install hookify@claude-plugins-official

# Skill development — eval/benchmark framework, blind A/B testing
/plugin install skill-creator@claude-plugins-official

# Context window management — sandboxes large outputs, 98% token savings
/plugin marketplace add mksglu/context-mode
/plugin install context-mode@context-mode
```

### 4. Optional tools

```bash
brew install rtk       # Token optimization (60-90% savings on Bash output)
brew install ast-grep  # AST-based repo mapping (devkit workflow run repo-map)
```

### Verify

```bash
/devkit:status
```

This shows which CLIs are installed, which agents are available, and which commands are ready.

---

## Quick Start

```bash
# These activate automatically — just ask naturally:
# "write tests for src/parser.ts"
# "generate a changelog"
# "help me understand this codebase"
# "research the best auth library for Node"

# Slash commands for complex workflows:
/devkit:pr-ready              # Full PR pipeline
/tri:review                   # Multi-agent code review
devkit workflow run self-lint "npm run lint"  # Fix all lint errors
```

---

## Commands

8 tab-completable slash commands. All other workflows are context-activated via skills or invoked directly with `devkit workflow run <name>`.

| Command | What it does |
|---|---|
| `/tri:review` | Code review from 1-3 agents, consolidated report |
| `/tri:debug` | Independent root-cause analysis from each agent |
| `/tri:security` | Security audit with severity-ranked consensus |
| `/devkit:pr-ready` | Lint, test, security, changelog, create PR |
| `/devkit:pr-monitor` | Watch CI, fetch reviewer comments, fix iteratively, push |
| `/devkit:workflow` | Run user-defined YAML workflows |
| `/devkit:status` | Health check |
| `/devkit:setup-rules` | Install language-specific coding rules to `~/.claude/rules/` |

### Workflows (via `devkit workflow run <name>`)

All 18 YAML workflows can be invoked directly. Skills auto-activate for common triggers (e.g., "research X", "fix this bug", "add a feature").

| Workflow | What it does |
|---|---|
| `feature` | Brainstorm, plan, implement, test, lint, review |
| `bugfix` | Reproduce, diagnose, fix, regression test, verify |
| `refactor` | Analyze smells, plan, restructure, verify nothing broke |
| `research` | Clarify, decompose, parallel search, corroborate, synthesize |
| `deep-research` | ACH: hypotheses, disconfirmation, evidence matrix |
| `self-test` | Run tests, fix failures, repeat until passing |
| `self-lint` | Run linter, fix violations, repeat until clean |
| `self-perf` | Benchmark, optimize, repeat until target met |
| `self-improve` | Run metric, fix issues, repeat until passing |
| `self-migrate` | Migrate code incrementally with test gate |
| `self-audit` | Measure codebase, rank improvements by evidence |
| `autoloop` | Autonomous audit/fix/measure/keep-or-revert loop |
| `audit` | Dependencies, vulnerabilities, licenses, lint, security |
| `pr-ready` | Full PR preparation pipeline |
| `tri-review` | Multi-agent code review |
| `tri-debug` | Multi-agent debugging |
| `tri-security` | Multi-agent security audit |
| `tri-dispatch` | Send any task to multiple agents |

---

## Skills

Skills activate automatically based on context. No slash command needed.

| Trigger | Skill |
|---|---|
| "write tests for X" | `test-gen` |
| "document this module" | `doc-gen` |
| "generate a changelog" | `changelog` |
| "explain this codebase" | `onboard` |
| "research X" | `research` |
| "deep research", "validate this" | `deep-research` |
| "scrape this URL" | `scrape` |
| "create an ADR" | `adr` |

Coding principles (`clean-code`, `dry`, `yagni`, `dont-reinvent`, `executing`, `stuck`, `scratchpad`) load as reference when relevant.

---

## Hooks

10 hooks across 4 lifecycle events. All installed automatically with the plugin.

| Event | Hook | What it catches |
|---|---|---|
| PreToolUse | **safety-check** | `rm -rf /`, `DROP TABLE`, force push, editing secrets |
| PreToolUse | **security-patterns** | `eval()`, XSS, shell injection, weak hashes, hardcoded secrets |
| PreToolUse | **audit-trail** | Logs every command to `.devkit/audit.log` |
| PreToolUse | **pr-gate** | Prompts to run `/devkit:pr-ready` before `gh pr create` |
| PreToolUse | **rtk-rewrite** | Compresses Bash output via RTK (no-op if not installed) |
| PostToolUse | **post-validate** | Suppressed errors, leaked secrets, writes outside repo |
| PostToolUse | **slop-detect** | AI code patterns — doc/code imbalance, restating comments |
| PostToolUse | **lang-review** | Language-aware checks: Go, TypeScript, Rust, Python, Shell |
| SubagentStop | **subagent-stop** | Verifies subagent work before accepting |
| Stop | **stop-gate** | Merge conflicts, cross-domain test gaps, linter pass |

---

## Agents

| Agent | Model | Used by |
|---|---|---|
| `reviewer` | Opus | tri-review workflow, feature workflow |
| `researcher` | Sonnet | research, deep-research, tri-debug workflows |
| `improver` | Opus | self-improve, self-lint, self-perf, refactor workflows |
| `test-writer` | Sonnet | self-test, tri-test-gen workflows |
| `documenter` | Haiku | doc-gen skill |
| `security-auditor` | Opus | tri-security, pr-ready, audit workflows |

All agents run in worktree isolation.

---

## Coding Rules

Language-specific rules that auto-activate when Claude reads matching files. Installed to `~/.claude/rules/` — rules guide how to write, hooks catch what you missed.

```bash
/devkit:setup-rules
```

| Language | Examples |
|---|---|
| Go | Error wrapping, context.Context, defer traps, JSON float64 gotcha |
| TypeScript | `unknown` not `any`, discriminated unions, catch narrowing |
| Python | Exception chains, type hints, dataclasses, pathlib |
| Rust | Ownership, `?` propagation, newtypes, clippy-as-errors |
| Shell | `set -euo pipefail`, quoting, macOS portability |

---

## Go CLI Harness

The compiled Go binary handles deterministic orchestration — the machine controls the loop, the agent is the body.

### Build

```bash
cd src && make install
```

### What it does that plugins can't

- **Exact iteration counts** — Go owns the loop, not the LLM
- **Command steps** — run shell commands directly in workflows, $0 cost
- **Loop gates** — shell command after each iteration, auto-revert on failure
- **YAML workflows** — branching, loops, parallel dispatch, budget enforcement
- **Triage-based skipping** — typo fix doesn't run a 14-step pipeline
- **Crash recovery** — SQLite state survives crashes
- **Hard budget caps** — stops at your dollar limit
- **True parallel dispatch** — goroutines, not sequential prompts

### Examples

```bash
# Run 50 improvement iterations, stop at $20
devkit improve --metric "npm test" --iterations 50 --budget 20.00

# Implement a feature with test verification
devkit feature "add JWT auth" --target src/auth/ --test "npm test"

# Multi-agent review
devkit review

# Run any YAML workflow
devkit workflow feature "add JWT auth"

# Check session history
devkit status
```

### Testing

```bash
cd src && go test ./... -v
```

140+ tests across 6 packages. All use mock runners — no API calls needed.

---

## Architecture

```
Workflow Engine (Go binary)
  ├── Parse YAML → validate steps, branches, budget
  ├── Create session + git branch
  ├── Walk steps:
  │   ├── Command steps → shell execution (deterministic, $0)
  │   ├── Prompt steps → LLM runner (Claude/Codex/Gemini)
  │   ├── Loop with gate → run, verify, keep or revert
  │   ├── Branch → case-insensitive substring match → goto
  │   ├── Parallel → goroutines with mutex
  │   └── Budget check every step
  └── Commit, report, clean up

Multi-Agent (tri-* workflows)
  ├── Claude  → native background agent (always)
  ├── Codex   → plugin or CLI (optional)
  └── Gemini  → plugin or CLI (optional)

Self-Improvement (self-* workflows)
  └── Loop: propose → measure → keep/revert → repeat
```

---

## Repository Structure

```
devkit/
├── commands/          # 8 slash commands (tab-completable entry points)
├── skills/            # 19 context-activated skills
├── agents/            # 6 agents (reviewer, researcher, improver, ...)
├── hooks/             # 10 hooks (safety, security, quality gates)
├── workflows/         # 18 YAML workflow definitions
├── resources/rules/   # Language-specific coding rules
├── presets/            # Reserved for future use
├── src/               # Go CLI harness
│   ├── engine/        # YAML workflow engine (parser, executor, tests)
│   ├── runners/       # Claude, Codex, Gemini interfaces
│   ├── loops/         # Improve, feature, bugfix, refactor, testgen
│   ├── lib/           # DB, git, metrics, reporting
│   └── cmd/           # CLI entry points
└── .github/workflows/ # CI (build+test+vet) + auto-release
```
