---
name: scratchpad
description: Persistent iteration memory — prevents Groundhog Day loops by recording what was tried, what failed, and what to try next.
---

# Scratchpad Protocol

Use scratchpads in any iterative loop to prevent repeating failed approaches.

## Location

`.devkit/scratchpads/current.md` — active scratchpad for the current task.

Create `.devkit/scratchpads/` if it doesn't exist. Only one active scratchpad at a time.

## Before Each Iteration

Read `.devkit/scratchpads/current.md` if it exists. Check:
- What approaches were already tried?
- What failed and why?
- What was suggested to try next?

**Do not repeat a failed approach.** If you're about to try something already listed as failed, stop and pick a different strategy.

## After Each Iteration

Append to `.devkit/scratchpads/current.md`:

```markdown
## Iteration {N} — {timestamp}

**Approach:** What you tried (one sentence)
**Result:** pass | fail
**Details:** What happened — error message, unexpected behavior, or success details
**Next:** What to try next if this failed, or "N/A" if it passed
```

## On Completion

When the task succeeds or the workflow ends, delete `.devkit/scratchpads/current.md`.
Don't leave stale scratchpads — they'll confuse the next task.

## Integration with stuck detection

If `.devkit/scratchpads/current.md` shows 3+ failed iterations:
1. Stop iterating
2. Review all failed approaches in the scratchpad
3. The pattern of failures often reveals the real problem
4. Escalate to the user with the scratchpad content as evidence

## Rules

- One scratchpad per active task — don't create per-agent scratchpads
- Keep entries concise — the scratchpad is read every iteration
- Record failures honestly — "it didn't work" is useless; "returned 404 because endpoint expects POST not GET" is useful
- Clean up when done — stale scratchpads are worse than no scratchpads
