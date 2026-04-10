# Changelog

## 2.1.2

### Make the MCP engine actually work + tighten enforcement (PR #54)

Ships the critical fix that 2.1.0 missed, plus the determinism gaps that a full mega-pr review on #52 uncovered. Restores the engine to a state where step enforcement is not bypassable and command steps are not shell-injectable.

#### Fixed (CRITICAL)
- **Engine binary now actually ships with the plugin.** 2.1.0/2.1.1 set `mcpServers.devkit-engine.command` to `${CLAUDE_PLUGIN_ROOT}/bin/devkit` but `bin/` was gitignored, so the marketplace install had no binary. MCP server silently failed to start and every workflow silently fell back to direct agent dispatch. Fixed by committing a POSIX shell wrapper at `bin/devkit` that downloads the matching release asset on first run, verifies SHA256, caches it next to itself, and execs it. Local dev builds (`make install-plugin`) are used directly via a fast path.
- **Shell injection in command steps.** `runCommand` interpolated `{{input}}` and `{{step-id}}` into `sh -c <command>`. LLM-chosen input or contaminated prior-step output could execute arbitrary shell. Fixed: command and gate strings are now literal YAML; values are passed via `$DEVKIT_INPUT` and `$DEVKIT_OUT_<step_id>` env vars. Workflow validator rejects `{{...}}` in command/gate strings at parse time with an explicit "shell injection mitigation" error. The 5 self-* workflows were updated to use `$DEVKIT_INPUT`.
- **PreToolUse guard was trivially bypassable.** The blocklist only matched 11 hardcoded tool names (Bash, Edit, Write, ...) and missed Task, SlashCommand, TodoWrite, mcp__*, and any new tools Claude Code adds. Fixed: guard now uses an allowlist — during command steps, only `mcp__devkit*` tools and `TodoWrite` are permitted. Everything else returns exit 2.
- **Stop guard failed open on corrupt session.json, guard failed closed.** Opposite policies on the same condition meant workflow completion could be silently approved on a corrupt state file. Fixed: both hooks now fail closed with a clear diagnostic when JSON parsing fails.
- **Loop `until` matched substrings.** `strings.Contains("no failures found", "fail")` returned true, silently terminating loops. Same bug in branch `when:` matching. Fixed: both now use word-boundary matching (grep -w semantics) — `fail` won't match `failures`.

#### Fixed (HIGH)
- `completeWorkflow` ignored `git.CommitAll` errors — users saw "WORKFLOW COMPLETE" while their branch work was unpersisted. Now surfaces warnings in the result text and on stderr.
- `runCommand` threw away command output on non-`*exec.ExitError` failures. Now returns the combined stream alongside the error, and distinguishes `context.DeadlineExceeded` (5-minute timeout) from "command crashed" with exit 124.
- Hook fail-silent when `CLAUDE_PLUGIN_DATA` is unset. Now emits a stderr warning so the degraded state is observable.
- TOCTOU race between hook `[[ -f session.json ]]` check and python `open()`: if the engine cleared the file between those two, hooks spuriously failed closed. Python code now handles `FileNotFoundError` explicitly as "no session".

#### Fixed (MEDIUM)
- YAML workflow parsing now uses `KnownFields(true)` — typos like `commnd:` fail loudly instead of silently dropping the field.
- `make sync-version` no longer downgrades `plugin.json`. It only writes when the git tag is strictly higher than the current version, which prevents local dev builds on feature branches from clobbering manual version bumps.
- `bin/devkit` wrapper emits `devkit:` diagnostics to stderr on download, checksum, install, and failure paths. Stdout stays clean so it doesn't corrupt the MCP stdio protocol.

#### Added
- **CI `fresh-install-smoke` job.** Runs on every PR in a clean checkout. Asserts `bin/devkit` exists and is executable, that `bin/devkit-engine` is not tracked, that `plugin.json`'s MCP command matches the wrapper path, runs `shellcheck bin/devkit`, and runs `./bin/devkit --version` to confirm it either succeeds or fails loudly — never exits 0 with empty output. This is the check that would have caught the v2.1.0 binary-shipping bug before merge.
- Regression tests: `TestRunWorkflowCommandRejectsInterpolation`, `TestRunWorkflowCommandEnvInput`, `TestRunWorkflowCommandEnvPriorOutput`, `TestLoopUntilRejectsSubstring`.

#### Removed
- `presets/` — empty v1 directory with only `.gitkeep`, no references anywhere.

## 2.1.0

### MCP Engine — Deterministic Workflow Enforcement (PR #52)

Replaces the broken subprocess-spawning engine with an MCP server that runs inside Claude Code. Step skipping is now structurally impossible.

#### Added
- **MCP server** (`src/mcp/`) — Go server exposes 4 tools: `devkit_start`, `devkit_advance`, `devkit_status`, `devkit_list`. Registered via `mcpServers` in plugin.json.
- **PreToolUse guard hook** (`hooks/devkit-guard.sh`) — reads `session.json`, blocks Bash/Edit/Write/Read/Glob/Grep/Agent/WebFetch/WebSearch/NotebookEdit/Skill during command steps (exit 2).
- **Stop guard hook** (`hooks/devkit-stop-guard.sh`) — blocks session end if workflow is incomplete.
- **Condensed principles** (`skills/_principles.yml`) — ~120 tokens of DRY/YAGNI/clean-code/dont-reinvent/executing/scratchpad/stuck/test-gen rules injected per workflow step instead of loading full skill files.
- **Hot session state** (`src/lib/state_json.go`) — atomic write to `$CLAUDE_PLUGIN_DATA/session.json` for fast hook reads (<50ms).
- **Workflow YAML extensions** — `enforce` (hard/soft), `branch` (git branch per session), `principles` (per-workflow and per-step override).
- **New MCP tools** — 6 integration tests covering lifecycle, loops with gates, principle injection, expect-failure, path traversal rejection.

#### Changed
- **Engine role** — CLI that spawned subprocesses → MCP server + state machine
- **Claude runner** — `claude -p` subprocess (broken with OAuth) → Claude Code IS the runner
- **Enforcement** — None (markdown honor system) → MCP tool scoping + PreToolUse exit 2
- **Principle skills** — Loaded if Claude decided to → injected by engine per step
- **Token usage** — ~50k+ for 8-step workflow → ~17k (~65% reduction)
- **Skills and commands** — All 8 entry points (research, deep-research, autoloop, tri-review, tri-debug, tri-security, pr-ready, status) now use MCP tools instead of `ensure-engine.sh` + `devkit workflow`

#### Removed
- `scripts/ensure-engine.sh` — no longer needed; replaced by the committed `bin/devkit` wrapper in 2.1.2 (see below), which downloads the engine binary on first run and caches it next to itself. In 2.1.0 the wrapper was missing entirely, which shipped a broken plugin — 2.1.2 fixes that and adds CI smoke tests to prevent regression.
- `scripts/install-engine.sh` — installed by plugin manifest

#### Fixed
- Engine can now run inside Claude Code (was impossible with OAuth tokens and `claude -p` subprocess)

## 2.0.34

### Deterministic Workflow Conversion (PRs #38–#45)

Major architectural shift: all command logic moved from LLM-interpreted markdown to Go-engine-driven YAML workflows.

#### Changed
- **24 → 8 slash commands** — 16 commands deleted, logic now in YAML workflows invoked via `devkit workflow <name> "<description>"` or context-activated skills
- **4 ultra-thin wrappers** — tri-review, tri-debug, tri-security, pr-ready (one-liner pointing to workflow)
- **4 kept as-is** — pr-monitor, status, setup-rules, workflow
- **~3,600 lines removed** across PRs 1–5

#### Added
- **`expect` field** on command steps — `expect: success` (fail on non-zero) and `expect: failure` (fail on zero exit). Enables bugfix reproduction gates.
- **3 new YAML workflows** — `self-migrate.yml`, `audit.yml`, `pr-ready.yml`
- **`until: DONE` fix** — All self-improvement loops now use LLM output matching instead of `exit code: 0`

#### Fixed
- Self-improvement loop termination — `until` checks LLM text output, not gate exit code
- Research skill escalation — all 4 trigger conditions restored in fallback
- Stale README/status.md command references after rebase

## 2.0.1

### Fixed
- Migrate all 20 command files from `commands/` into `skills/` — completes the command-to-skill conversion started in v2.0.0
- Restore `"commands"` key in manifest for 20 deterministic workflows (tab-completable), keep `"skills"` for 14 context-activated entries — both point to `skills/` directory
- Fix `manifest.json` version (was stuck at `1.3.5` while `plugin.json` was at `2.0.0`)
- Remove duplicate `skills/` block in README repository structure
- Fix orphaned `skill-authoring` reference in CONTRIBUTING.md

## 2.0.0

### New Commands
- Add `devkit:audit` — unified project health audit (deps, vulnerabilities, outdated packages, licenses, lint, security) with scored report
- Add `devkit:pr-monitor` — post-PR review monitor that watches CI and iteratively resolves reviewer comments
- Add `devkit:repo-map` — AST-based symbol index with dependency graph, cached to .devkit/repo-map.json

### New Hooks
- Add `security-patterns` — PreToolUse on Edit/Write catches eval, XSS, shell injection, weak hashes, hardcoded secrets across JS/TS/Python/Go
- Add `audit-trail` — logs all Bash commands to `.devkit/audit.log` with UTC timestamps, auto-rotates at 10k lines
- Add `slop-detect` — PostToolUse on Edit/Write catches excessive docs, restating comments, JSDoc overuse
- Add `pr-gate` — prompts to run pr-ready pipeline before `gh pr create`, 10-minute cooldown

### Upgraded
- Upgrade `self:perf` to hypothesis-driven investigation — evidence gathering, ranked hypotheses, one-at-a-time testing with 3x benchmark runs

### Companion Ecosystem
- Define holistic setup: devkit + superpowers + feature-dev + pr-review-toolkit + commit-commands + hookify + skill-creator + context-mode
- Evaluated all official marketplace plugins — documented which to install, which to skip, and why
- Fix superpowers install: use `@claude-plugins-official`, not separate marketplace

### Skills
- Convert 6 commands to context-activated skills (no slash command needed): test-gen, doc-gen, changelog, onboard, research, scrape
- Add `dont-reinvent` skill — prefer existing solutions over custom code, reduce maintenance burden
- Add `gcli` skill — Google Workspace CLI reference with safety confirmation gate
- Remove 5 skills that overlap with superpowers: brainstorming, planning, writing-tests, skill-authoring, verify
- Evaluated `code-simplifier` as replacement — rejected (thin, React-specific, no test verification)
- Commands reduced from 26 to 20, skills increased from 6 to 14

### Fixes
- Fix stop-gate: disabled — fires every turn, not just session end. Needs architectural redesign.
- Fix stop-gate conflict marker false positive — grep pattern was matching its own source code

### Docs
- Complete README overhaul with companion ecosystem diagram and holistic setup guide
- Added "why these and not others" section explaining plugin selection rationale
- Updated roadmap with completed and planned items

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
