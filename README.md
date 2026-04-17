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
brew install ast-grep  # AST-based repo mapping (used by onboard skill)

# Browser automation — enables scrape (JS-rendered), screenshot, and browser skills
npx playwright install chromium
```

**Playwright** (optional) enables three skills: enhanced `scrape` for JS-heavy sites, `screenshot` for page captures, and `browser` for full automation (clicking, form filling, multi-step flows, codegen). Free and local — no API keys. Install only the browsers you need (`chromium` is ~170MB).

### Verify

```bash
/health
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
/tri-review                   # Multi-agent code review
# Or just describe: "submit a PR", "ship this" → pr-ready skill auto-activates
```

---

## Local models (optional)

devkit can dispatch fast-tier work (doc generation, changelogs, summarization, test stubs) to any OpenAI-compatible local inference server. Cloud tiers — Claude, Codex, Gemini — still handle tri-review, architecture planning, and security review.

### Environment variables

| Var | Default | Purpose |
|---|---|---|
| `DEVKIT_LOCAL_ENABLED` | unset | Set to `1` to enable |
| `DEVKIT_LOCAL_ENDPOINT` | `http://localhost:11434/v1` | Base URL — must end in `/v1` |
| `DEVKIT_LOCAL_MODEL` | `qwen3:32b` | Model name sent in the request payload |
| `DEVKIT_LOCAL_API_KEY` | unset | Bearer token, if endpoint requires auth |
| `DEVKIT_LOCAL_TIMEOUT` | `600` | Per-request timeout (seconds) |
| `DEVKIT_LOCAL_DEBUG` | unset | Set to `1` to log probe failures to stderr |

### Default ports by stack

| Stack | Default port | Model-name source |
|---|---|---|
| llama-server (llama.cpp) | 8080 | loaded via `-m` at launch |
| llama-swap | 8080 | names in llama-swap YAML |
| Ollama | 11434 | `ollama list` |
| LM Studio | 1234 | GUI-loaded model |
| vLLM | 8000 | `--model` flag at launch |
| SGLang | 30000 | `--model-path` at launch |
| LocalAI | 8080 | models directory |

Port 8080 is the llama.cpp convention; llama-swap reuses it since it fronts llama-server processes.

### Examples

Single model, no router (llama-server, LM Studio, LocalAI, vLLM, SGLang):

```bash
export DEVKIT_LOCAL_ENABLED=1
export DEVKIT_LOCAL_ENDPOINT=http://<host>:<port>/v1
export DEVKIT_LOCAL_MODEL=<model-loaded-at-launch>
devkit-engine probe-local
```

Multi-model via router (Ollama, llama-swap, LocalAI):

```bash
export DEVKIT_LOCAL_ENABLED=1
export DEVKIT_LOCAL_ENDPOINT=http://<host>:<port>/v1
export DEVKIT_LOCAL_MODEL=<name-the-router-recognizes>
devkit-engine probe-local
```

### Verifying the setup

`devkit-engine probe-local` calls `$DEVKIT_LOCAL_ENDPOINT/models`, reports reachability and latency, and checks whether your configured model is present in the server's model list. Exit 0 on healthy, 1 otherwise. Add `--json` for structured output.

### Limits

- Fast tier only — local models have higher tool-call error rates than cloud tiers, so tri-review / architecture / security still go to the cloud.
- No function-calling (plain `/v1/chat/completions` only).
- No streaming.

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

## Workflows

All 22 YAML workflows are invoked via the MCP engine. Every workflow has a trigger skill so natural-language keywords dispatch deterministically — saying "build a feature", "fix this bug", "tri review", or "deep research X" fires the matching skill, which calls `devkit_start` and the engine takes over.

Every workflow is also a tab-completable slash command. Bare names work (`/feature`, `/bugfix`, `/tri-review`, `/health`, `/setup-rules`); the fully-qualified `/devkit:<name>` form also works if you want to disambiguate from another plugin or a Claude Code built-in.

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
| `test-gen` | Generate tests via test-writer agent, iterate until passing |
| `doc-gen` | Generate docs via documenter agent |
| `onboard` | Generate codebase onboarding guide via researcher agent |

---

## Skills

Skills activate automatically based on context. No slash command needed. Every workflow has a matching trigger skill — saying the keyword dispatches to the engine which then enforces every step.

**Workflow trigger skills** (dispatch to engine-enforced workflows):

| Trigger | Skill → Workflow |
|---|---|
| "build a feature", "new feature X" | `feature` |
| "fix this bug", "this is broken" | `bugfix` |
| "refactor this", "clean up X" | `refactor` |
| "audit this project", "project health" | `audit` |
| "research X" | `research` |
| "deep research", "validate this" | `deep-research` |
| "make a PR", "ship this", "create a pull request" | `pr-ready` |
| "tri review", "triple review" | `tri-review` |
| "tri debug", "triple debug" | `tri-debug` |
| "tri security", "triple security audit" | `tri-security` |
| "tri dispatch", "send to three models" | `tri-dispatch` |
| "self-audit", "audit the codebase" | `self-audit` |
| "self-improve", "keep fixing until X passes" | `self-improve` |
| "self-lint", "fix all lint" | `self-lint` |
| "self-migrate", "migrate incrementally" | `self-migrate` |
| "self-perf", "optimize performance" | `self-perf` |
| "self-test", "fix failing tests" | `self-test` |
| "autoloop", "run experiments overnight" | `autoloop` |
| "write tests for X" | `test-gen` |
| "document this module" | `doc-gen` |
| "onboard to this codebase" | `onboard` |

**Other skills** (tools, meta-orchestration, content):

| Trigger | Skill |
|---|---|
| "generate a changelog" | `changelog` |
| "create an ADR" | `adr` |
| "mega PR review" | `mega-pr` (dispatches tri-review + pr-review-toolkit in parallel) |
| "scrape this URL" | `scrape` |
| "screenshot this page" | `screenshot` (requires Playwright) |
| "automate this browser flow" | `browser` (requires Playwright) |
| Google Workspace CLI commands | `gcli` |

Coding principles (`clean-code`, `dry`, `yagni`, `dont-reinvent`, `executing`, `stuck`, `scratchpad`) are injected as condensed rules (~120 tokens) per workflow step — not loaded as full skill files.

---

## Hooks

12 hooks across 4 lifecycle events. All installed automatically with the plugin.

| Event | Hook | What it catches |
|---|---|---|
| PreToolUse | **safety-check** | `rm -rf /`, `DROP TABLE`, force push, editing secrets |
| PreToolUse | **security-patterns** | `eval()`, XSS, shell injection, weak hashes, hardcoded secrets |
| PreToolUse | **audit-trail** | Logs every command to `.devkit/audit.log` |
| PreToolUse | **pr-gate** | Prompts to run the pr-ready skill before `gh pr create` |
| PreToolUse | **rtk-rewrite** | Compresses Bash output via RTK (no-op if not installed) |
| PreToolUse | **devkit-guard** | Blocks out-of-step tools during workflow command AND prompt steps (hard enforce); soft enforce emits a reminder. Skills are intentionally unguarded. |
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
/setup-rules
```

| Language | Examples |
|---|---|
| Common | Cross-platform paths, assumption-surfacing, error messages, temp dirs |
| Go | Error wrapping, context.Context, defer traps, JSON float64 gotcha |
| TypeScript | `unknown` not `any`, discriminated unions, catch narrowing |
| Python | Exception chains, type hints, dataclasses, pathlib |
| Rust | Ownership, `?` propagation, newtypes, clippy-as-errors |
| Shell | `set -euo pipefail`, quoting, macOS portability |
| Java | Optional, records, try-with-resources, BigDecimal for money |
| Kotlin | `val` default, sealed classes, coroutines, Elvis operator |
| Swift | `guard let`, struct-first, async/await, weak self |
| C# | Records, pattern matching, async Task, Path.Combine |

---

## Architecture

```
MCP Server (bin/devkit mcp — auto-started by plugin)
  ├── bin/devkit = POSIX shell wrapper (committed to git)
  │   └── On first run, downloads matching release asset from GitHub,
  │       verifies SHA256, caches as bin/devkit-engine-v<ver>-<os>-<arch>,
  │       then execs it. Local dev builds (make install-plugin) are used
  │       directly via the fast path.
  ├── Tools: devkit_start, devkit_advance, devkit_status, devkit_list
  ├── State: session.json (hot, <50ms reads) + SQLite (cold history)
  ├── Parse YAML → validate steps, branches, budget
  ├── Walk steps:
  │   ├── Command steps → engine executes shell directly ($0 cost)
  │   │   Values passed via $DEVKIT_INPUT / $DEVKIT_OUT_<step_id>
  │   │   env vars — never interpolated into the command string.
  │   ├── Prompt steps → Claude works, calls devkit_advance when done
  │   ├── Loop with gate → run, verify, keep or revert
  │   ├── Branch → case-insensitive word-boundary match → goto
  │   └── Parallel → Agent tool dispatch (Claude/Codex/Gemini)
  └── Principles injected per step (~120 tokens, not full skill files)

Enforcement:
  ├── MCP tool scoping — Claude can only call devkit_advance to progress
  ├── PreToolUse hook — exit 2 blocks tools during command steps
  └── Stop hook — blocks session end during active workflows

Terminal usage (devkit workflow <name> "<description>"):
  ├── Subprocess runners for Codex/Gemini CLI
  └── In-process HTTP runner for Ollama/llama-server/vLLM (opt-in via DEVKIT_LOCAL_ENABLED=1)
```

---

## Repository Structure

```
devkit/
├── commands/          # Legacy (references/ only); new entry points go in skills/
├── skills/            # 39 skills (workflow triggers, principles, tools, utilities) + _principles.yml
├── agents/            # 6 agents (reviewer, researcher, improver, ...)
├── hooks/             # 12 hooks (safety, security, quality gates, workflow enforcement)
├── workflows/         # 22 YAML workflow definitions
├── resources/rules/   # Language-specific coding rules
├── src/               # Go engine + MCP server
│   ├── mcp/           # MCP server (tools, principles loader, session management)
│   ├── engine/        # YAML workflow engine (parser, executor, tests)
│   ├── runners/       # Claude/Codex/Gemini CLI interfaces + local HTTP runner (Ollama-compat, opt-in)
│   ├── lib/           # DB, git, metrics, session state, reporting
│   └── cmd/           # CLI entry points (including `devkit mcp`)
├── bin/               # devkit wrapper (committed) + downloaded engine binaries (gitignored)
└── .github/workflows/ # CI (build+test+vet) + auto-release (6 platforms)
```
