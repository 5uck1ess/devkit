---
name: devkit:audit
description: Unified project health audit — dependencies, vulnerabilities, outdated packages, licenses, lint, and security in one report.
---

# Project Audit

Full-project health check in a single pass. Consolidates dependency vulnerabilities, outdated packages, license compliance, code quality, and security findings into one scored report.

## Parameters

1. **Target** — directory to audit (default: project root)
2. **Budget** — max USD (default: $1)

## Budget

- **Token budget:** ~100k tokens. Mostly tool output from audit commands.
- Prefer `haiku` for parsing/summarizing tool output, `sonnet` for the final report.

## Step 1: Detect Ecosystem

Detect which package managers and tools are present:

```bash
echo "=== Ecosystem Detection ==="
[ -f package.json ] && echo "node: package.json found"
[ -f package-lock.json ] && echo "node: package-lock.json found"
[ -f yarn.lock ] && echo "node: yarn.lock found"
[ -f pnpm-lock.yaml ] && echo "node: pnpm-lock.yaml found"
[ -f requirements.txt ] && echo "python: requirements.txt found"
[ -f pyproject.toml ] && echo "python: pyproject.toml found"
[ -f Pipfile.lock ] && echo "python: Pipfile.lock found"
[ -f go.mod ] && echo "go: go.mod found"
[ -f Cargo.toml ] && echo "rust: Cargo.toml found"
[ -f Gemfile.lock ] && echo "ruby: Gemfile.lock found"
```

## Step 2: Dependency Vulnerabilities

Run the appropriate audit tool for the detected ecosystem:

| Ecosystem | Command |
|-----------|---------|
| Node (npm) | `npm audit --json 2>/dev/null` |
| Node (yarn) | `yarn audit --json 2>/dev/null` |
| Node (pnpm) | `pnpm audit --json 2>/dev/null` |
| Python | `pip-audit --format json 2>/dev/null` or `safety check --json 2>/dev/null` |
| Go | `govulncheck ./... 2>/dev/null` |
| Rust | `cargo audit --json 2>/dev/null` |
| Ruby | `bundle audit check 2>/dev/null` |

If the audit tool isn't installed, note it as "skipped — {tool} not installed" and continue.

Parse the JSON output and extract:
- Total vulnerabilities by severity (critical, high, moderate, low)
- Top 5 most severe findings with package name + advisory

## Step 3: Outdated Packages

```bash
# Node
npm outdated --json 2>/dev/null || true

# Python
pip list --outdated --format json 2>/dev/null || true

# Go
go list -m -u all 2>/dev/null | grep '\[' || true

# Rust
cargo outdated --format json 2>/dev/null || true
```

Categorize as:
- **Major** — breaking version behind (e.g., v3 → v5)
- **Minor** — feature versions behind
- **Patch** — only patch versions behind

## Step 4: License Compliance

Check for potentially problematic licenses in dependencies:

```bash
# Node
npx license-checker --json --production 2>/dev/null | head -200 || true

# Python
pip-licenses --format json 2>/dev/null | head -200 || true
```

Flag:
- **Copyleft** licenses (GPL, AGPL, LGPL) in non-copyleft projects
- **Unknown** or missing licenses
- License conflicts with the project's own license (read LICENSE or package.json license field)

If license tools aren't installed, skip with a note.

## Step 5: Code Quality Summary

Run whatever linter/type checker is configured:

```bash
# Detect and run
[ -f .eslintrc* ] || [ -f eslint.config.* ] && npx eslint . --format json 2>/dev/null | head -100
[ -f tsconfig.json ] && npx tsc --noEmit 2>&1 | tail -5
[ -f pyproject.toml ] && (ruff check . --output-format json 2>/dev/null || flake8 . --format json 2>/dev/null) | head -100
[ -f .golangci.yml ] && golangci-lint run --out-format json 2>/dev/null | head -100
```

Count total errors and warnings. Don't fix anything — just report.

## Step 6: Security Patterns

Spawn the `security-auditor` agent for a quick scan:

```
Task: Quick security scan of {target}. Check for:
  - Hardcoded secrets, API keys, tokens
  - SQL injection patterns
  - XSS vulnerabilities
  - Insecure dependencies usage
  - Exposed debug/admin endpoints
Agent: security-auditor
Report: list of findings with severity and file:line
```

## Step 7: Generate Report

Produce a scored report:

```
## Project Audit Report

**Project:** {name}
**Date:** {date}
**Overall Score:** {score}/100

### Scoring
| Category | Score | Weight | Details |
|----------|-------|--------|---------|
| Vulnerabilities | {0-100} | 30% | {critical}C {high}H {moderate}M {low}L |
| Dependencies | {0-100} | 20% | {major_outdated} major, {minor_outdated} minor behind |
| Licenses | {0-100} | 15% | {issues} issues found |
| Code Quality | {0-100} | 20% | {errors} errors, {warnings} warnings |
| Security | {0-100} | 15% | {findings} findings |

### Critical Findings (fix immediately)
{list critical/high severity items across all categories}

### Warnings (fix soon)
{list moderate items}

### Informational
{list low-severity and suggestions}

### Skipped Checks
{list any tools that weren't available and how to install them}

### Recommended Actions
1. {highest priority fix}
2. {second priority}
3. {third priority}
```

## Scoring Guide

- **Vulnerabilities:** Start at 100. -25 per critical, -10 per high, -3 per moderate, -1 per low.
- **Dependencies:** Start at 100. -5 per major-version-behind package, -1 per minor.
- **Licenses:** Start at 100. -20 per copyleft in non-copyleft project, -10 per unknown license.
- **Code Quality:** Start at 100. -2 per error, -0.5 per warning. Floor at 0.
- **Security:** Start at 100. -25 per critical, -10 per high, -3 per moderate.

Overall = weighted average, rounded to nearest integer.

## Rules

- Never fix anything — audit only, report only
- Always continue if a tool is missing — skip and note it
- Use `--json` output formats where available for reliable parsing
- Cap command output with `head` to avoid context bloat
- Run ecosystem-specific checks only for detected ecosystems
- The security-auditor agent runs in worktree isolation
- Report should be actionable — every finding needs a "what to do" line
