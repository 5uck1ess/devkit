---
name: tri-security
description: Three-tier parallel security audit (injection + auth + config, each examined by a different model tier) with severity-ranked consolidation — use when the user asks for tri-security, triple security audit, three-way security review, multi-model vulnerability scan, or parallel security audit of a code change, branch, module, or endpoint. Worth the extra cost when: touching auth/payments/user-uploads/admin surfaces, before prod deploy of security-sensitive code, after a security incident, or when the user is paranoid about a specific attack class. Do NOT use for general code quality review (use tri-review), bug diagnosis (use tri-debug), or approach comparison on greenfield work (use tri-dispatch). Do NOT use for a single-line "is this XSS safe" question, or for running pre-existing scanners like npm audit / gosec / semgrep — those are single-tool tasks, not a tri workflow.
---

# Tri-Security

Deterministic three-tier security audit. Gather → audit-injection → audit-auth → audit-config (in parallel) → consolidate with severity ranking. Runs under enforce: soft so the gather step can call git diff.

## Invoke

Use the `devkit_start` tool with workflow: "tri-security" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
