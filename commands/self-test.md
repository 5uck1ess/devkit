---
description: Run tests, fix failures deterministically, repeat until all pass.
---

# Self-Test

Deterministic test-fix loop: run tests → fix failures → gate check → repeat until exit code 0.

## Invoke

```
devkit workflow run self-test "{test_command}"
```

If `devkit workflow` is not available, follow this manually:

1. **Baseline** — Run the test command and capture output
2. **Fix loop** — Fix one group of related failures at a time; re-run tests after each fix; stop when all pass (max 8 iterations)
3. **Verify** — Run tests one final time
4. **Summary** — Report what was fixed and iteration count

## Rules

- One group of related fixes per iteration
- The bug might be in the test or the code under test
- Don't refactor unrelated code
- Match existing test conventions
