# Devkit

Guardrails and consistency for Claude Code.

AI agents are powerful but unpredictable — they skip steps, jump to conclusions, and refactor things you didn't ask them to touch. Devkit enforces deterministic, step-by-step workflows that keep Claude on track: propose one change, measure it, keep or revert, repeat. No freestyling.

Every command follows a defined sequence. Self-improvement loops gate each change behind a metric. Multi-agent commands dispatch the same task to multiple models and consolidate consensus. The result is reproducible, auditable work — not whatever Claude felt like doing.

Works with just Claude. Optionally adds Codex and Gemini for multi-perspective analysis.

## Install

```bash
/plugin install github.com/5uck1ess/devkit
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

Coding methodology guides that enforce consistent practices. These are loaded as reference material when relevant commands run.

| Skill | Description |
|---|---|
| `devkit:planning` | Write implementation plans — small testable steps, dependency ordering, risk identification |
| `devkit:executing` | Execute plans methodically — understand, implement, verify, commit |
| `devkit:writing-tests` | Arrange-act-assert, one behavior per test, test contracts not implementation |
| `devkit:clean-code` | Meaningful names, small functions, single responsibility, flat nesting |
| `devkit:dry` | Rule of Three, when duplication is fine, extracting the right abstraction |
| `devkit:yagni` | Build only what's needed, no speculative features or premature abstractions |
| `devkit:brainstorming` | Diverge/converge ideation, evaluating ideas, avoiding premature commitment |

---

## Safety Hooks

Devkit includes a `PreToolUse` hook that automatically protects against dangerous operations. Installed with the plugin — no setup required.

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
  ├── Codex   → CLI subprocess (if installed)
  └── Gemini  → CLI subprocess (if installed)

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
│   └── status.md            # Health check
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
│   └── brainstorming.md
├── hooks/                   # Safety hooks
│   ├── hooks.json           # Hook config (auto-loaded by plugin)
│   └── safety-check.sh      # Dangerous operation blocker
├── workflows/               # User-defined YAML workflows
│   └── .gitkeep
├── presets/                  # Reusable prompt templates (planned)
│   └── .gitkeep
└── src/
    └── TODO.md              # Go harness roadmap
```

---

## Autonomy Flags

Set automatically in each multi-agent command:

| Agent | Flags |
|---|---|
| Claude | `--dangerously-skip-permissions` |
| Codex | `--full-auto --dangerously-bypass-approvals-and-sandbox` |
| Gemini | `-y` |

---

## Prerequisites

**Required:** Claude Code (you're already here)

**Optional** (for multi-agent commands):
```bash
brew install codex gemini-cli
```

Check status with `/devkit:status`.

---

## Roadmap

- [ ] Go CLI harness for deterministic loop control, process management, and unattended runs (see `src/TODO.md`)
- [ ] More presets (Python security, Go performance, etc.)
- [ ] Preset library — curated prompt templates for common review/improvement scenarios

---

## References

### Multi-Agent & Orchestration

| Tool | Description | Link |
|---|---|---|
| pthd | mprocs-based parallel agent panes | [GitHub](https://github.com/pandego/parallel-thread-skill) |
| skill-codex | Claude Code ↔ Codex bridge | [GitHub](https://github.com/skills-directory/skill-codex) |
| OpenClaw | Personal AI assistant platform | [GitHub](https://github.com/openclaw/openclaw) |
| claw-multi-agent | OpenClaw parallel orchestration | [GitHub](https://github.com/zcyynl/claw-multi-agent) |
| NemoClaw | NVIDIA sandboxed OpenClaw runtime | [GitHub](https://github.com/NVIDIA/NemoClaw) |
| GSD-2 | Autonomous project execution (milestone→task) | [GitHub](https://github.com/gsd-build/gsd-2) |
| metaswarm | 18 agents, 13 skills for Claude/Gemini/Codex | [GitHub](https://github.com/dsifry/metaswarm) |

### Skills & Marketplaces

| Tool | Description | Link |
|---|---|---|
| superpowers-marketplace | Curated Claude Code plugin marketplace | [GitHub](https://github.com/obra/superpowers-marketplace) |
| superpowers-skills | Community skills for superpowers | [GitHub](https://github.com/obra/superpowers-skills) |
| awesome-claude-skills | 50+ verified skills collection | [GitHub](https://github.com/karanb192/awesome-claude-skills) |
| taste-skill | Frontend design quality for AI agents | [GitHub](https://github.com/Leonxlnx/taste-skill) |
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
