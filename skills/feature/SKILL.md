---
name: feature
description: Build a new feature end-to-end — use when asked to add a feature, build X, implement Y, ship a new capability, or "new feature ...". NOT for /feature-dev:feature-dev which is a separate plugin. Triggers the deterministic feature workflow (triage → brainstorm → plan → implement → test → lint → review).
---

# Feature

Deterministic full-lifecycle feature workflow. Triage → brainstorm → plan → implement → gen-tests → run-tests → lint → review → final-report. Includes a fast path for trivial changes.

## Invoke

Use the `devkit_start` tool with workflow: "feature" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
