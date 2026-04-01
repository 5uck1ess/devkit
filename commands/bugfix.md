---
name: devkit:bugfix
description: Full lifecycle bug fix — reproduce, diagnose, fix, regression test, verify.
---

# Bug Fix Workflow

Complete bug fix lifecycle: reproduce → diagnose root cause → fix → regression test → verify.

## Step 1: Reproduce

```
Bug report: {user's input}

Reproduce the issue:
1. Find the relevant code
2. Understand the expected vs actual behavior
3. Identify a minimal reproduction path
4. Confirm the bug exists (run the failing case if possible)
```

## Step 2: Diagnose

```
Trace the root cause using the reproduction from Step 1.
Read the code path, check assumptions, examine edge cases.
Determine exactly WHY this happens, not just WHERE.

Propose a specific fix with reasoning.
```

## Step 3: Fix

```
Implement the fix based on the diagnosis.
Keep changes minimal — only touch what's needed to resolve the root cause.
Don't refactor surrounding code.
```

## Step 4: Regression Test

```
Write a regression test that:
1. Would have caught this bug before the fix
2. Verifies the fix works
3. Covers the edge case that triggered the bug
```

## Step 5: Run Tests

```bash
# Run the full test suite including the new regression test
# Auto-detect test command from project config
```

If tests fail, fix them. Determine if the test or the code is wrong. Loop up to 5 times until all pass.

## Step 6: Summary

```
## Bug
What was reported and how it was reproduced.

## Root Cause
What was actually wrong and why.

## Fix
What was changed and why.

## Regression Test
What test was added to prevent recurrence.

## Status
Test suite status. Ready to commit or remaining concerns.
```

## Rules

- Reproduce before diagnosing — confirm the bug exists first
- Root cause, not symptoms — understand WHY, not just WHERE
- Minimal fix — don't refactor, don't clean up, just fix the bug
- Regression test required — every fix must include a test that would have caught it
- All tests must pass before reporting done
- Loop on test failures up to 5 times
