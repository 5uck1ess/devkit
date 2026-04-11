---
name: audit
description: Audit a project for health — use when asked to audit this project, check project health, run a project audit, look for issues, or assess code/dep/lint/security state. Triggers the deterministic audit workflow (detect ecosystem → deps → lint → security → report).
---

# Project Audit

Deterministic project health audit. Detect ecosystem → audit deps → lint → security → consolidated report.

## Invoke

Use the `devkit_start` tool with workflow: "audit" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
