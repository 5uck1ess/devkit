---
description: Full lifecycle bug fix — reproduce, diagnose, fix, regression test, verify.
---

# Bug Fix

Deterministic bug fix lifecycle: triage → reproduce → diagnose → fix → regression test → verify.

## Invoke

```
devkit workflow run bugfix "{bug_description}"
```

If `devkit workflow` is not available, follow this manually:

1. **Triage** — Classify as TRIVIAL / NORMAL / COMPLEX. Trivial bugs skip to quick-fix path.
2. **Reproduce** — Read relevant code, understand expected vs actual, identify minimal reproduction path
3. **Diagnose** — Trace root cause (WHY, not just WHERE); propose specific fix with reasoning
4. **Fix** — Implement minimal fix; don't refactor surrounding code
5. **Regression test** — Write a test that would have caught this bug; verify it fails without the fix and passes with it
6. **Run tests** — Run full test suite including new regression test
7. **Fix test failures** — If tests fail, determine if bug is in test or code; fix and re-run (loop max 5)
8. **Summary** — Report bug, root cause, fix, regression test, and test suite status

## Rules

- Reproduce before diagnosing — don't guess at root causes
- Minimal changes only — fix the bug, don't refactor
- Always write a regression test
- Tests must pass before declaring fixed
- Use scratchpad (`.devkit/scratchpads/current.md`) to track attempts across iterations
