---
name: devkit:verify
description: Output validation checklist — verify work meets criteria before proceeding to the next step or reporting completion.
---

# Verification

Before marking any step complete, run through this checklist.

## Code Changes

- [ ] The change does what was asked — not more, not less
- [ ] Tests pass (`exit 0` from the test command)
- [ ] No new lint errors introduced
- [ ] No unrelated files modified
- [ ] No debug code left in (console.log, print, TODO comments from this session)

## Test Changes

- [ ] Tests actually assert something meaningful (no empty tests or `expect(true)`)
- [ ] Tests fail when the feature is removed (not tautological)
- [ ] Tests cover the happy path and at least one error case
- [ ] Tests don't depend on execution order or timing

## Before Committing

- [ ] `git diff` shows only intentional changes
- [ ] No secrets, credentials, or .env values in the diff
- [ ] Commit message describes why, not what

## Before Reporting Done

- [ ] Re-read the original request
- [ ] Compare what was asked vs what was delivered
- [ ] If there's a gap, state it explicitly — don't pretend it's complete
- [ ] If assumptions were made, list them

## Self-Improvement Loops

For `self:*` commands, also verify:
- [ ] Metric actually improved (compare numbers, don't eyeball)
- [ ] The improvement is real, not an artifact of changed test/benchmark
- [ ] Reverted iterations are clean (no partial changes left)

## When to Skip

Skip verification for trivial changes (typo fixes, comment updates, formatting). Apply the full checklist for anything that changes behavior.
