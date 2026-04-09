---
name: mega-pr
description: Run both tri-review and pr-review-toolkit review-pr in parallel for maximum coverage — use when the user asks for mega PR review, mega-pr, mega review, full PR review, or wants both tri-review and pr-review-toolkit at once.
---

# Mega PR Review

Run `/devkit:tri-review` and `/pr-review-toolkit:review-pr` simultaneously for maximum review coverage across all available agents.

## Step 1: Launch Both Reviews in Parallel

Invoke both skills at the same time using the Skill tool:

**[PARALLEL]** — both calls in a single message:

1. `Skill: devkit:tri-review` — dispatches Claude + Codex + Gemini reviewers
2. `Skill: pr-review-toolkit:review-pr` — dispatches specialized aspect reviewers (silent-failure-hunter, type-design-analyzer, test-analyzer, code-reviewer, etc.)

Pass through any user-provided arguments (custom prompt, specific files) to both skills.

## Step 2: Present Combined Results

After both complete, present a single unified report:

```
## Mega PR Review: {branch_name}

### tri-review Results
{output from tri-review — consensus + unique findings}

### pr-review-toolkit Results
{output from pr-review-toolkit — categorized by severity}

### Summary
- Total agents participated: {count}
- Critical issues: {count}
- Warnings: {count}
- Suggestions: {count}
```

## Rules

- Always run both in parallel — never sequential
- If one skill fails, still present the other's results
- Do not deduplicate across the two — let the user see both perspectives
- Pass user arguments to both skills unchanged
