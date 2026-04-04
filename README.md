# Devkit

Guardrails and consistency for Claude Code.

AI agents are powerful but unpredictable — they skip steps, jump to conclusions, and refactor things you didn't ask them to touch. Devkit enforces deterministic, step-by-step workflows that keep Claude on track: propose one change, measure it, keep or revert, repeat. No freestyling.

Every command follows a defined sequence. Self-improvement loops gate each change behind a metric. Multi-agent commands dispatch the same task to multiple models and consolidate consensus. The result is reproducible, auditable work — not whatever Claude felt like doing.

Works with just Claude. Optionally adds Codex and Gemini for multi-perspective analysis.

## Install

```bash
/plugin marketplace add https://github.com/5uck1ess/marketplace.git
/plugin install devkit@5uck1ess-plugins
```

### Recommended Companions

Devkit handles enforcement and orchestration. These plugins handle complementary concerns with no overlap.

**[Superpowers](https://github.com/obra/superpowers)** — Methodology: brainstorming, planning, TDD, verification, debugging.

```bash
/plugin marketplace add https://github.com/obra/superpowers-marketplace
/plugin install superpowers
```

**[Context Mode](https://github.com/mksglu/context-mode)** — Context window management: sandboxes large outputs, session continuity via SQLite, 98% context savings.

```bash
# See repo for install instructions — runs as an MCP server
```

---

## Quick Start

```bash
# Check what's available
/devkit:status

# Generate tests for your code
/devkit:test-gen src/parser.ts

# Fix all lint errors automatically
/self:lint --lint "npm run lint" --target src/

# Full PR preparation pipeline
/devkit:pr-ready

# Multi-agent code review (uses whatever CLIs you have)
/tri:review

```

---

## Commands

### Solo Commands (Claude-only, no external CLIs needed)

| Command | Description |
|---|---|
| `/devkit:test-gen` | Generate test suite — writes tests, runs them, fixes failures |
| `/devkit:doc-gen` | Generate documentation from code analysis |
| `/devkit:pr-ready` | Full PR pipeline — lint, test, security, changelog, create PR |
| `/devkit:onboard` | Generate codebase onboarding guide for new contributors |
| `/devkit:changelog` | Generate structured changelog from git history |
| `/devkit:workflow` | Run user-defined YAML workflows from `workflows/` |
| `/devkit:bugfix` | Full bug fix lifecycle — reproduce, diagnose, fix, regression test, verify |
| `/devkit:feature` | Full feature lifecycle — brainstorm, plan, implement, test, lint, review |
| `/devkit:refactor` | Full refactor lifecycle — analyze, plan, restructure, verify, compare |
| `/devkit:research` | Deep research — clarify, parallel search, analyze sources, synthesize |
| `/devkit:decompose` | Goal decomposition — break into task DAG, assign agents, execute in dependency order |
| `/devkit:scrape` | URL-to-Markdown conversion via Jina Reader / Firecrawl / WebFetch |
| `/devkit:audit` | Full project health audit — deps, vulnerabilities, licenses, lint, security |
| `/devkit:status` | Health check — installed CLIs, available agents, ready commands |

### Self-Improvement Loops (Claude-only)

Automated propose → measure → keep/discard → repeat cycles.

| Command | Description |
|---|---|
| `/self:improve` | General-purpose improvement loop with custom metric gate |
| `/self:test` | Iteratively generate tests until coverage target is hit |
| `/self:lint` | Iteratively fix lint/type errors until zero remain |
| `/self:perf` | Iteratively optimize with benchmark as the gate |
| `/self:migrate` | Incremental migration (JS→TS, class→hooks, etc.) with test gate |

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
| `security-auditor` | Opus | Worktree | High | 10 | tri:security, pr-ready |

---

## Skills

Coding methodology guides that enforce consistent practices. These are loaded as reference material when relevant commands run. For brainstorming, planning, TDD, verification, and skill authoring — use [superpowers](https://github.com/obra/superpowers) (complementary plugin).

| Skill | Description |
|---|---|
| `devkit:executing` | Execute plans methodically — understand, implement, verify, commit |
| `devkit:clean-code` | Meaningful names, small functions, single responsibility, flat nesting |
| `devkit:dry` | Rule of Three, when duplication is fine, extracting the right abstraction |
| `devkit:yagni` | Build only what's needed, no speculative features or premature abstractions |
| `devkit:creating-workflows` | How to create workflow YAML files — schema, step types, interpolation |
| `devkit:stuck` | Detect agent looping/failing, structured recovery — backtrack, simplify, escalate |

---

## RTK Token Optimization

Devkit includes an optional [RTK](https://github.com/rtk-ai/rtk) integration that compresses Bash command output before it reaches the context window — **60-90% token savings** on common operations.

| Operation | Before | After | Savings |
|---|---|---|---|
| Directory listing | ~2,000 tokens | ~400 tokens | 80% |
| Test output | ~25,000 tokens | ~2,500 tokens | 90% |
| Git operations | ~3,000 tokens | ~600 tokens | 80% |
| Search results | ~16,000 tokens | ~3,200 tokens | 80% |

**How it works:** A `PreToolUse` hook rewrites Bash commands through RTK (e.g., `git status` → `rtk git status`). If RTK is not installed, the hook is a no-op — everything works normally, just without compression.

**Install RTK:**
```bash
brew install rtk
```

Verify with `/devkit:status` — RTK will show as installed with version.

## Safety Hooks

Devkit includes `PreToolUse` hooks that automatically protect against dangerous operations and log all commands. Installed with the plugin — no setup required.

### Audit Trail

Every Bash command is logged to `.devkit/audit.log` with UTC timestamps. The log auto-rotates at 10k lines. This file is gitignored — it stays local only.

### Blocked (hard stop)

| Pattern | Why |
|---|---|
| `rm -rf /`, `rm -rf ~`, `rm -rf .` | Filesystem destruction |
| `DROP TABLE`, `DROP DATABASE`, `TRUNCATE` | Database destruction |
| `DELETE FROM` without `WHERE` | Unbounded data deletion |
| `dd if=... of=/dev/`, `mkfs` | Disk/partition destruction |
| Writing to `.pem`, `.key`, `.p12`, `.pfx` files | Private key overwrite |

### Prompted (asks for confirmation)

| Pattern | Why |
|---|---|
| `git push --force` to main/master | Destroys remote history |
| `git reset --hard` | Discards uncommitted work |
| `git checkout -- .` | Discards all unstaged changes |
| `git clean -f` | Permanently removes untracked files |
| `git branch -D main/master` | Deletes primary branch |
| `chmod -R 777` | World-writable permissions |
| `sudo rm` | Elevated privilege deletion |
| Editing `.env`, credentials, secrets, tokens | May contain sensitive data |

### Edit-Time Security Patterns

A `PreToolUse` hook on Edit/Write that catches known vulnerability patterns at the moment of creation. Warns once per file+pattern per session.

| Language | Patterns detected |
|---|---|
| JS/TS | `eval()`, `new Function()`, `dangerouslySetInnerHTML`, `.innerHTML =`, `document.write`, `child_process.exec()`, weak hashes (MD5/SHA-1), `Math.random` for security |
| Python | `eval()`, `exec()`, `pickle.load`, `os.system`, `subprocess(shell=True)`, `yaml.load`, `__import__`, weak hashes |
| Go | Shell exec via `sh -c`, weak hashes, string concat in SQL |
| Any | SQL injection via string concatenation, hardcoded secrets |

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
├── manifest.json            # Plugin manifest
├── ROADMAP.md               # Implemented features and future plans
├── PREFERENCES.md           # Agent behavior guidelines and coding standards
├── commands/                # Claude Code skills
│   ├── tri-review.md        # Multi-agent review
│   ├── tri-dispatch.md      # Multi-agent dispatch
│   ├── tri-debug.md         # Multi-agent debugging
│   ├── tri-test-gen.md      # Multi-agent test generation
│   ├── tri-security.md      # Multi-agent security audit
│   ├── self-improve.md      # General improvement loop
│   ├── self-test.md         # Test coverage loop
│   ├── self-lint.md         # Lint fix loop
│   ├── self-perf.md         # Performance optimization loop
│   ├── self-migrate.md      # Migration loop
│   ├── test-gen.md          # Solo test generation
│   ├── doc-gen.md           # Documentation generation
│   ├── pr-ready.md          # PR preparation pipeline
│   ├── onboard.md           # Codebase onboarding
│   ├── changelog.md         # Changelog generation
│   ├── workflow.md          # YAML workflow runner
│   ├── status.md            # Health check
│   ├── bugfix.md            # Bug fix lifecycle
│   ├── feature.md           # Feature development lifecycle
│   ├── refactor.md          # Refactoring lifecycle
│   ├── research.md          # Deep research workflow
│   └── decompose.md         # Goal decomposition into task DAG
├── agents/                  # Agent configs
│   ├── reviewer.md
│   ├── researcher.md
│   ├── improver.md
│   ├── test-writer.md
│   ├── documenter.md
│   └── security-auditor.md
├── skills/                  # Coding methodology guides
│   ├── planning.md
│   ├── executing.md
│   ├── writing-tests.md
│   ├── clean-code.md
│   ├── dry.md
│   ├── yagni.md
│   ├── brainstorming.md
│   ├── skill-authoring.md
│   ├── creating-workflows.md
│   ├── stuck.md
│   └── verify.md
├── hooks/                   # Safety + optimization hooks
│   ├── hooks.json           # Hook config (auto-loaded by plugin)
│   ├── safety-check.sh      # Dangerous operation blocker
│   └── rtk-rewrite.sh       # RTK token optimization (optional)
├── workflows/               # YAML workflow definitions
│   ├── feature.yml          # Full feature lifecycle
│   ├── bugfix.yml           # Bug fix lifecycle
│   ├── refactor.yml         # Refactor lifecycle
│   ├── research.yml         # Deep research pipeline
│   ├── self-improve.yml     # Metric-gated improvement loop
│   ├── self-test.yml        # Test fix loop
│   ├── self-lint.yml        # Lint fix loop
│   ├── self-perf.yml        # Performance optimization loop
│   ├── tri-review.yml       # Three-tier code review
│   ├── tri-dispatch.yml     # Three-tier task dispatch
│   ├── tri-debug.yml        # Three-tier debugging
│   └── tri-security.yml     # Three-tier security audit
├── presets/                  # Reusable prompt templates (planned)
│   └── .gitkeep
├── .github/workflows/       # CI/CD
│   ├── ci.yml               # Build + test + vet on push/PR
│   └── release.yml          # Auto-tag + release on version bump
└── src/                     # Go CLI harness
    ├── go.mod
    ├── main.go
    ├── Makefile
    ├── cmd/                 # Cobra commands
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
- **Crash recovery** — SQLite state + handoff files survive crashes
- **Hard budget caps** — stops spawning at your dollar limit
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

# Resume a crashed session
devkit resume abc123def456

# Check what happened
devkit status
```

### Testing

```bash
cd src && go test ./... -v
```

76+ tests across 4 packages (lib, runners, loops, cmd). Loop tests use mock runners — no API calls needed.

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

Check status with `/devkit:status`.

### Codex Plugin Commands

Installing [codex-plugin-cc](https://github.com/openai/codex-plugin-cc) gives you these additional commands:

| Command | Description |
|---|---|
| `/codex:rescue` | Delegate a task to Codex (investigation, fixes, continuation) |
| `/codex:review` | Code review via Codex (compare against base branch) |
| `/codex:adversarial-review` | Adversarial review — Codex actively tries to break your code |
| `/codex:result` | Retrieve results from a background Codex task |
| `/codex:status` | Check status of running Codex tasks |

**Flags:** `--model gpt-5.4` / `gpt-5.4-mini` / `gpt-5.4-nano`, `--effort high`, `--background`, `--wait`, `--resume`, `--fresh`

Devkit's `tri:*` commands use `/codex:rescue --background` for multi-agent dispatch.

### Gemini Plugin Commands

Installing [gemini-plugin-cc](https://github.com/abiswas97/gemini-plugin-cc) gives you these additional commands:

| Command | Description |
|---|---|
| `/gemini:rescue` | Delegate a task to Gemini (investigation, fixes, continuation) |
| `/gemini:review` | Code review via Gemini |
| `/gemini:adversarial-review` | Adversarial review — Gemini challenges design decisions |
| `/gemini:result` | Retrieve results from a background Gemini task |
| `/gemini:status` | Check status of running Gemini tasks |

Devkit's `tri:*` commands use `/gemini:rescue --background` for multi-agent dispatch.

---

## Roadmap

See [ROADMAP.md](ROADMAP.md) for full details.

- [x] Go CLI harness — 9 commands, SQLite state, crash recovery, budget enforcement, multi-agent support
- [x] CI/CD pipeline — build, vet, test, auto-release on version bump
- [x] Branch protection — PRs required for main
- [ ] Cost event hooks — budget threshold events with auto-downgrade actions
- [ ] Execution registry — centralized step tracking with timing and token usage
- [ ] Preset library — curated prompt templates for common review/improvement scenarios

---

## References

### Token Optimization

| Tool | Description | Link |
|---|---|---|
| RTK | Rust Token Killer — 60-90% token savings on Bash output | [GitHub](https://github.com/rtk-ai/rtk) |

### Multi-Agent & Orchestration

| Tool | Description | Link |
|---|---|---|
| pthd | mprocs-based parallel agent panes | [GitHub](https://github.com/pandego/parallel-thread-skill) |
| OpenClaw | Personal AI assistant platform | [GitHub](https://github.com/openclaw/openclaw) |
| claw-multi-agent | OpenClaw parallel orchestration | [GitHub](https://github.com/zcyynl/claw-multi-agent) |
| NemoClaw | NVIDIA sandboxed OpenClaw runtime | [GitHub](https://github.com/NVIDIA/NemoClaw) |
| GSD-2 | Autonomous project execution (milestone→task) | [GitHub](https://github.com/gsd-build/gsd-2) |
| metaswarm | 18 agents, 13 skills for Claude/Gemini/Codex | [GitHub](https://github.com/dsifry/metaswarm) |
| skill-codex | Claude Code ↔ Codex bridge | [GitHub](https://github.com/skills-directory/skill-codex) |
| codex-plugin-cc | Official OpenAI Codex plugin for Claude Code | [GitHub](https://github.com/openai/codex-plugin-cc) |
| gemini-plugin-cc | Gemini plugin for Claude Code (optional — standalone reviews & task delegation) | [GitHub](https://github.com/abiswas97/gemini-plugin-cc) |

### Skills & Marketplaces

| Tool | Description | Link |
|---|---|---|
| superpowers-marketplace | Curated Claude Code plugin marketplace | [GitHub](https://github.com/obra/superpowers-marketplace) |
| superpowers-skills | Community skills for superpowers | [GitHub](https://github.com/obra/superpowers-skills) |
| awesome-claude-skills | 50+ verified skills collection | [GitHub](https://github.com/karanb192/awesome-claude-skills) |
| taste-skill | Frontend design quality for AI agents | [GitHub](https://github.com/Leonxlnx/taste-skill) |
| claude-code-system-prompts | Claude Code system prompt collection | [GitHub](https://github.com/Leonxlnx/claude-code-system-prompts) |
| rune | Lean skill ecosystem | [GitHub](https://github.com/Rune-kit/rune) |

### TDD & Quality

| Tool | Description | Link |
|---|---|---|
| agent-skill-tdd | TDD + requirements workflow | [GitHub](https://github.com/Shelpuk-AI-Technology-Consulting/agent-skill-tdd) |
| pdca-code-generation | Plan-Do-Check-Act with TDD | [GitHub](https://github.com/kenjudy/pdca-code-generation-process) |
| claude-wizard | 8-phase dev with TDD + adversarial testing | [GitHub](https://github.com/vlad-ko/claude-wizard) |

### Official Claude Code Plugins

Install with `/plugin install <name>@claude-plugins-official`:

| Plugin | Description |
|---|---|
| `code-review` | Code review a PR |
| `pr-review-toolkit` | Multi-agent PR review |
| `commit-commands` | Commit, push, open PR |
| `feature-dev` | Guided feature development |
| `hookify` | Create hooks from conversation |
| `code-simplifier` | Code quality/efficiency review |
| `security-guidance` | Security-focused review |
| `skill-creator` | Create new skills |
| `plugin-dev` | Create plugins end-to-end |
| `claude-md-management` | Update CLAUDE.md with learnings |
