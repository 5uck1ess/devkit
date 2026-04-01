---
name: self:lint
description: Self-improvement loop targeting lint and type errors. Iteratively fixes issues until zero remain or iterations exhausted.
---

# Self-Improve: Lint / Type Errors

Automated loop that runs your linter or type checker, fixes issues one at a time, and verifies no regressions.

## Parameters

1. **Lint command** — command that reports lint/type errors (required)
2. **Target** — file or directory scope (default: entire project)
3. **Iterations** — max cycles (default: 20)
4. **Budget** — max USD (default: $2)

## Budget & Early Exit

- **Token budget:** ~200k tokens. Lint fixes are usually cheap.
- **Early exit:** Stop immediately when error count reaches 0.
- **Stuck detection:** If 3 consecutive iterations show no improvement (error count doesn't decrease), stop and report. See the `stuck` skill.

## Step 1: Establish Baseline

```bash
git checkout -b self-lint/$(date +%Y%m%d-%H%M%S)
BASELINE=$({lint_command} 2>&1)
ERRORS=$(echo "$BASELINE" | grep -cE '(error|warning)')
echo "Baseline: $ERRORS issues" > /tmp/self-lint-log.txt
echo "$BASELINE" > /tmp/self-lint-baseline.txt
```

## Step 2: Run the Loop

For each iteration, spawn the `improver` agent:

```
Task: Fix lint/type errors in {target}.
Agent: improver
Context:
  - Iteration: {i} of {max}
  - Remaining errors: {error_count}
  - Current lint output: {lint_output}
  - Iteration history: (cat /tmp/self-lint-log.txt)
```

The improver agent:
1. Reads the lint output
2. Picks the highest-priority error
3. Fixes ONE issue (or a group of related issues in one file)

Then the orchestrator:
```bash
RESULT=$({lint_command} 2>&1)
NEW_ERRORS=$(echo "$RESULT" | grep -cE '(error|warning)')

if [ $NEW_ERRORS -lt $PREV_ERRORS ]; then
  echo "ITERATION $i: PASS — $PREV_ERRORS → $NEW_ERRORS issues" >> /tmp/self-lint-log.txt
  git add -A && git commit -m "self-lint: iteration $i — $NEW_ERRORS remaining"
  PREV_ERRORS=$NEW_ERRORS
else
  echo "ITERATION $i: FAIL — no improvement or regression, reverting" >> /tmp/self-lint-log.txt
  git checkout -- .
fi
```

Stop early if zero errors remain.

## Step 3: Report

```
## Self-Lint Report

**Lint command:** {lint_command}
**Errors:** {baseline_count} → {final_count}
**Iterations:** {completed} / {total}

### Log
| # | Result | Errors | Fix |
|---|--------|--------|-----|
| 1 | PASS   | 12→10  | Fixed unused imports in api.ts |
| 2 | PASS   | 10→8   | Added missing return types |
| 3 | FAIL   | 8→8    | Reverted — no improvement |

### Next Steps
- Review: `git diff main...HEAD`
- Merge: `git checkout main && git merge self-lint/{branch}`
```

## Rules

- Uses `improver` agent with worktree isolation
- Always branches first
- Error count must strictly decrease to keep a change
- Discard on regression or no improvement
- Stop early at zero errors
- Never disable lint rules — fix the underlying issue
