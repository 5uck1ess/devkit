---
description: Full lifecycle feature development — brainstorm, plan, implement, test, lint, review.
---

# Feature

Deterministic feature lifecycle: triage → brainstorm → plan → implement → test → lint → review → report.

## Invoke

```
devkit workflow run feature "{feature_description}"
```

If `devkit workflow` is not available, follow this manually:

1. **Triage** — Classify as TINY / SMALL / MEDIUM / LARGE. Tiny changes skip to quick-fix path.
2. **Brainstorm** — Think through design: what components change, simplest approach, risks, edge cases. Produce a short design summary. (SMALL scope skips this step and goes directly to Plan.)
3. **Plan** — Create numbered implementation todo list ordered by dependency
4. **Implement** — Execute one todo at a time; small focused changes; track progress in scratchpad (loop max 20)
5. **Generate tests** — Write tests for the new feature covering happy path, edge cases, and error conditions
6. **Run tests** — Run full test suite
7. **Fix test failures** — Fix any failures, determine if bug is in test or implementation (loop max 8)
8. **Lint** — Run linter on changed files; fix violations if any (loop max 4)
9. **Review** — Parallel smart + fast review of all changes
10. **Final report** — Summary of what was built, test coverage, review findings, and status

## Rules

- Triage honestly — most changes are smaller than they seem
- Design before implementing — don't jump to code
- One todo per iteration — keep changes small and focused
- Tests must pass before review
- Lint must be clean before review
- Use scratchpad (`.devkit/scratchpads/current.md`) to track progress across iterations
