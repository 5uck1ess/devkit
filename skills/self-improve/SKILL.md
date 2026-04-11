---
name: self-improve
description: Metric-gated improvement loop — use when asked to self-improve, run an improvement loop, fix issues until a metric passes, or "keep fixing until X is green". Triggers the deterministic self-improve workflow (baseline → improve → verify → loop until passing).
---

# Self-Improve

Deterministic metric-gated improvement loop. Baseline → improve → verify → loop until the gate passes → summary.

## Invoke

Use the `devkit_start` tool with workflow: "self-improve" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
