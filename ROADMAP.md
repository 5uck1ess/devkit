# Devkit Roadmap

## Implemented

- **23 slash commands** — Lifecycle workflows, self-improvement loops, multi-agent dispatch, project health audit, post-PR monitoring, AST repo mapping, autoresearch-inspired self-audit, autoloop
- **17 context-activated skills** — 8 auto-trigger workflows (test-gen, doc-gen, changelog, onboard, research, deep-research, scrape, autoloop) + 6 coding principles (executing, clean-code, DRY, YAGNI, dont-reinvent, stuck) + 2 tools (gcli, creating-workflows) + 1 iteration memory (scratchpad)
- **6 agents** — Scoped tool access, worktree isolation, model assignment
- **10 hooks** — Safety (destructive command blocking, edit-time security patterns, PR gate), observability (audit trail, slop detection, post-validation, subagent verification, language-aware code review), optimization (RTK token compression)
- **Graceful degradation** — tri:* commands work with 1-3 agents depending on installed CLIs
- **Goal decomposition** — Task DAG with dependency ordering and parallel execution
- **Concurrency limits** — Max 3 parallel agents in multi-agent commands
- **Early-exit conditions** — Self-improvement loops stop when goal is met, not just at max iterations
- **Token budget guidance** — Per-command budget recommendations with model downgrade patterns
- **RTK token optimization** — Optional PreToolUse hook compresses Bash output via RTK (60-90% savings)
- **15 YAML workflows** — Portable workflow definitions (feature, bugfix, refactor, research, deep-research, autoloop, self-*, tri-*)
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
| Preset library | The 15 YAML workflows and 17 skills already serve this purpose |
| Framework-specific review checklists | `lang-review.sh` covers language-level patterns; framework-specific rules are better added per-project via hookify |
| Conditional hook firing | Hooks already self-filter internally (extension checks, changed-file checks); a generic condition system adds complexity for no current need |
