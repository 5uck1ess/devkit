---
description: Metric-gated improvement loop — run command, fix issues, repeat until passing.
---

# Self-Improve

Deterministic improvement loop: run metric → fix issues → gate check → repeat until exit code 0.

## Invoke

```
devkit workflow run self-improve "{metric_command}"
```

If `devkit workflow` is not available, follow this manually:

1. **Baseline** — Run the metric command and capture output
2. **Improve loop** — Analyze failures, make targeted fixes, re-run metric; stop when passing (max 10 iterations)
3. **Verify** — Run metric one final time
4. **Summary** — Report what was fixed, iteration count, final metric status

## Rules

- Only change what's needed to improve the metric
- Don't refactor unrelated code
- One group of related fixes per iteration
- Discard on regression
