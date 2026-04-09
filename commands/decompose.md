---
description: Decompose a high-level goal into a task DAG — break down, assign to agents, resolve dependencies, execute in order.
---

# Goal Decomposition

Break a high-level goal into an executable task graph: clarify → decompose → resolve order → execute → report.

## Invoke

```
devkit workflow run decompose "{goal_description}"
```

If `devkit workflow` is not available, follow this manually:

1. **Clarify the goal** — Restate precisely. Ask user for clarification if ambiguous.
2. **Decompose into tasks** — Break into 3-10 concrete, testable tasks. Each must have clear done criteria. Identify parallelizable groups.
3. **Resolve execution order** — Build dependency graph. Tasks with no dependencies can run in parallel.
4. **Execute** — Run tasks in dependency order, dispatching independent tasks to agents in parallel where possible.
5. **Report** — Summary of completed tasks, any failures, and remaining work.

## Rules

- Each task must be independently testable
- No circular dependencies
- Prefer parallel execution where dependencies allow
- Stop and report if a blocking task fails
