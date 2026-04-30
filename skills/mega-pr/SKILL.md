---
name: mega-pr
description: >-
  Fan-out PR review that runs BOTH `/devkit:tri-review` AND
  `/pr-review-toolkit:review-pr` in parallel for maximum coverage, then
  presents unified results — this skill does not compete with its sub-skills,
  it delegates to both simultaneously. Use when the user asks for "mega PR
  review", "mega-pr", "mega review", "full PR review with everything", "both
  review tools", "maximum coverage review", or explicitly wants every
  available reviewer looking at a change at once (Claude + Codex + Gemini
  model diversity PLUS specialized aspect reviewers like
  silent-failure-hunter, type-design-analyzer, test-analyzer, code-reviewer).
  Worth the extra cost when: the PR is high-stakes and the user wants every
  angle covered, or before merging a critical or hard-to-revert change. Do
  NOT use when the user only wants one review system (use tri-review or
  pr-review-toolkit:review-pr directly). Do NOT use for routine code review
  where a single reviewer suffices. This is deliberate overkill for when you
  want absolutely everything.
---

# Mega PR Review

Run `/devkit:tri-review` and `/pr-review-toolkit:review-pr` simultaneously for maximum review coverage when those skills/plugins are available. Under Codex, run devkit's tri-review MCP workflow and any available external review command; if `pr-review-toolkit` is not installed in the host, report that part as unavailable instead of fabricating results.

## Step 1: Launch Both Reviews

**[PARALLEL]** — invoke both in a single message using the Skill tool:

1. `Skill: devkit:tri-review` — dispatches available model-diverse reviewers
2. `Skill: pr-review-toolkit:review-pr` — dispatches specialized aspect reviewers when the Claude plugin is installed (silent-failure-hunter, type-design-analyzer, test-analyzer, code-reviewer, etc.)

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
