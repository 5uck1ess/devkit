---
name: tri-dispatch
description: Three model tiers (smart + general + fast) each tackle the SAME open-ended task independently, then their divergent approaches are compared — this is the GENERAL-PURPOSE tri-* skill for tasks that don't fit the specialized siblings. Use when the user asks for tri-dispatch, triple dispatch, three-way comparison of model approaches, says "send this to three models and show me their takes", or wants to see divergent solutions before picking one. Worth the extra cost when: the task is exploratory or greenfield (design, architecture, new implementation, optimization), the user wants to compare model strengths on a novel problem, or is deliberately seeking diversity of approach before committing. Do NOT use for code review of existing code (use tri-review), debugging a specific failure (use tri-debug), or security audits (use tri-security). Do NOT use when the user already knows which approach they want — tri-dispatch is for exploration, not execution.
---

# Tri-Dispatch

Deterministic three-tier task dispatch. smart-take → general-take → fast-take (in parallel) → compare approaches.

## Invoke

Use the `devkit_start` tool with workflow: "tri-dispatch" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
