# Changelog

## 1.4.0

- Add `devkit:audit` command — unified project health audit (deps, vulnerabilities, outdated packages, licenses, lint, security) with scored report
- Add audit trail hook — logs all Bash commands to `.devkit/audit.log` with UTC timestamps, auto-rotates at 10k lines
- Remove 5 skills that overlap with superpowers plugin: brainstorming, planning, writing-tests, skill-authoring, verify
- Keep 6 skills that are unique to devkit: executing, clean-code, dry, yagni, creating-workflows, stuck
- Add superpowers and context-mode as recommended companion plugins in README
- Add edit-time security pattern hook — catches eval, XSS, shell injection, weak hashes, hardcoded secrets on Write/Edit
- Fix stop-gate cooldown — removed set -euo pipefail that was killing the script before writing cooldown file

## 1.3.0

- Add Go CLI harness: deterministic loop control, multi-agent dispatch, SQLite state
- Add test suite for runners and lib packages with race detection
- Add CI pipeline (GitHub Actions): build, vet, test, gofmt check
- Add `test-gen` command as composable Go primitive
- Add `feature`, `bugfix`, `refactor` commands as composable Go primitives
- Fix security findings: budget enforcement, permissions, DB error handling
- Fix consensus findings from tri-review: races, exit codes, verify-before-commit
- Add Gemini agent integration alongside Codex

## 1.2.0

- Add RTK token optimization hook — 60-90% savings on Bash output (optional, no-op if rtk not installed)
- Add 12 YAML workflow definitions from pikit: feature, bugfix, refactor, research, self-improve, self-test, self-lint, self-perf, tri-review, tri-dispatch, tri-debug, tri-security
- Move marketplace to separate repo (`5uck1ess/marketplace`) for multi-plugin support
- Update install instructions to use new marketplace URL
- Add RTK to `/devkit:status` health check

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
