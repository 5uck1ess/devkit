---
name: self-audit
description: Evidence-ranked codebase self-assessment (karpathy-style) — measures quality, security, and git signals in parallel then produces a prioritized list of what's actually wrong here, ranked by how much evidence supports each finding, with an actionable improvement plan. Use when the user asks to "self-audit", "audit the codebase against its own metrics", "run a karpathy-style audit", "what's wrong with this repo and what should I fix first", "rank issues by evidence", or wants a synthesized improvement plan rather than a scanner dump. Worth using when the user wants judgment and prioritization (where should I invest effort?), not raw warnings; when they're choosing the next big refactor; or when they want the biggest issues surfaced first with evidence showing why. Do NOT use for a plain scanner rundown of deps/lint/CVEs (use audit — that just runs the tools), security review of a specific code change (use tri-security), or debugging a specific failure (use tri-debug or bugfix). This skill is about ranking and judgment, not tool execution.
---

# Self-Audit

Deterministic codebase audit. Detect ecosystem → measure-quality → measure-security → measure-git (in parallel) → analyze → synthesize actionable plan.

## Invoke

Use the `devkit_start` tool with workflow: "self-audit" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
