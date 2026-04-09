---
description: Profile performance, optimize hot paths deterministically, verify improvement.
---

# Self-Perf

Deterministic performance optimization: benchmark → optimize → gate check → repeat until benchmark exits 0.

## Invoke

```
devkit workflow run self-perf "{benchmark_command}"
```

If `devkit workflow` is not available, follow this manually:

1. **Baseline** — Run the benchmark command and capture metrics
2. **Optimize loop** — Identify bottleneck, make one targeted optimization, re-benchmark; stop when target met (max 5 iterations)
3. **Verify** — Run benchmark one final time
4. **Summary** — Report improvement (absolute and percentage)

## Rules

- The benchmark command must exit non-zero when the target is not met (wrap bare benchmarks in a threshold-checking script)
- One optimization at a time — no speculative refactoring
- Only change what impacts the metric
