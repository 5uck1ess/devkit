---
name: self-lint
description: Run linter and fix violations until clean — use when asked to self-lint, lint and fix, fix lint errors, "make lint pass", or "fix all the lint issues". Triggers the deterministic self-lint workflow (baseline → fix → verify → loop until clean).
---

# Self-Lint

Deterministic lint-and-fix loop. Baseline lint → fix violations → verify → loop until clean → summary.

## Invoke

Use the `devkit_start` tool with workflow: "self-lint" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
