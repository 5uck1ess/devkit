---
name: pr-ready
description: Full PR pipeline — use when asked to submit a PR, create a pull request, ship this, open a PR, or "make this PR-ready". Runs lint, tests, security scan, generates changelog, creates the PR, and monitors CI + reviewer comments until merge-ready.
---

# PR Ready

Deterministic PR pipeline: validate → necessity → lint (loop) → test (loop) → security → changelog → doc-check → create PR → monitor (loop).

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
6. **changelog** — generates entry from git diff
7. **doc-check** — classifies the diff (feature/bugfix/breaking/internal/docs-only) and decides per-file whether README, ROADMAP, CLAUDE.md, docs/, plugin.json, or workflows/ need updates. Applies mechanical edits directly (moving roadmap bullets, adding command rows); flags ambiguous updates as `[!]` in the output checklist. Commits applied edits with `docs: update ...` so they land in the PR alongside the code. CHANGELOG.md is intentionally skipped — it is managed by the release pipeline.
8. **create-pr** — pushes branch, creates PR via gh pr create with title/summary/changelog/test plan
9. **monitor** — waits for CI, classifies reviewer comments (code_fix/style_nit/question/false_positive/out_of_scope), applies fixes, replies, pushes, loops until all resolved

## Rules

- Never force-push
- Never dismiss reviews — only re-request after fixing
- Reply to false positives with evidence, not dismissal
- Escalate architectural changes via AskUserQuestion
- Stop after 10 monitor iterations or when stuck (3 iters with zero progress)
