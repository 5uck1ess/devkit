---
name: bugfix
description: Deterministic full-lifecycle bugfix workflow — triage → reproduce → diagnose → fix → regression-test → run-tests, with a reproduction gate that must fail before the fix and pass after. Use when the user asks to "fix a bug", "debug this issue", "resolve this error", "patch a defect", "this is broken, fix it", or reports a specific failure they want systematically fixed with a regression test added. Worth the ceremony when — the bug is non-trivial, reproduction isn't obvious, a regression test would prevent recurrence, or the fix needs the full test suite to verify nothing else broke. Do NOT use for one-line typos, missing imports, or obvious null-checks (just fix them directly — the workflow has a fast path but the lookup is still overhead). Do NOT use for hard bugs where the user wants divergent hypotheses before committing to one (use tri-debug). Do NOT use for "something's slow" performance work (use self-perf) or for fixing failing tests specifically (use self-test).
---

# Bug Fix

Deterministic full-lifecycle bugfix workflow. Triage → reproduce → diagnose → fix → regression-test → run-tests → fix-tests → summary. Includes a fast path for trivial fixes.

## Invoke

Use the `devkit_start` tool with workflow: "bugfix" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
