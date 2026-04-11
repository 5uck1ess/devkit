---
name: refactor
description: Deterministic refactor workflow with tests as the safety net — analyze → plan → refactor → run-tests → fix-tests → comparison. The workflow will not exit until tests pass, guaranteeing behavior preservation. Use when the user asks to "refactor", "restructure", "reorganize", "clean up", "extract", "rename across files", or "modernize" existing code that has test coverage. Worth the ceremony when — the refactor spans multiple files, behavior must be preserved exactly, test coverage exists to prove nothing broke, or the user explicitly wants a safety-netted transformation. Do NOT use for renaming a single variable or function in one file (just do it — the workflow overhead isn't worth it). Do NOT use for adding new behavior (use feature) or fixing broken behavior (use bugfix). Do NOT use when there are no tests — the safety net is the entire point; without tests, do the refactor manually with careful review. If the user wants refactoring advice or a discussion (not the transformation), answer directly instead of dispatching the workflow.
---

# Refactor

Deterministic refactor workflow. Analyze → plan → refactor → run-tests → fix-tests → comparison. Tests are the safety net — the workflow will not exit until they pass.

## Invoke

Use the `devkit_start` tool with workflow: "refactor" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
