---
name: harness-audit
description: Audit the agent harness itself — CLAUDE.md, hooks, MCP configs, agent definitions, and permissions — for misconfigurations, injection risks, and security gaps. Use when the user asks to "audit my setup", "check my harness config", "is my agent config secure", "review my hooks", "check my MCP setup", or before trusting a new project's harness with autonomous work. Worth using on first clone of an unfamiliar repo, after adding new hooks or MCP servers, or when onboarding a new team member. Do NOT use for auditing project source code (use audit or tri-security), debugging workflow failures (use tri-debug or bugfix), or reviewing code changes (use tri-review).
---

# Harness Audit

Audit agent harness configuration for security and misconfiguration issues.

## Invoke

Use the `devkit_start` tool with workflow: "harness-audit" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
