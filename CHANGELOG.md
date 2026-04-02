# Changelog

## 1.1.0

- Add `decompose` command — goal decomposition into task DAG with dependency ordering
- Add 4 new skills: `skill-authoring`, `creating-workflows`, `stuck`, `verify`
- Add `ROADMAP.md` and `PREFERENCES.md`
- Add Budget & Early Exit sections to all self:* commands with stuck detection
- Add Concurrency & Budget sections to all tri:* commands with `[PARALLEL]` markers
- Add token budget guidance to bugfix, feature, refactor, research commands
- Update research command with structured `AskUserQuestion` calls

## 1.0.0

- 21 commands: solo workflows, self-improvement loops, multi-agent dispatch
- 7 skills: planning, executing, writing-tests, clean-code, dry, yagni, brainstorming
- 6 agents: reviewer, researcher, improver, test-writer, documenter, security-auditor
- Safety hooks: PreToolUse blocking destructive operations
- Graceful degradation: tri:* commands work with 1-3 agents
- Codex integration via official codex-plugin-cc
- Workflow runner for custom YAML workflows
