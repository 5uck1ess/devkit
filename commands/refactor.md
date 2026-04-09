---
description: Full lifecycle refactor — analyze code smells, plan transformations, restructure, verify nothing broke.
---

# Refactor

Deterministic refactor lifecycle: analyze → plan → refactor → test → compare.

## Invoke

```
devkit workflow run refactor "{target_and_objective}"
```

If `devkit workflow` is not available, follow this manually:

1. **Analyze** — Identify code smells, duplication, complexity hotspots in the target
2. **Plan** — Create ordered list of transformations; each should be a single testable change
3. **Refactor** — Execute transformations one at a time; run tests after each (loop max 15)
4. **Run tests** — Full test suite must pass
5. **Before/after comparison** — Show what changed: complexity metrics, line counts, readability improvements

## Rules

- Tests must pass after every transformation — no "fix later"
- One transformation at a time
- Preserve behavior — refactoring changes structure, not functionality
- If tests fail, revert the transformation
- Token budget: ~400k tokens
