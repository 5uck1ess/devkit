---
name: bugfix
description: Fix a bug end-to-end — use when asked to fix a bug, debug an issue, resolve an error, patch a defect, or "this is broken, fix it". Triggers the deterministic bugfix workflow (triage → reproduce → diagnose → fix → regression test → run tests).
---

# Bug Fix

Deterministic full-lifecycle bugfix workflow. Triage → reproduce → diagnose → fix → regression-test → run-tests → fix-tests → summary. Includes a fast path for trivial fixes.

## Invoke

Use the `devkit_start` tool with workflow: "bugfix" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
