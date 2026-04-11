---
name: refactor
description: Refactor code safely — use when asked to refactor, restructure, clean up, reorganize, extract, rename, or modernize a piece of code. Triggers the deterministic refactor workflow (analyze → plan → restructure → verify nothing broke).
---

# Refactor

Deterministic refactor workflow. Analyze → plan → refactor → run-tests → fix-tests → comparison. Tests are the safety net — the workflow will not exit until they pass.

## Invoke

Use the `devkit_start` tool with workflow: "refactor" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
