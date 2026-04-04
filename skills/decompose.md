---
name: devkit:decompose
description: Decompose a high-level goal into a task DAG — break down, assign to agents, resolve dependencies, execute in order.
---

# Goal Decomposition

Break a high-level goal into a dependency-ordered task graph, assign tasks to available agents, and execute.

## Step 1: Clarify the Goal

Use `AskUserQuestion` to clarify if the goal is vague:
- What is the desired end state?
- What files or systems are involved?
- Are there constraints (language, framework, time)?

## Step 2: Decompose into Tasks

Break the goal into discrete tasks. For each task:

```
| # | Task | Agent | Depends On | Est. Effort |
|---|------|-------|------------|-------------|
| 1 | Research existing patterns | researcher | — | Low |
| 2 | Design the approach | — (orchestrator) | 1 | Low |
| 3 | Implement core logic | improver | 2 | High |
| 4 | Write tests | test-writer | 3 | Medium |
| 5 | Security review | security-auditor | 3 | Medium |
| 6 | Final review | reviewer | 4, 5 | Low |
```

### Rules for Decomposition

- Each task should be completable by a single agent in one pass
- Tasks with no dependencies can run in parallel
- Tasks must declare all dependencies explicitly
- Prefer small, focused tasks over large monolithic ones
- Include verification tasks (tests, review) — don't skip them

## Step 3: Resolve Execution Order

**[PARALLEL]** Topologically sort the DAG. Tasks at the same depth with no mutual dependencies run concurrently.

```
Depth 0: [Task 1]              — no dependencies
Depth 1: [Task 2]              — depends on 1
Depth 2: [Task 3]              — depends on 2
Depth 3: [Task 4, Task 5]      — both depend on 3, run in parallel
Depth 4: [Task 6]              — depends on 4 and 5
```

**Concurrency limit:** Max 3 parallel agents to avoid API rate limits.

## Step 4: Execute

For each depth level:

1. Dispatch all tasks at this depth **[PARALLEL]** (up to concurrency limit)
2. Wait for all to complete
3. If any task fails:
   - **Cascade skip** all tasks that depend on the failed task
   - Continue executing independent tasks at deeper levels
   - Report the failure and skipped tasks
4. Inject upstream task outputs as context into downstream task prompts

### Agent Dispatch

```
Task: {task description}
Agent: {assigned agent}
Context:
  - Goal: {original goal}
  - Upstream results: {outputs from dependency tasks}
  - Constraints: {any user-specified constraints}
```

## Step 5: Report

```
## Decomposition Report: {goal}

### Task Graph
{ASCII or table representation of the DAG}

### Execution Summary
| # | Task | Agent | Status | Duration |
|---|------|-------|--------|----------|
| 1 | Research | researcher | Done | — |
| 2 | Design | orchestrator | Done | — |
| 3 | Implement | improver | Done | — |
| 4 | Tests | test-writer | Done | — |
| 5 | Security | security-auditor | Done | — |
| 6 | Review | reviewer | Done | — |

### Results
{consolidated output from all tasks}

### Failed / Skipped
{any tasks that failed or were cascade-skipped}
```

## Rules

- Clarify before decomposing — don't guess at vague goals
- Every task needs a clear definition of done
- Cascade failures — if a task fails, skip all dependents
- Max 3 concurrent agents (rate limit protection)
- Inject upstream context into downstream tasks
- Report which tasks ran, which were skipped, and why
