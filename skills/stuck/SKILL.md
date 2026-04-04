---
name: devkit:stuck
description: Detect when an agent is looping or failing repeatedly, and trigger structured recovery — backtrack, simplify, or escalate.
---

# Stuck Detection & Recovery

## Symptoms

You are stuck if any of these are true:
- Same error message appeared 2+ times in a row
- You reverted the same change twice
- The metric hasn't improved in 3+ iterations
- You're making changes that don't address the actual error
- You're adding complexity to work around a problem instead of fixing it

## Recovery Protocol

### 1. Stop and Diagnose

Don't retry the same approach. Read the error carefully:
- What is the **actual** error message?
- What file and line does it point to?
- Is this the same error as last iteration, or a new one?

### 2. Backtrack

If the last 2+ attempts failed:
- Revert to the last known-good state: `git checkout -- .`
- Re-read the original goal and the current state of the code
- The problem may be upstream of where you're looking

### 3. Simplify

Reduce scope:
- Fix one thing at a time, not three
- Remove the last thing you added and see if the base case works
- Use the simplest possible implementation, even if inelegant

### 4. Change Strategy

If the same approach failed twice:
- Try a fundamentally different approach, not a variation
- Read the docs or source of the library/API you're calling
- Check if the function signature or types changed

### 5. Escalate

If you've tried 3+ different approaches:
- Report what you've tried and what failed
- Show the exact error
- Ask the user for guidance — they may know something you don't

## Anti-Patterns

- **Retry loop** — Running the same command hoping for a different result
- **Shotgun debugging** — Changing multiple things at once without understanding why
- **Complexity spiral** — Adding workarounds on top of workarounds
- **Ignoring the error** — Suppressing or catching exceptions instead of fixing the cause
