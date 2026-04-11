---
name: tri-debug
description: Three independent root-cause analyses of a hard bug or failure — smart + general + fast model tiers diagnose in parallel, then theories are compared and reconciled. Use when the user asks for tri-debug, triple debug, three-way debugging, multi-model diagnosis, parallel debug, consensus root-cause analysis, or is stuck on a hard bug and wants divergent hypotheses. Worth the extra cost when — the bug is hard (heisenbug, intermittent, cross-system), logs are ambiguous, the obvious fixes have already failed, or the user explicitly wants three independent theories before committing to one. Do NOT use for code review (use tri-review), security audits (use tri-security), or comparing greenfield implementation approaches (use tri-dispatch). Do NOT use for simple "what does this error mean" or routine null-check / typo bugs — a single pass is faster.
---

# Tri-Debug

Deterministic three-model diagnosis. smart-diagnosis → general-diagnosis → fast-diagnosis (in parallel) → compare theories → consensus.

## Invoke

Use the `devkit_start` tool with workflow: "tri-debug" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
