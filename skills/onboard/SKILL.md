---
name: onboard
description: Generate a codebase onboarding guide — use when asked to explain this codebase, help understand the architecture, give a tour of the repo, or onboard a new contributor. Triggers the deterministic onboard workflow (analyze structure → architect via researcher agent → write guide).
---

# Codebase Onboarding

Deterministic onboarding guide generation. Analyze structure → architect via the researcher subagent → write guide to `docs/ONBOARDING.md`.

## Invoke

Use the `devkit_start` tool with workflow: "onboard" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
