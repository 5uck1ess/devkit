# Devkit Roadmap

## Implemented

- **MCP engine** — Go server exposes `devkit_start`, `devkit_advance`, `devkit_status`, `devkit_list` tools inside Claude Code. Step ordering enforced via MCP tool scoping + PreToolUse hook exit 2. Session state in session.json (hot path, <50ms hook reads) + SQLite (cold history). ~65% token reduction vs old monolithic prompts.
- **Skills-first architecture** — All entry points are skills in `skills/` (tab-completable slash commands in current Claude Code; bare names like `/tri-review` work, `/devkit:<name>` form also works for disambiguation). Primary user-facing commands: `/tri-review`, `/tri-debug`, `/tri-security`, `/health`, `/setup-rules` (user-only via `disable-model-invocation`). Every workflow also has a dedicated skill for natural-language dispatch. The `commands/` directory is retained for backward compat but empty of new entries — redundant tri-* command files were removed (skills take precedence per Claude Code docs) and the generic `/devkit:workflow <name>` runner was removed since every workflow now has its own slash command (`/feature`, `/bugfix`, etc.).
- **Deterministic workflow conversion** — All command logic moved from LLM-interpreted markdown to Go-engine-driven YAML workflows; ~3,600 lines of inline logic removed
- **39 skills** — 22 workflow trigger skills (feature, bugfix, refactor, audit, harness-audit, research, deep-research, pr-ready, autoloop, test-gen, doc-gen, onboard, tri-review, tri-debug, tri-security, tri-dispatch, self-audit, self-improve, self-lint, self-migrate, self-perf, self-test) + 7 coding principles (executing, clean-code, DRY, YAGNI, dont-reinvent, stuck, scratchpad) + 4 tools (gcli, scrape, screenshot, browser) + 1 meta-orchestration (mega-pr) + 2 content (changelog, adr) + 1 reference (creating-workflows)
- **Deterministic skill dispatch for every workflow** — Every one of the 21 workflows has a natural-language trigger skill with keyword-rich description. Saying "build a feature", "tri review", "deep research X", etc. deterministically invokes the matching skill, which calls `devkit_start` and the engine enforces every step from there. Closes the entry-gate non-determinism where 11/18 workflows previously had no natural-language path. Skill tool added to the guard allowlist so nested mid-workflow skill dispatch works.
- **6 agents** — Scoped tool access, worktree isolation, model assignment
- **12 hooks** — Safety (destructive command blocking, edit-time security patterns, PR gate), observability (audit trail, slop detection, post-validation, subagent verification, language-aware code review), optimization (RTK token compression), workflow enforcement (devkit-guard, devkit-stop-guard)
- **Graceful degradation** — tri:* commands work with 1-3 agents depending on installed CLIs
- **Goal decomposition** — Task DAG with dependency ordering and parallel execution
- **Concurrency limits** — Max 3 parallel agents in multi-agent commands
- **Early-exit conditions** — Self-improvement loops stop when goal is met, not just at max iterations
- **Token budget guidance** — Per-command budget recommendations with model downgrade patterns
- **RTK token optimization** — Optional PreToolUse hook compresses Bash output via RTK (60-90% savings)
- **22 YAML workflows** — Portable workflow definitions (feature, bugfix, refactor, research, deep-research, autoloop, harness-audit, self-*, tri-*, test-gen, doc-gen, onboard)
- **Separate marketplace** — Multi-plugin marketplace at `5uck1ess/marketplace`
- **Companion ecosystem** — Evaluated official marketplace, documented holistic setup with 7 complementary plugins
- **Hypothesis-driven perf** — Evidence gathering, ranked hypotheses, one-at-a-time testing replaces blind benchmark loops
- **Post-PR monitoring** — CI watching + iterative reviewer comment resolution
- **AST repo mapping** — Symbol index with dependency graph, cached for agent navigation
- **Generic YAML workflow engine** — Deterministic step execution, branching, loops, parallel dispatch, budget enforcement in compiled Go
- **Triage-based phase skipping** — TINY/SMALL/MEDIUM/LARGE classification with fast paths
- **Iteration scratchpads** — Persistent memory across loop iterations to prevent repeated failures
- **Cross-domain dirty-bit enforcement** — Blocks completion without test evidence per domain
- **Language-universal hooks** — Consolidated language-specific hooks into `lang-review.sh` with Go, TypeScript, Rust, Python, and Shell support
- **Hook consolidation** — Merged 14 hooks into 10, reduced per-edit shell processes from 7 to 4

## Retired

Items below were on the roadmap but determined to be unnecessary — either already solved by existing infrastructure or too speculative to justify the complexity.

| Item | Why removed |
|------|-------------|
| Stop hook redesign | Still fires every turn, but exits early with `approve` when no files are changed — near-instant on clean trees, so the performance concern is moot. Revisit only if it causes measurable latency. |
| Cost event hooks | Budget enforcement already exists in the Go engine via `overBudget()` + `addCost()` callbacks with hard limits |
| Execution registry | Step tracking already handled by SQLite via `lib.DB` with status, cost, and timing per step |
| Preset library | The 18 YAML workflows and 22 skills already serve this purpose |
| Framework-specific review checklists | `lang-review.sh` covers language-level patterns; framework-specific rules are better added per-project via hookify |
| Conditional hook firing | Hooks already self-filter internally (extension checks, changed-file checks); a generic condition system adds complexity for no current need |
