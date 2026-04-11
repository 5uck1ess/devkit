---
name: self-migrate
description: Incremental migration loop — use when asked to migrate, port, upgrade, convert, or modernize a codebase incrementally with tests as a safety gate. Triggers the deterministic self-migrate workflow (baseline → migrate one piece → verify → loop).
---

# Self-Migrate

Deterministic incremental migration loop. Baseline → migrate one piece → verify with tests → loop until done → summary. Tests are the safety gate at every step.

## Invoke

Use the `devkit_start` tool with workflow: "self-migrate" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
