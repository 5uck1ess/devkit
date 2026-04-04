---
name: executing
description: Execute implementation plans — work through steps methodically, verify each one, keep changes small and reviewable.
---

# Executing

## The Loop

For each step in the plan:

1. **Understand** — Re-read the step. What exactly needs to change?
2. **Implement** — Make the smallest change that completes the step.
3. **Verify** — Run tests, check behavior, confirm it works.
4. **Commit** — Save the working state before moving on.

Do not skip verify. Do not batch multiple steps into one change.

## Discipline

- **One step at a time.** Finish step N before starting step N+1. The urge to "quickly also do" the next thing leads to half-done work.
- **Don't go down rabbitholes.** If you discover something unrelated that needs fixing, note it and come back later. Stay on the current step.
- **Keep changes small.** A 20-line diff is easy to review. A 200-line diff hides bugs. If a step produces a large diff, it should have been broken into smaller steps.
- **Verify at the boundary.** After completing a step, confirm the system still works end-to-end, not just the part you touched.

## When Things Go Wrong

- **Step is harder than expected** — Stop. Re-plan. Break it into sub-steps. Don't push through with a "it'll work out" mindset.
- **Step reveals a flaw in the plan** — Update the plan. Plans are living documents.
- **Tests break** — Fix them now, not later. Broken tests that you "plan to fix" become permanently broken tests.
- **You're stuck for 15+ minutes** — Step back. Re-read the requirements. Try a different approach. Ask for help.

## Checkpointing

Commit after each verified step. Commit messages should reference the plan step:

```
step 3: add input validation to parseConfig
```

This creates a clean, reviewable history and makes it easy to bisect if something breaks later.

## Done Means Done

A step is done when:
- The implementation matches the step description
- All existing tests pass
- Any new behavior has tests
- The code is clean (no TODOs, no commented-out code, no "temporary" hacks)
