---
name: mega-pr
description: Run tri-review and pr-review-toolkit review-pr in parallel for maximum coverage — use when asked for mega PR review, mega-pr, mega review, full PR review, or both review tools at once.
---

# Mega PR Review

Run `/devkit:tri-review` and `/pr-review-toolkit:review-pr` simultaneously for maximum review coverage.

## Step 1: Launch Both Reviews

**[PARALLEL]** — invoke both in a single message using the Skill tool:

1. `Skill: devkit:tri-review` — dispatches Claude + Codex + Gemini reviewers
2. `Skill: pr-review-toolkit:review-pr` — dispatches specialized aspect reviewers (silent-failure-hunter, type-design-analyzer, test-analyzer, code-reviewer, etc.)

Pass any user-provided arguments (custom prompt, specific files) to both skills unchanged.

## Step 2: Present Combined Results

After both complete, present a unified report:

```
## Mega PR Review: <branch_name>

### tri-review Results
<consensus + unique findings>

### pr-review-toolkit Results
<categorized by severity>

### Summary
- Total agents participated: <count>
- Critical issues: <count>
- Warnings: <count>
- Suggestions: <count>
```

## Rules

- Always run both in parallel — never sequential
- If one skill fails, still present the other's results
- Do not deduplicate — let the user see both perspectives
