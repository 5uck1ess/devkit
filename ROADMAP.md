# Devkit Roadmap

## Implemented

- **21 commands** — Solo workflows, self-improvement loops, multi-agent dispatch
- **11 skills** — Coding methodology guides including skill/workflow authoring, stuck recovery, verification
- **6 agents** — Scoped tool access, worktree isolation, model assignment
- **Safety hooks** — PreToolUse hook blocking destructive operations, prompting for risky ones
- **Graceful degradation** — tri:* commands work with 1-3 agents depending on installed CLIs
- **Goal decomposition** — Task DAG with dependency ordering and parallel execution
- **Concurrency limits** — Max 3 parallel agents in multi-agent commands
- **Early-exit conditions** — Self-improvement loops stop when goal is met, not just at max iterations
- **Token budget guidance** — Per-command budget recommendations with model downgrade patterns
- **RTK token optimization** — Optional PreToolUse hook compresses Bash output via RTK (60-90% savings)
- **12 YAML workflows** — Portable workflow definitions (feature, bugfix, refactor, research, self-*, tri-*)
- **Separate marketplace** — Multi-plugin marketplace at `5uck1ess/marketplace`

## Future

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

### Go CLI Harness
Deterministic loop control, process management, and unattended runs. See `src/TODO.md`.

**When:** When markdown-described workflows need tighter execution guarantees.
