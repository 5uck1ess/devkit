---
description: Autonomous improvement loop — audit, fix, measure, keep or revert, repeat.
---

# Autoloop

Autonomous iterative improvement: gather inputs → baseline → audit → fix → measure → compare → keep/revert → repeat.

## Invoke

```
devkit workflow run autoloop "{metric_command}"
```

If `devkit workflow` is not available, follow this manually:

1. **Gather inputs** — Identify metric command, objective, guard command (optional), and iteration count
2. **Baseline** — Run metric command, capture starting state
3. **Audit** — Analyze codebase for improvement opportunities
4. **Fix** — Make one targeted improvement
5. **Measure** — Re-run metric command
6. **Compare** — Did the metric improve? Run guard command if set.
7. **Keep or revert** — Keep if improved and guard passes; revert otherwise. If 3+ consecutive failures, escalate.
8. **Report** — Summary of iterations, improvements kept, and final state

## Rules

- One change per iteration — don't bundle
- Always measure before and after
- Revert on regression or guard failure
- Stop after 3 consecutive failures and report
- Guard command (if set) must pass to keep a change
