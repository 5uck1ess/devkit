---
name: self-perf
description: Profile and optimize performance — use when asked to self-perf, optimize performance, profile hot paths, speed this up, or "make it faster". Triggers the deterministic self-perf workflow (baseline → profile → optimize → verify improvement).
---

# Self-Perf

Deterministic performance optimization loop. Baseline → optimize hot paths → verify improvement → summary.

## Invoke

Use the `devkit_start` tool with workflow: "self-perf" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
