---
name: audit
description: External-scanner-driven project health check — runs the actual ecosystem tools (npm audit, eslint, govulncheck, semgrep, pip-audit, etc.) to surface what the standard scanners flag: outdated or vulnerable dependencies, lint violations, known CVEs, and config drift. Use when the user asks to "audit this project", "check project health", "run a project audit", "check deps/lint/security state", "is this repo up to date", or wants a stateless rundown of what the standard tools report. Worth using when kicking off work on an unfamiliar project, before a release, after a dependency bump, or when triaging a long-neglected repo. Do NOT use for introspective "what's wrong with this repo and what should I fix first" questions (use self-audit — that ranks issues by evidence), security-focused review of a specific code change (use tri-security), or debugging specific failures (use tri-debug or bugfix). This skill reports what scanners say, not what a human should prioritize.
---

# Project Audit

Deterministic project health audit. Detect ecosystem → audit deps → lint → security → consolidated report.

## Invoke

Use the `devkit_start` tool with workflow: "audit" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
