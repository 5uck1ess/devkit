# Devkit

A deterministic development harness for AI agents. The MCP engine controls workflow execution (step ordering, gates, loops, branches). The agent handles creativity. Every step is enforced, measured, and auditable.

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
```

---

## How It Works

Devkit runs as an **MCP server** inside Claude Code. When a workflow starts, the engine takes control:

```
devkit_start("research", "best Go testing frameworks")
  → Engine creates session, returns Step 1 + condensed principles
  → Claude executes the step using standard tools
  → Claude calls devkit_advance(session_id)
  → Engine validates, records output, returns Step 2
  → ...repeat until WORKFLOW COMPLETE

Enforcement (runs automatically):
  PreToolUse hook → blocks out-of-step actions during command steps
  Stop hook → prevents session end during active workflows
```

**Why MCP?** Claude can't skip steps because the engine controls what comes next. Claude can't call tools that aren't valid for the current step. The engine holds state — Claude doesn't self-report.

---

## Commands

6 tab-completable slash commands. All other workflows are context-activated via skills (auto-triggered by natural language) or invoked via MCP tools.

| Command | What it does |
|---|---|
| `/tri:review` | Code review from 1-3 agents, consolidated report |
| `/tri:debug` | Independent root-cause analysis from each agent |
| `/tri:security` | Security audit with severity-ranked consensus |
| `/devkit:workflow` | Run any YAML workflow by name |
| `/devkit:status` | Health check |
| `/devkit:setup-rules` | Install language-specific coding rules to `~/.claude/rules/` |

Tasks like "ship this PR" or "submit a PR" auto-activate the `pr-ready` skill — no slash command needed.

### Workflows

All 18 YAML workflows are invoked via the MCP engine. Skills auto-activate for common triggers (e.g., "research X", "fix this bug", "add a feature").

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

Coding principles (`clean-code`, `dry`, `yagni`, `dont-reinvent`, `executing`, `stuck`, `scratchpad`) are injected as condensed rules (~120 tokens) per workflow step — not loaded as full skill files.

---

## Hooks

12 hooks across 4 lifecycle events. All installed automatically with the plugin.

| Event | Hook | What it catches |
|---|---|---|
| PreToolUse | **safety-check** | `rm -rf /`, `DROP TABLE`, force push, editing secrets |
| PreToolUse | **security-patterns** | `eval()`, XSS, shell injection, weak hashes, hardcoded secrets |
| PreToolUse | **audit-trail** | Logs every command to `.devkit/audit.log` |
| PreToolUse | **pr-gate** | Prompts to run `/devkit:pr-ready` before `gh pr create` |
| PreToolUse | **rtk-rewrite** | Compresses Bash output via RTK (no-op if not installed) |
| PreToolUse | **devkit-guard** | Blocks out-of-step tools during workflow command steps |
| PostToolUse | **post-validate** | Suppressed errors, leaked secrets, writes outside repo |
| PostToolUse | **slop-detect** | AI code patterns — doc/code imbalance, restating comments |
| PostToolUse | **lang-review** | Language-aware checks: Go, TypeScript, Rust, Python, Shell |
| SubagentStop | **subagent-stop** | Verifies subagent work before accepting |
| Stop | **stop-gate** | Merge conflicts, cross-domain test gaps, linter pass |
| Stop | **devkit-stop-guard** | Blocks session end during active workflows |

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

## Architecture

```
MCP Server (bin/devkit mcp — auto-started by plugin)
  ├── Tools: devkit_start, devkit_advance, devkit_status, devkit_list
  ├── State: session.json (hot, <50ms reads) + SQLite (cold history)
  ├── Parse YAML → validate steps, branches, budget
  ├── Walk steps:
  │   ├── Command steps → engine executes shell directly ($0 cost)
  │   ├── Prompt steps → Claude works, calls devkit_advance when done
  │   ├── Loop with gate → run, verify, keep or revert
  │   ├── Branch → case-insensitive substring match → goto
  │   └── Parallel → Agent tool dispatch (Claude/Codex/Gemini)
  └── Principles injected per step (~120 tokens, not full skill files)

Enforcement:
  ├── MCP tool scoping — Claude can only call devkit_advance to progress
  ├── PreToolUse hook — exit 2 blocks tools during command steps
  └── Stop hook — blocks session end during active workflows

Terminal fallback (devkit workflow run <name>):
  └── Subprocess runners for Codex/Gemini CLI usage
```

---

## Repository Structure

```
devkit/
├── commands/          # 6 slash commands (tab-completable entry points)
├── skills/            # 20 context-activated skills + _principles.yml
├── agents/            # 6 agents (reviewer, researcher, improver, ...)
├── hooks/             # 12 hooks (safety, security, quality gates, workflow enforcement)
├── workflows/         # 18 YAML workflow definitions
├── resources/rules/   # Language-specific coding rules
├── src/               # Go engine + MCP server
│   ├── mcp/           # MCP server (tools, principles loader, session management)
│   ├── engine/        # YAML workflow engine (parser, executor, tests)
│   ├── runners/       # Codex, Gemini interfaces (terminal fallback)
│   ├── loops/         # Improve, feature, bugfix, refactor, testgen
│   ├── lib/           # DB, git, metrics, session state, reporting
│   └── cmd/           # CLI entry points (including `devkit mcp`)
├── bin/               # Auto-PATH binary (built by make install-plugin)
└── .github/workflows/ # CI (build+test+vet) + auto-release (6 platforms)
```
