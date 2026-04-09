---
description: Full PR preparation pipeline — validate branch, DRY review, lint, test, security, changelog, create PR.
---

# PR Ready

Deterministic PR preparation: validate → necessity check → DRY review → lint → test → security → changelog → create PR.

## Invoke

```
devkit workflow run pr-ready
```

If `devkit workflow` is not available, follow this manually:

1. **Validate branch** — Confirm not on main/master, check for uncommitted changes, verify remote tracking
2. **Necessity check** — Review each changed file: is it needed? Remove unnecessary additions, debug artifacts, or unrelated changes.
3. **DRY review** — Check for duplicated logic across changed files. Extract shared code if 3+ repetitions.
4. **Lint** — Run project linter on changed files. Fix violations.
5. **Test** — Run full test suite. Fix failures.
6. **Security quick-check** — Scan for: hardcoded secrets, SQL injection, XSS, command injection, path traversal, insecure dependencies
7. **Generate changelog** — Create changelog entry from git diff summarizing what changed and why
8. **Create PR** — Push branch, create PR with title + summary + test plan

## Rules

- Every step must pass before proceeding to the next
- Remove unnecessary changes before reviewing code quality
- Fix lint and test failures — don't skip them
- Security check is not optional
- Changelog should explain WHY, not just WHAT
