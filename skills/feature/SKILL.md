---
name: feature
description: Deterministic full-lifecycle feature workflow (engine-driven, gated, looped) — triage → brainstorm → plan → implement → gen-tests → run-tests → lint → review → final-report. Use when the user asks to "add a feature", "build X", "implement Y", "ship a new capability", "new feature X", or wants an end-to-end pass from rough spec to PR-ready implementation with lint and test gates enforced. Worth the ceremony when — the feature is non-trivial (touches multiple files, needs tests, needs a plan), the user wants a structured walk from idea to working code, or wants gated iteration instead of free-form building. Do NOT use for the separate `/feature-dev:feature-dev` plugin — that is an agent-guided collaborative workflow; if the user explicitly invokes that command, respect it. Do NOT use for one-file quick additions (just write the code), bug fixes (use bugfix), or pure refactoring of existing behavior (use refactor). Includes a fast path for trivial changes so the workflow self-adjusts to scale.
---

# Feature

Deterministic full-lifecycle feature workflow. Triage → brainstorm → plan → implement → gen-tests → run-tests → lint → review → final-report. Includes a fast path for trivial changes.

## Invoke

Use the `devkit_start` tool with workflow: "feature" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
