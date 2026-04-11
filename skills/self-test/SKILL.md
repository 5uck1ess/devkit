---
name: self-test
description: Run tests and fix failures until green — use when asked to self-test, fix tests, "make tests pass", "fix the failing tests", or "run tests and fix everything". Triggers the deterministic self-test workflow (baseline → fix → verify → loop until passing).
---

# Self-Test

Deterministic test-fix loop. Baseline test run → fix failures → verify → loop until all pass → summary.

## Invoke

Use the `devkit_start` tool with workflow: "self-test" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
