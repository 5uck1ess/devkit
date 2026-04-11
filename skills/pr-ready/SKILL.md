---
name: pr-ready
description: Full end-to-end PR pipeline — validate → necessity-check → lint (loop) → test (loop) → security scan → doc-check → changelog → create-pr → monitor reviews (loop). Takes a branch from "code done" to "merged" with every gate enforced. Use when the user asks to "submit a PR", "open a pull request", "ship this", "make this PR-ready", "finalize this branch", or wants the full pipeline run end-to-end with CI monitoring and reviewer-comment handling automated. Worth the ceremony when: lint and test gates must pass before the PR opens, docs need syncing alongside code (README, ROADMAP, SKILL.md, workflows), security scan is required, or the user wants reviewer comments automatically classified and responded to. Do NOT use for a quick commit+push+PR without gates — use `/commit-commands:commit-push-pr` for that lighter path. Do NOT use when already mid-PR and just handling existing review comments in isolation. Do NOT use on the main branch or with uncommitted changes — the validate step will block you. This is devkit's heaviest PR workflow; pick it when you want everything run.
---

# PR Ready

Deterministic PR pipeline: validate → necessity → lint (loop) → test (loop) → security → doc-check → changelog → create PR → monitor (loop).

## Invoke

Start the workflow via the devkit engine:

Use the `devkit_start` tool with workflow: "pr-ready" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.

## What it does

1. **validate** — checks not on main, no uncommitted changes, commits ahead of main
2. **necessity** — removes debug prints, unrelated changes, stray files from the diff
3. **lint** — runs linter, fixes violations, loops until clean
4. **test** — runs test suite, fixes failures, loops until passing
5. **security** — scans for hardcoded secrets, injection, XSS, traversal, insecure deps
6. **doc-check** — classifies the diff (feature/bugfix/breaking/internal/docs-only) and decides per-file whether README, ROADMAP, CLAUDE.md, docs/, `skills/*/SKILL.md`, plugin.json, or `workflows/*.yml` need updates. Applies mechanical edits directly (moving roadmap bullets, adding command rows, syncing SKILL.md step lists); flags ambiguous updates as `[!]` in the output checklist. Commits applied edits with `docs: update ...` so they land in the PR alongside the code. CHANGELOG.md is intentionally skipped — it is managed by the release pipeline. Runs **before** `changelog` so any doc commit it creates is captured in the PR description.
7. **changelog** — generates entry from git diff (now includes any doc-check commits)
8. **create-pr** — pushes branch, creates PR via gh pr create with title/summary/changelog/test plan
9. **monitor** — waits for CI, classifies reviewer comments (code_fix/style_nit/question/false_positive/out_of_scope), applies fixes, replies, pushes, loops until all resolved

## Rules

- Never force-push
- Never dismiss reviews — only re-request after fixing
- Reply to false positives with evidence, not dismissal
- Escalate architectural changes via AskUserQuestion
- Stop after 10 monitor iterations or when stuck (3 iters with zero progress)
