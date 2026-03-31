---
name: self:improve
description: Self-recursive improvement loop — automated refactoring with test gate. Uses native improver agent in worktree isolation. Propose → measure → keep/discard → repeat.
---

# Self-Improve Loop

Autonomous, iterative improvement of a target file or directory. Each iteration: propose change → run metric → keep if better → discard if worse → repeat.

## Parameters

1. **Target** — file or directory to improve (required)
2. **Metric** — shell command that exits 0 on success (required)
3. **Objective** — what to optimize for (required)
4. **Iterations** — max cycles (default: 10)
5. **Budget** — max USD (default: $2)

## Step 1: Establish Baseline

```bash
git checkout -b self-improve/$(date +%Y%m%d-%H%M%S)
BASELINE=$({metric_command} 2>&1)
echo "$BASELINE" > /tmp/self-improve-baseline.txt
echo "$BASELINE" > /tmp/self-improve-best.txt
```

## Step 2: Run the Loop

For each iteration, spawn the `improver` agent as a native background task with worktree isolation:

```
Task: Improve {target} toward objective: {objective}
Agent: improver
Context:
  - Iteration history: (cat /tmp/self-improve-log.txt)
  - Current best metric: (cat /tmp/self-improve-best.txt)
  - Target file(s): {target}
```

The improver agent:
1. Reads the target and iteration history
2. Proposes ONE focused change
3. Applies it

Then the orchestrator:
```bash
RESULT=$({metric_command} 2>&1)
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
  echo "ITERATION $i: SUCCESS" >> /tmp/self-improve-log.txt
  echo "$RESULT" > /tmp/self-improve-best.txt
  git add -A && git commit -m "self-improve: iteration $i — passed"
else
  echo "ITERATION $i: FAILED — reverting" >> /tmp/self-improve-log.txt
  git checkout -- .
fi
```

## Step 3: Report

```
## Self-Improve Report

**Target:** {target}
**Objective:** {objective}
**Iterations:** {completed} / {total}

### Baseline → Final
{baseline} → {final}

### Log
| # | Result | Change |
|---|--------|--------|
| 1 | PASS   | Refactored validation |
| 2 | FAIL   | Caching — broke tests |
| 3 | PASS   | Extracted helper |

### Next Steps
- Review: `git diff main...HEAD`
- Merge: `git checkout main && git merge self-improve/{branch}`
- Discard: `git checkout main && git branch -D self-improve/{branch}`
```

## Presets

```
/self:improve --target src/ --metric "npm test" --objective "fix failing tests"
/self:improve --target train.py --metric "python train.py" --objective "minimize val_bpb" --iterations 50
```

## Rules

- Uses native `improver` agent with worktree isolation (token-efficient)
- Always branches first — never modifies main
- One change per iteration
- Discard on failure — `git checkout -- .`
- Never modify the metric command
- Human merges at end — never auto-merges
