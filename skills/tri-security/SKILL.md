---
name: tri-security
description: Triple-agent security audit — use when asked for tri security, triple security audit, three-way security review, multi-agent vulnerability scan, or parallel security audit. Three-tier parallel review focused on injection, auth, and config. Triggers the deterministic tri-security workflow.
---

# Tri-Security

Deterministic three-tier security audit. Gather → audit-injection → audit-auth → audit-config (in parallel) → consolidate with severity ranking. Runs under enforce: soft so the gather step can call git diff.

## Invoke

Use the `devkit_start` tool with workflow: "tri-security" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
