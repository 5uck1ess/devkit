---
name: tri-review
description: Triple-agent code review — use when asked for a tri review, triple review, three-way review, multi-agent review, parallel review, or consensus review. Dispatches Claude/Codex/Gemini in parallel and consolidates findings. Triggers the deterministic tri-review workflow.
---

# Tri-Review

Deterministic three-tier model review. Gather → review-smart → review-general → review-fast (in parallel) → consolidate. Runs under enforce: soft so the gather step can call git diff.

## Invoke

Use the `devkit_start` tool with workflow: "tri-review" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
