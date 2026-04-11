---
name: tri-review
description: Three independent code reviews (smart + general + fast model tiers running Claude, Codex, and Gemini) with consolidated findings — use when the user asks for tri-review, triple code review, three-way code review, multi-model code review, consensus code review, or says "get three opinions on this diff/PR/change before I ship". Worth the extra cost when the change is high-stakes, the user doesn't trust a single-model review, or they want breadth of perspectives over depth. Do NOT use for bug diagnosis (use tri-debug), security audits (use tri-security), or comparing implementation approaches on greenfield work (use tri-dispatch). Do NOT use for routine "can you review this PR" requests where a single-model pass is fine.
---

# Tri-Review

Deterministic three-tier model review. Gather → review-smart → review-general → review-fast (in parallel) → consolidate. Runs under enforce: soft so the gather step can call git diff.

## Invoke

Use the `devkit_start` tool with workflow: "tri-review" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
