---
description: Incremental migration loop — migrate code one piece at a time with tests as safety gate.
---

# Self-Migrate

Deterministic migration loop: run tests → migrate one piece → gate check → repeat until gate exits 0.

## Invoke

```
devkit workflow run self-migrate "{test_command}"
```

If `devkit workflow` is not available, follow this manually:

1. **Baseline** — Run tests to confirm green starting point
2. **Migrate loop** — Migrate one file or closely-related group per iteration; update imports/references; re-run gate command after each step (max 20 iterations). To detect migration completeness, use a gate command that exits non-zero until all files are converted (e.g., `npm test && ! grep -r 'require(' src/`).
3. **Verify** — Run tests one final time
4. **Summary** — Report which files were migrated and what remains

## Rules

- One file or closely-related group per iteration
- Tests must pass to keep — no exceptions
- Preserve existing behavior — migration, not refactoring
- Update imports and references in the same iteration
