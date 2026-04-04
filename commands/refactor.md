---
description: Full lifecycle refactor — analyze code smells, plan transformations, restructure, verify nothing broke.
---

# Refactor Workflow

Complete refactor lifecycle: analyze → plan → restructure → verify → compare.

## Budget & Early Exit

- **Token budget:** ~400k tokens. Refactors can touch many files.
- **Early exit:** Stop the refactor loop when all planned steps are complete and tests pass.
- **Stuck detection:** If 3 consecutive refactor steps break tests, stop and report. See the `stuck` skill.

## Step 1: Analyze

```
Analyze the code to refactor: {user's input}

Identify:
- Code smells (duplication, long functions, deep nesting, god objects)
- Complexity hotspots
- Coupling issues
- Naming problems
- What the code SHOULD look like after refactoring

Don't change anything yet — just assess.
```

## Step 2: Plan

```
Based on the analysis, create a step-by-step refactoring plan.

Each step should be a single, safe transformation that preserves behavior.
Order matters — do renames before extractions, extractions before moves.
Include "run tests" checkpoints between risky steps.
```

## Step 3: Refactor

```
Execute the next incomplete step from the plan.
Make the change, verify it compiles, and confirm behavior is preserved.
```

Loop until all steps are complete. Max 15 iterations.

## Step 4: Run Tests

Run the full test suite to verify the refactoring didn't break anything. If tests fail, fix them — update tests to match the new structure, or fix regressions. Loop up to 6 times until all pass.

## Step 5: Before/After Comparison

```
## What Changed
List each transformation that was made.

## Improvements
What's better now (readability, complexity, coupling).

## Metrics
Lines added/removed, functions extracted, files changed.

## Risk Areas
Anything that should be watched closely after this refactor.
```

## Rules

- Analyze before changing — understand the full picture first
- Behavior-preserving transformations only — refactoring must not change what the code does
- One transformation per step — don't combine rename + extract + move
- Order matters — renames before extractions, extractions before moves
- Test between risky steps — verify nothing broke at each checkpoint
- All tests must pass before reporting done
- Don't add features during a refactor — separate concerns
