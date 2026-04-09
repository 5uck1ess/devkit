---
description: Unified project health audit — dependencies, vulnerabilities, outdated packages, licenses, lint, and security.
---

# Project Audit

Deterministic project health check: detect ecosystem → audit dependencies → check licenses → lint → security scan → report.

## Invoke

```
devkit workflow run audit
```

If `devkit workflow` is not available, follow this manually:

1. **Detect ecosystem** — Check for go.mod, package.json, requirements.txt/pyproject.toml, Cargo.toml
2. **Dependency vulnerabilities** — Run ecosystem audit tools (`npm audit`, `go vuln`, `pip-audit`, `cargo audit`)
3. **Outdated packages** — Check for outdated dependencies, report major/minor/patch updates available
4. **License compliance** — Scan dependency licenses, flag non-permissive or unknown licenses
5. **Code quality** — Run linters for detected ecosystems, count errors and warnings
6. **Security patterns** — Grep for common vulnerabilities: hardcoded secrets, SQL injection, command injection, path traversal
7. **Generate report** — Health score (A-F) with breakdown by category, actionable recommendations

## Rules

- Auto-detect ecosystem — don't ask the user what language
- Run actual audit tools — don't guess at vulnerabilities
- Score overall health: A (excellent) through F (critical issues)
- Prioritize findings by severity
- Token budget: ~100k tokens
