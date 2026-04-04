# Devkit Roadmap

## Implemented

- **20 slash commands** — Lifecycle workflows, self-improvement loops, multi-agent dispatch, project health audit, post-PR monitoring, AST repo mapping
- **14 skills** — 6 context-activated workflows (test-gen, doc-gen, changelog, onboard, research, scrape) + 6 coding principles (executing, clean-code, DRY, YAGNI, dont-reinvent, stuck) + 2 tools (gcli, creating-workflows)
- **6 agents** — Scoped tool access, worktree isolation, model assignment
- **8 hooks** — Safety (destructive command blocking, edit-time security patterns, PR gate), observability (audit trail, slop detection, post-validation, subagent verification), optimization (RTK token compression)
- **Graceful degradation** — tri:* commands work with 1-3 agents depending on installed CLIs
- **Goal decomposition** — Task DAG with dependency ordering and parallel execution
- **Concurrency limits** — Max 3 parallel agents in multi-agent commands
- **Early-exit conditions** — Self-improvement loops stop when goal is met, not just at max iterations
- **Token budget guidance** — Per-command budget recommendations with model downgrade patterns
- **RTK token optimization** — Optional PreToolUse hook compresses Bash output via RTK (60-90% savings)
- **12 YAML workflows** — Portable workflow definitions (feature, bugfix, refactor, research, self-*, tri-*)
- **Separate marketplace** — Multi-plugin marketplace at `5uck1ess/marketplace`
- **Companion ecosystem** — Evaluated official marketplace, documented holistic setup with 7 complementary plugins
- **Hypothesis-driven perf** — Evidence gathering, ranked hypotheses, one-at-a-time testing replaces blind benchmark loops
- **Post-PR monitoring** — CI watching + iterative reviewer comment resolution
- **AST repo mapping** — Symbol index with dependency graph, cached for agent navigation

## Future

### Stop Hook Redesign
The Stop hook fires on every turn, not just session end. Redesign to fire only on explicit session end or make it opt-in via a command flag.

**When:** Next release. This is a usability blocker.

### Framework-Specific Review Checklists
React hooks rules, Django ORM patterns, Go concurrency, Rust safety — 20-50 patterns per framework loaded dynamically based on detected tech stack.

**When:** When tri-review needs to produce more actionable, framework-aware findings.

### Conditional Hook Firing
Hooks that only fire on certain git branches, when files exist, or env vars are set. Similar to skill-bus's condition system but built into devkit's hook stack.

**When:** When users need project-specific hook behavior without modifying global hooks.

### Task DAG Scheduler (Runtime)
Replace markdown-described DAGs with a runtime scheduler that topologically sorts and auto-parallelizes. Currently decompose describes the pattern; a Go/TS harness would execute it deterministically.

**When:** When workflows regularly exceed 10+ steps with complex interdependencies.

### Shared Context Injection
Automatically inject upstream agent outputs into downstream agent prompts. Currently done manually in command descriptions; could be automated.

**When:** When tri:* and decompose commands need richer inter-agent communication.

### Cost Event Hooks
Budget threshold events (warning at 80%, critical at 90%, exceeded at 100%) with auto-downgrade actions. Currently budget is guidance only.

**When:** When users need hard budget enforcement, not just guidance.

### Execution Registry
Centralized tracking of step state (pending/running/done/failed/skipped) with timing and token usage per step. Currently tracked in temp files during execution.

**When:** When workflow observability needs to improve beyond log files.

### Preset Library
Curated prompt templates for common review/improvement scenarios (Python security, Go performance, React optimization, etc.).

**When:** When the community identifies common patterns worth standardizing.
