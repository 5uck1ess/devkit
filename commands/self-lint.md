---
description: Run linter, fix violations deterministically, repeat until clean.
---

# Self-Lint

Deterministic lint-fix loop: run linter → fix violations → gate check → repeat until exit code 0.

## Invoke

```
devkit workflow run self-lint "{lint_command}"
```

If `devkit workflow` is not available, follow this manually:

1. **Baseline** — Run the lint command and capture output
2. **Fix loop** — Fix one group of related lint/type issues at a time; re-run linter after each fix; stop when clean (max 20 iterations)
3. **Verify** — Run linter one final time
4. **Summary** — Report what was fixed and what remains

## Rules

- Prioritize errors over warnings
- Fix one group of related issues at a time
- Don't change code behavior — only fix lint issues
- Never disable lint rules — fix the underlying issue
