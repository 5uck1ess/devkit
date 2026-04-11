---
name: tri-dispatch
description: Triple-tier task dispatch — use when asked for tri dispatch, triple dispatch, three-way comparison, "send this to three models", or "compare model approaches". Sends the same task to three model tiers in parallel and compares their approaches. Triggers the deterministic tri-dispatch workflow.
---

# Tri-Dispatch

Deterministic three-tier task dispatch. smart-take → general-take → fast-take (in parallel) → compare approaches.

## Invoke

Use the `devkit_start` tool with workflow: "tri-dispatch" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
