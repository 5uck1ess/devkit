---
name: self-audit
description: Self-audit the codebase — use when asked to self-audit, audit the codebase against its own metrics, run a karpathy-style audit, measure and rank issues by evidence, or "what's wrong with this repo". Triggers the deterministic self-audit workflow (detect → measure quality/security/git → analyze → synthesize).
---

# Self-Audit

Deterministic codebase audit. Detect ecosystem → measure-quality → measure-security → measure-git (in parallel) → analyze → synthesize actionable plan.

## Invoke

Use the `devkit_start` tool with workflow: "self-audit" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
