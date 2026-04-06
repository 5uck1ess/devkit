# Devkit

A deterministic development harness for AI agents.

AI agents write code fast but cut corners — they skip error handling, introduce race conditions, ignore edge cases, and refactor things you didn't ask them to touch. Devkit is the infrastructure layer between you and the agents. A Go engine executes workflows deterministically — branching, loops, budget enforcement, and parallel dispatch all happen in compiled code, not LLM reasoning. 9 language-aware hooks enforce quality at every stage of the lifecycle across Go, TypeScript, Rust, Python, and Shell — many learned directly from bugs found in previous review cycles. The system gets better every time it catches something new.

Define your workflow in YAML. The engine handles orchestration. The agent handles creativity. Every change is measured, gated, and auditable.

Works with just Claude. Optionally adds Codex and Gemini for multi-agent consensus.

## Install

```bash
/plugin marketplace add https://github.com/5uck1ess/marketplace.git
/plugin install devkit@5uck1ess-plugins
```

### Holistic Setup

Devkit focuses on enforcement, orchestration, and multi-agent workflows. For a complete setup, add these companion plugins — each handles a different concern with no overlap.

| Plugin | What it handles | Install |
|---|---|---|
| **[superpowers](https://github.com/obra/superpowers)** | Methodology — brainstorming, planning, TDD, verification, debugging | `/plugin install superpowers@claude-plugins-official` |
| **[feature-dev](https://github.com/anthropics/claude-plugins-official)** | Deep feature exploration — parallel codebase analysis, architecture proposals, interactive design | `/plugin install feature-dev@claude-plugins-official` |
| **[pr-review-toolkit](https://github.com/anthropics/claude-plugins-official)** | Specialized review agents — comment accuracy, type design, silent failure hunting, error handling | `/plugin install pr-review-toolkit@claude-plugins-official` |
| **[commit-commands](https://github.com/anthropics/claude-plugins-official)** | Quick commits — auto-message `/commit`, one-shot `/commit-push-pr`, stale branch cleanup `/clean_gone` | `/plugin install commit-commands@claude-plugins-official` |
| **[hookify](https://github.com/anthropics/claude-plugins-official)** | Hook creation — markdown-based rules, hot reload, conversation analysis for auto-detection | `/plugin install hookify@claude-plugins-official` |
| **[skill-creator](https://github.com/anthropics/claude-plugins-official)** | Skill development — eval/benchmark framework, blind A/B comparison, iterative improvement | `/plugin install skill-creator@claude-plugins-official` |
| **[context-mode](https://github.com/mksglu/context-mode)** | Context window management — sandboxes large outputs, session continuity via SQLite, 98% savings | See below |

#### Context Mode Install

Plugin install (recommended — includes hooks + slash commands):
```bash
/plugin marketplace add mksglu/context-mode
/plugin install context-mode@context-mode
```

MCP-only install (lighter — sandbox tools only, no auto-routing):
```bash
claude mcp add context-mode -- npx -y context-mode
```

Verify with `/context-mode:ctx-doctor` (plugin install) or check MCP tools are available (MCP install).

**Why these and not others?** We evaluated every plugin in the official marketplace. These are the ones that add unique value without duplicating what devkit already does. Notably:

- **`code-simplifier`** — skip it. Thin, single-agent, hardcoded to React/TS. Devkit's `refactor` + `clean-code`/`dry`/`yagni` skills are more comprehensive.
- **`security-guidance`** — skip it. Devkit's `security-patterns` hook + `tri-security` command cover more patterns across more languages.
- **`code-review`** — skip it. Devkit's `tri-review` provides cross-model diversity (Claude + Codex + Gemini).
- **`ralph-loop`** — skip it. Devkit's `self-*` loops are specialized with proper metric gates.

### How they fit together

```
┌─────────────────────────────────────────────────────┐
│                   Your Project                       │
├──────────┬──────────┬──────────┬─────────────────────┤
│ Thinking │ Building │ Shipping │ Maintaining         │
├──────────┼──────────┼──────────┼─────────────────────┤
│superpow- │ devkit   │ devkit   │ devkit              │
│ers:      │ feature  │ pr-ready │ self-improve/test/   │
│ brain-   │ bugfix   │ pr-moni- │ lint/perf/migrate   │
│ storm    │ refactor │ tor      │                     │
│ plan     │ decompose│          │ tri-review/debug/   │
│ TDD      │          │ commit-  │ security/test-gen   │
│ debug    │feature-  │ commands │                     │
│          │dev:      │          │ pr-review-toolkit   │
│          │ explore  │          │                     │
│          │ design   │          │ audit, repo-map     │
├──────────┴──────────┴──────────┴─────────────────────┤
│ Auto skills: test-gen, doc-gen, changelog, onboard,  │
│ research, scrape (no slash command needed)            │
├──────────────────────────────────────────────────────┤
│ Always active: devkit hooks (safety, security,       │
│ audit, slop, lang-review, security, stop-gate)       │
├──────────────────────────────────────────────────────┤
│ Meta: hookify (create hooks), skill-creator (skills) │
│       context-mode (token management)                │
└──────────────────────────────────────────────────────┘
```

---

## Quick Start

```bash
# Check what's available
/devkit:status

# These activate automatically — just ask naturally:
# "write tests for src/parser.ts"
# "generate a changelog"
# "help me understand this codebase"
# "research the best auth library for Node"
# "scrape this URL: https://example.com"

# Slash commands for complex workflows:
/self:lint --lint "npm run lint" --target src/
/devkit:pr-ready
/tri:review
```

---

## Commands

### Solo Commands (Claude-only, no external CLIs needed)

| Command | Description |
|---|---|
| `/devkit:pr-ready` | Full PR pipeline — lint, test, security, changelog, create PR |
| `/devkit:pr-monitor` | Post-PR review monitor — watches CI, resolves reviewer comments iteratively |
| `/devkit:repo-map` | AST-based symbol index — exports, classes, imports, dependency graph, cached |
| `/devkit:workflow` | Run user-defined YAML workflows from `workflows/` |
| `/devkit:bugfix` | Full bug fix lifecycle — reproduce, diagnose, fix, regression test, verify |
| `/devkit:feature` | Full feature lifecycle — brainstorm, plan, implement, test, lint, review |
| `/devkit:refactor` | Full refactor lifecycle — analyze, plan, restructure, verify, compare |
| `/devkit:decompose` | Goal decomposition — break into task DAG, assign agents, execute in dependency order |
| `/devkit:audit` | Full project health audit — deps, vulnerabilities, licenses, lint, security |
| `/devkit:deep-research` | ACH-enhanced deep research — competing hypotheses, directed disconfirmation, evidence matrix, sensitivity analysis |
| `/devkit:status` | Health check — installed CLIs, available agents, ready commands |
| `/devkit:setup-rules` | Install language-specific coding rules to `~/.claude/rules/` |

### Self-Improvement Loops (Claude-only)

Automated propose → measure → keep/discard → repeat cycles.

| Command | Description |
|---|---|
| `/self:improve` | General-purpose improvement loop with custom metric gate |
| `/self:test` | Iteratively generate tests until coverage target is hit |
| `/self:lint` | Iteratively fix lint/type errors until zero remain |
| `/self:perf` | Hypothesis-driven performance investigation — evidence, hypotheses, one-at-a-time testing |
| `/self:migrate` | Incremental migration (JS→TS, class→hooks, etc.) with test gate |
| `/self:audit` | Codebase audit inspired by karpathy/autoresearch — measure everything, rank hypotheses, report only |
| `/devkit:autoloop` | Autonomous improvement loop — audit, fix, measure, keep or revert, repeat |

### Multi-Agent Commands (Claude + optional Codex/Gemini)

These run with whatever agents are available. Claude always runs. Codex and Gemini are used if installed.

| Command | Description |
|---|---|
| `/tri:review` | Code review from 1–3 agents, consolidated report |
| `/tri:dispatch` | Send any task to available agents, compare outputs |
| `/tri:debug` | Multi-perspective debugging — independent root-cause analysis |
| `/tri:test-gen` | Generate tests from multiple agents, merge best coverage |
| `/tri:security` | Security audit from multiple agents, severity-ranked consensus |

---

## Agents

| Agent | Model | Isolation | Effort | Max Turns | Used by |
|---|---|---|---|---|---|
| `reviewer` | Opus | Worktree | High | 10 | tri:review |
| `researcher` | Sonnet | Worktree | Medium | 15 | tri:dispatch, tri:debug, onboard |
| `improver` | Opus | Worktree | High | 10 | self:*, tri:dispatch |
| `test-writer` | Sonnet | Worktree | Medium | 15 | test-gen, self:test, tri:test-gen |
| `documenter` | Haiku | Worktree | Medium | 10 | doc-gen |
| `security-auditor` | Opus | Worktree | High | 10 | tri:security, pr-ready, audit |

---

## Skills

Skills activate automatically based on context — no slash command needed. Just ask naturally.

### Context-Activated Workflows

These replace slash commands. Ask naturally and the skill fires:

| Skill | Triggers on |
|---|---|
| `devkit:test-gen` | "write tests for X", "add test coverage", "generate tests" |
| `devkit:doc-gen` | "document this module", "generate API docs", "write docs for" |
| `devkit:changelog` | "generate a changelog", "release notes", "what changed since" |
| `devkit:onboard` | "explain this codebase", "help me understand the architecture", "onboard" |
| `devkit:research` | "research X", "compare approaches for", "investigate options" |
| `devkit:deep-research` | "deep research", "validate this", "make sure this is right", "rigorous analysis" |
| `devkit:scrape` | "scrape this URL", "fetch content from", "extract from this page" |
| `devkit:adr` | "create an ADR", "document this decision", "record why we chose X" |

### Coding Principles

Loaded as reference material when relevant:

| Skill | Description |
|---|---|
| `devkit:executing` | Execute plans methodically — understand, implement, verify, commit |
| `devkit:clean-code` | Meaningful names, small functions, single responsibility, flat nesting |
| `devkit:dry` | Rule of Three, when duplication is fine, extracting the right abstraction |
| `devkit:yagni` | Build only what's needed, no speculative features or premature abstractions |
| `devkit:dont-reinvent` | Use existing libraries, tools, and stdlib before building custom solutions |
| `devkit:stuck` | Detect agent looping/failing, structured recovery — backtrack, simplify, escalate |
| `devkit:scratchpad` | Iteration memory protocol — prevents Groundhog Day loops by recording what was tried |

### Tools

| Skill | Description |
|---|---|
| `devkit:gcli` | Google Workspace CLI (Gmail, Calendar, Drive) via gcli with `--for-ai` |
| `devkit:creating-workflows` | How to create workflow YAML files — schema, step types, interpolation |

For brainstorming, planning, TDD, verification, and skill authoring — install [superpowers](https://github.com/obra/superpowers).

### Coding Rules (user-level)

Language-specific rules that auto-activate when Claude reads matching files. Installed to `~/.claude/rules/` (not part of the plugin — plugins don't support `rules/`). These complement hooks: **rules guide how to write, hooks catch what you missed**.

Install with one command:

```
/devkit:setup-rules
```

| File | Paths | What it covers |
|---|---|---|
| `go.md` | `**/*.go` | Error wrapping, context.Context, defer traps, import grouping, table-driven tests |
| `typescript.md` | `**/*.{ts,tsx,js,jsx,mjs,cjs}` | Type safety (unknown not any), discriminated unions, catch narrowing, nullish coalescing |
| `python.md` | `**/*.py` | Exception chains, type hints, dataclasses, pathlib, comprehensions |
| `rust.md` | `**/*.rs` | Ownership patterns, ? propagation, newtypes, iterator preference, clippy-as-errors |
| `shell.md` | `**/*.sh` | set -euo pipefail, quoting, macOS portability, subshell traps |

Each file uses `paths:` frontmatter for scoped activation. Rules are ~10 lines each — minimal token overhead.

---

## Hooks

Devkit ships 10 hooks across 4 lifecycle events. All are installed automatically with the plugin — no setup required.

### PreToolUse

| Hook | Matcher | What it does |
|---|---|---|
| **safety-check** | Bash, Edit, Write | Blocks destructive commands (`rm -rf /`, `DROP TABLE`, private key writes). Prompts on risky operations (force push, `git reset --hard`, editing secrets). |
| **security-patterns** | Edit, Write | Catches vulnerability patterns at creation time — `eval()`, XSS, shell injection, weak hashes, hardcoded secrets, path traversal. Language-aware (JS/TS/Python/Go). |
| **audit-trail** | Bash | Logs every command to `.devkit/audit.log` with UTC timestamps. Auto-rotates at 10k lines. |
| **pr-gate** | Bash | Detects `gh pr create` and prompts to run `/devkit:pr-ready` first. 10-minute cooldown. |
| **rtk-rewrite** | Bash | Rewrites commands through [RTK](https://github.com/rtk-ai/rtk) for 60-90% token savings. No-op if RTK not installed. |

### PostToolUse

| Hook | Matcher | What it does |
|---|---|---|
| **post-validate** | Bash, Edit, Write | Warns on suppressed errors, leaked secrets in written content, writes outside repo. |
| **slop-detect** | Edit, Write | Catches AI code patterns — doc/code ratio imbalance, restating comments, excessive JSDoc in .js files. |
| **lang-review** | Edit, Write | Language-aware code quality checks. Detects language from file extension and runs the right checks: Go (error-path access, map races, nil-error returns, filepath traversal), TypeScript (empty catches, any-type, unhandled promises), Rust (unwrap in non-test, let _ = discard, unsafe blocks), Python (bare except, pass-in-except, mutable defaults), Shell (macOS portability — grep -P, sed -i, readlink -f, timeout). |

### SubagentStop

| Hook | Matcher | What it does |
|---|---|---|
| **subagent-stop** | Stop | Verifies subagent work products before accepting. Recognizes Go, Node, Python, Rust, and generic test frameworks. |

### Stop

| Hook | Matcher | What it does |
|---|---|---|
| **stop-gate** | Stop | Consolidated quality gate: detects merge conflicts, checks cross-domain test evidence (blocks if backend + frontend changed but only one tested), runs language-appropriate linter (go vet + race detector, cargo clippy, tsc --noEmit, ruff). |

---

## RTK Token Optimization

Optional [RTK](https://github.com/rtk-ai/rtk) integration compresses Bash output before it reaches the context window.

| Operation | Before | After | Savings |
|---|---|---|---|
| Directory listing | ~2,000 tokens | ~400 tokens | 80% |
| Test output | ~25,000 tokens | ~2,500 tokens | 90% |
| Git operations | ~3,000 tokens | ~600 tokens | 80% |
| Search results | ~16,000 tokens | ~3,200 tokens | 80% |

```bash
brew install rtk
```

---

## Presets

Reusable prompt templates in `presets/`. Reference with `--preset`:

```bash
/tri:review --preset security-web
/tri:security --preset security-go
/self:perf --preset react-perf
```

### Included Presets

None yet — `presets/` is reserved for future use.

---

## Architecture

```
devkit workflow <name> (generic YAML engine)
  ├── Parse YAML → validate steps, branches, budget
  ├── Create session + git branch
  ├── Walk steps sequentially:
  │   ├── Interpolate {{variables}} in prompts
  │   ├── Call runner (Claude/Codex/Gemini)
  │   ├── Evaluate branches (case-insensitive substring match → goto)
  │   ├── Loop with hard counter + until-string match
  │   ├── Parallel dispatch via goroutines
  │   └── Budget enforcement (checked every step + inside loops)
  └── Commit, report, clean up

/tri:review (or any tri:* command)
  ├── Claude  → native background agent (always runs)
  ├── Codex   → plugin (preferred) or CLI subprocess (fallback)
  └── Gemini  → plugin (preferred) or CLI subprocess (fallback)

/self:improve (or any self:* command)
  └── Claude  → improver agent in worktree
      ↓ propose change
      ↓ run metric
      ↓ keep if pass / revert if fail
      ↓ repeat
```

---

## Repository Structure

```
devkit/
├── .claude-plugin/
│   └── plugin.json          # Plugin metadata (name, version, author)
├── ROADMAP.md               # Implemented features and future plans
├── PREFERENCES.md           # Agent behavior guidelines
├── commands/                # 24 slash commands (tab-completable)
│   ├── tri-*.md             # Multi-agent dispatch (5)
│   ├── self-*.md            # Self-improvement loops (6)
│   ├── pr-ready.md          # PR preparation pipeline
│   ├── pr-monitor.md        # Post-PR review monitor
│   ├── bugfix.md            # Bug fix lifecycle
│   ├── feature.md           # Feature lifecycle
│   ├── refactor.md          # Refactor lifecycle
│   ├── decompose.md         # Goal decomposition
│   ├── workflow.md          # YAML workflow runner
│   ├── audit.md             # Project health audit
│   ├── repo-map.md          # AST-based symbol index
│   ├── status.md            # Health check
│   └── setup-rules.md       # Install coding rules to ~/.claude/rules/
├── skills/                  # 18 context-activated skills
│   ├── executing/SKILL.md   # Principle: methodical execution
│   ├── clean-code/SKILL.md  # Principle: readability
│   ├── dry/SKILL.md         # Principle: don't repeat yourself
│   ├── yagni/SKILL.md       # Principle: no speculative features
│   ├── dont-reinvent/SKILL.md # Principle: use existing solutions
│   ├── stuck/SKILL.md       # Principle: loop recovery
│   ├── creating-workflows/SKILL.md # Tool: YAML workflow authoring
│   ├── gcli/SKILL.md        # Tool: Google Workspace CLI
│   ├── changelog/SKILL.md   # Auto: "generate a changelog"
│   ├── doc-gen/SKILL.md     # Auto: "document this module"
│   ├── test-gen/SKILL.md    # Auto: "write tests for X"
│   ├── onboard/SKILL.md     # Auto: "explain this codebase"
│   ├��─ research/SKILL.md    # Auto: "research X"
│   ├── deep-research/SKILL.md # Auto: "deep research", ACH pipeline
│   ├── scratchpad/SKILL.md  # Iteration memory protocol
│   ├── scrape/SKILL.md      # Auto: "scrape this URL"
│   └── adr/SKILL.md         # Auto: "create an ADR", decision records
├── agents/                  # 6 agents
│   ├── reviewer.md          # Opus, worktree isolation
│   ├── researcher.md        # Sonnet, worktree isolation
│   ├── improver.md          # Opus, worktree isolation
│   ├── test-writer.md       # Sonnet, worktree isolation
│   ├── documenter.md        # Haiku, worktree isolation
│   └── security-auditor.md  # Opus, worktree isolation
├── hooks/                   # 10 hooks across 4 lifecycle events
│   ├── hooks.json           # Hook config (auto-loaded)
│   ├── safety-check.sh      # Dangerous operation blocker
│   ├── security-patterns.sh # Edit-time vulnerability detection
│   ├── audit-trail.sh       # Command logging
│   ├── rtk-rewrite.sh       # Token optimization
│   ├── pr-gate.sh           # PR pipeline prompt
│   ├── post-validate.sh     # Output validation
│   ├── slop-detect.sh       # AI pattern detection
│   ├── lang-review.sh       # Language-aware code quality (Go/TS/Rust/Python/Shell)
│   ├── subagent-stop.sh     # Subagent work verification
│   └── stop-gate.sh         # Consolidated quality gate (cross-domain + vet/lint)
├── resources/rules/         # Reference rule files (installed via /devkit:setup-rules)
├── workflows/               # 15 YAML workflow definitions
├── presets/                  # Reserved for future use
├── .github/workflows/       # CI/CD
│   ├── ci.yml               # Build + test + vet on push/PR
│   └── release.yml          # Auto-tag + release on version bump
└── src/                     # Go CLI harness
    ├── cmd/                 # Cobra commands (including workflow)
    ├── engine/              # Generic YAML workflow engine — parser, executor, tests
    ├── lib/                 # DB, git, metric, state, report
    ├── loops/               # Improve, feature, bugfix, refactor, testgen, review, dispatch
    └── runners/             # Claude, Codex, Gemini runner interfaces
```

---

## Autonomy Flags

Set automatically in each multi-agent command:

| Agent | Flags |
|---|---|
| Claude | `--dangerously-skip-permissions` |
| Codex | `/codex:rescue --background` (via [codex-plugin-cc](https://github.com/openai/codex-plugin-cc)) or `codex -q` (CLI fallback) |
| Gemini | `/gemini:rescue --background` (via [gemini-plugin-cc](https://github.com/abiswas97/gemini-plugin-cc)) or `-y` (CLI fallback) |

---

## Go CLI Harness

Deterministic orchestration binary — the machine controls the loop, the agent is the body.

```bash
cd src && make build && make link
devkit --help
```

Or install directly:

```bash
cd src && make install
```

### Commands

All loop commands support `--agent` to choose the AI agent (default: `claude`).

| Command | Description |
|---|---|
| `devkit workflow` | **Generic YAML workflow engine** — runs any workflow from `workflows/` deterministically |
| `devkit improve` | Metric-gated iteration loop — one agent invocation per iteration |
| `devkit feature` | Plan, implement, test, lint — commits only after tests pass |
| `devkit bugfix` | Diagnose, fix, verify — reverts if tests break |
| `devkit refactor` | Analyze, transform, verify — reverts if behavior changes |
| `devkit test-gen` | Generate tests, run, fix failures — iterates until green |
| `devkit review` | Parallel multi-agent code review |
| `devkit dispatch` | Send any task to multiple agents, compare outputs |
| `devkit status` | Show all sessions, costs, iteration history |
| `devkit resume` | Pick up a crashed or paused session |

### What it does that plugins can't

- **Exact iteration counts** — Go binary owns the loop, not the LLM
- **Generic YAML workflows** — `devkit workflow <name>` executes any workflow definition deterministically. Branching, loops, parallel dispatch, budget enforcement, and variable interpolation all happen in Go — zero tokens spent on orchestration
- **Triage-based phase skipping** — feature and bugfix workflows classify task scope (TINY/SMALL/MEDIUM/LARGE) and skip unnecessary steps. A typo fix doesn't run a 14-step pipeline
- **Crash recovery** — SQLite state + handoff files survive crashes
- **Hard budget caps** — stops spawning at your dollar limit, including inside loops
- **CI/CD integration** — runs headless, no conversation needed
- **True parallel dispatch** — goroutines, not sequential prompts
- **Multi-agent support** — `--agent claude`, `--agent codex`, or `--agent gemini`

### Examples

```bash
# Run 50 improvement iterations overnight, stop at $20
devkit improve --metric "npm test" --iterations 50 --budget 20.00

# Same thing with Codex instead of Claude
devkit improve --metric "npm test" --iterations 50 --agent codex

# Implement a feature with test verification
devkit feature "add JWT auth" --target src/auth/ --test "npm test"

# Fix a bug with automated verification
devkit bugfix "login 500 on plus sign emails" --test "go test ./..."

# Generate tests for a module
devkit test-gen src/parser/ --test "go test ./..."

# Multi-agent review with all available agents
devkit review

# Run any YAML workflow by name
devkit workflow feature "add JWT auth"
devkit workflow bugfix "null pointer on empty input"

# List available workflows
devkit workflow list

# Resume a crashed session
devkit resume abc123def456

# Check what happened
devkit status
```

### Testing

```bash
cd src && go test ./... -v
```

100+ tests across 5 packages (engine, lib, runners, loops, cmd). Engine tests cover parsing, interpolation, branching, loops, budget, parallel dispatch, cycle detection. All tests use mock runners — no API calls needed.

### CI/CD

- **CI pipeline** (`.github/workflows/ci.yml`): build + vet + test (with `-race`) + gofmt check on every push/PR to main
- **Auto-release** (`.github/workflows/release.yml`): auto-bumps version, tags, and creates GitHub release on merged PRs

---

## Prerequisites

**Required:** Claude Code (you're already here)

**Optional** (for multi-agent commands):
```bash
# Codex plugin (preferred for tri:* commands)
/plugin marketplace add openai/codex-plugin-cc
/plugin install codex@openai-codex

# Gemini plugin (preferred for tri:* commands)
/plugin marketplace add abiswas97/gemini-plugin-cc
/plugin install gemini@abiswas97-gemini
```

**CLI fallbacks** (used only if plugins are not installed):
```bash
brew install codex gemini-cli
```

**Optional** (for token optimization):
```bash
brew install rtk
```

**Optional** (for AST-based repo mapping):
```bash
brew install ast-grep
```

Check status with `/devkit:status`.

---

## Roadmap

See [ROADMAP.md](ROADMAP.md) for full details.

- [x] Go CLI harness — 9 commands, SQLite state, crash recovery, budget enforcement, multi-agent support
- [x] CI/CD pipeline — build, vet, test, auto-release on version bump
- [x] Branch protection — PRs required for main
- [x] Edit-time security hooks — vulnerability pattern detection on Write/Edit
- [x] Slop detection — AI code pattern enforcement
- [x] Audit trail — command logging with timestamps
- [x] Project health audit — unified deps, vulns, licenses, lint, security
- [x] Post-PR monitor — CI watching + iterative comment resolution
- [x] AST-based repo map — symbol index with dependency graph
- [x] Hypothesis-driven perf — evidence gathering, ranked theories, one-at-a-time testing
- [x] Generic YAML workflow engine — deterministic step execution, branching, loops, parallel, budget
- [x] Triage-based phase skipping — TINY/SMALL/MEDIUM/LARGE classification with fast paths
- [x] Iteration scratchpads — persistent memory across loop iterations to prevent repeated failures
- [x] Cross-domain dirty-bit enforcement — blocks completion without test evidence per domain
- [x] Go code quality hooks — error-path access, nil-return, race detection, portability
- [x] **Language-universal hooks** — consolidated Go-specific hooks into `lang-review.sh` with Go, TypeScript, Rust, Python, and Shell support
- [x] **Hook consolidation** — merged 14 hooks into 10, reduced per-edit shell processes from 7 to 4. Consolidated stop hooks (dirty-bit + go-vet-stop + old stop-gate) into single `stop-gate.sh`

All planned items complete. See [ROADMAP.md](ROADMAP.md) for retired items and rationale.
