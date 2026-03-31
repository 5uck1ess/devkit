# Devkit

Multi-agent orchestration plugin for Claude Code. Dispatch tasks to Claude, Codex, and Gemini in parallel, run self-improvement loops, and get triple-agent code reviews.

## Install

```bash
/plugin install github.com/5uck1ess/devkit
```

---

## Skills

| Skill | Description |
|---|---|
| `/tri:review` | Triple-agent PR review — same prompt to Claude + Codex + Gemini, consolidated report |
| `/tri:dispatch` | Any task to all 3 agents in parallel, compare outputs |
| `/self:improve` | Recursive improvement loop — propose → measure → keep/discard → repeat |

> Single-agent dispatch (e.g., "run this through codex") doesn't need a skill — just ask Claude directly.

### Autonomy Flags

Set automatically in each skill:

| Agent | Flags |
|---|---|
| Claude | `--dangerously-skip-permissions` |
| Codex | `--full-auto --dangerously-bypass-approvals-and-sandbox` |
| Gemini | `-y` |

### Examples

```bash
# Triple PR review
/tri:review check for DRY violations and over-engineering

# Self-improvement loop
/self:improve --target src/ --metric "npm test" --objective "fix all failing tests"
```

---

## Agents

Thin configs — no persona prompts, just execution constraints. Skills provide the instructions.

| Agent | Model | Isolation | Effort | Max Turns | Used by |
|---|---|---|---|---|---|
| `reviewer` | Opus | Worktree | High | 10 | tri:review |
| `researcher` | Sonnet | Worktree | Medium | 15 | tri:dispatch |
| `improver` | Opus | Worktree | High | 10 | self:improve |

---

## Architecture

```
tri:review
  ├── Claude  → native background agent (orchestrator sees summary only)
  ├── Codex   → codex exec --full-auto (CLI, background process)
  └── Gemini  → gemini -p -y (CLI, background process)
```

---

## Repository Structure

```
devkit/
├── manifest.json          # Plugin manifest
├── commands/              # Claude Code skills
│   ├── tri-review.md
│   ├── tri-dispatch.md
│   └── self-improve.md
├── agents/                # Agent configs
│   ├── reviewer.md
│   ├── researcher.md
│   └── improver.md
└── src/
    └── TODO.md            # Go harness roadmap
```

---

## Roadmap

- [ ] Go CLI harness for deterministic loop control, process management, and unattended runs (see `src/TODO.md`)

---

## Prerequisites

Requires [Codex CLI](https://github.com/openai/codex) and [Gemini CLI](https://github.com/google-gemini/gemini-cli) for multi-agent dispatch:

```bash
brew install codex gemini-cli
```

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
