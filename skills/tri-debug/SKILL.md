---
name: tri-debug
description: Triple-agent debugging — use when asked for tri debug, triple debug, three-way debugging, multi-agent diagnosis, parallel debug, or consensus root-cause analysis. Three model tiers diagnose the bug independently and compare theories. Triggers the deterministic tri-debug workflow.
---

# Tri-Debug

Deterministic three-model diagnosis. smart-diagnosis → general-diagnosis → fast-diagnosis (in parallel) → compare theories → consensus.

## Invoke

Use the `devkit_start` tool with workflow: "tri-debug" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
