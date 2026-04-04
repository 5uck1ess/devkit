---
name: dry
description: Don't Repeat Yourself — Rule of Three, when duplication is fine, extracting the right abstraction.
---

# DRY (Don't Repeat Yourself)

## The Rule of Three

Don't abstract on the first duplication. Wait for three instances.

1. **First time** — Just write it.
2. **Second time** — Notice the duplication, wince, keep going.
3. **Third time** — Now extract. You have enough examples to see the real pattern.

Premature abstraction is worse than duplication. Wrong abstractions are expensive to undo.

## When Duplication Is Fine

Not all duplication is bad. Duplication is acceptable when:

- **The duplicated code serves different concerns.** Test setup that looks similar to production code isn't "duplication" — they change for different reasons.
- **The cost of coupling exceeds the cost of duplication.** If extracting a shared function means two unrelated modules now depend on each other, keep them separate.
- **The "duplication" is coincidental.** Two things look the same today but will evolve independently. Forcing them to share code creates a false coupling.

The test: if one copy changes, *must* the other change too? If yes, extract. If maybe not, leave it.

## Extracting the Right Abstraction

When you do extract, get the abstraction right:

- **Name it for what it does, not where it came from.** `formatCurrency` not `extractedFromOrderPage`.
- **Parameterize the differences.** The parts that vary between call sites become parameters.
- **Don't over-parameterize.** If the shared function needs 8 config options to handle all cases, the cases are probably too different to share.
- **Keep it simple.** If the abstraction is harder to understand than the duplication, the duplication was better.

## The Real DRY Principle

DRY is about knowledge, not code. The principle says every piece of *knowledge* should have a single authoritative source. Two functions with identical code but representing different business rules are not a DRY violation.

"Every piece of knowledge must have a single, unambiguous, authoritative representation within a system." — Andy Hunt & Dave Thomas
