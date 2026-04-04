---
name: yagni
description: You Aren't Gonna Need It — build only what's needed now, no speculative features or premature abstractions.
---

# YAGNI (You Aren't Gonna Need It)

## The Principle

Build the thing you need today. Not the thing you might need tomorrow.

Every line of speculative code has a cost: it must be understood, tested, maintained, and debugged. Code you don't write has zero bugs.

## Common Violations

**"Just in case" abstractions**
> "Let's make this a plugin system in case we need to swap implementations."

You don't have a second implementation. Build for the one you have. Refactoring later is almost always cheaper than maintaining an abstraction you don't need yet.

**Premature configurability**
> "Let's make the retry count configurable."

Hardcode it. If someone actually needs to change it, you'll make it configurable then — in 5 minutes. Until then, a config option is a decision you're forcing on every deployer.

**Speculative generalization**
> "This handles orders, but let's make it handle any entity."

You have orders. Build for orders. When you get a second entity type, you'll understand the *actual* commonalities (not the ones you imagined).

**Premature optimization**
> "Let's add caching in case this is slow."

Measure first. Most performance intuitions are wrong. Optimize when you have evidence, not anxiety.

## How to Apply

Before building something, ask:

1. **Do I need this to ship the current feature?** If no, don't build it.
2. **Am I solving a real problem or an imagined one?** Real problems have evidence.
3. **What's the cost of adding this later vs now?** Usually negligible. The code will be better because you'll understand the actual requirements.

## YAGNI Is Not "Never Refactor"

YAGNI says don't build features you don't need. It does *not* say write sloppy code. Clean code, good names, small functions, and tests are always needed — they serve today's requirements.

The goal: code that does exactly what's needed, no more, no less, and is easy to change when actual new requirements arrive.
