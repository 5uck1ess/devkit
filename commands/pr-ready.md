---
name: devkit:pr-ready
description: Full PR preparation pipeline — lint, test, security check, changelog, and create PR. One command to go from branch to reviewable PR.
---

# PR Ready

Multi-step pipeline to prepare a branch for PR review.

## Pipeline

### Step 1: Validate Branch

```bash
BRANCH=$(git branch --show-current)
if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
  echo "ERROR: Cannot PR from main/master. Create a feature branch first."
  exit 1
fi

DIFF_STAT=$(git diff main...HEAD --stat)
if [ -z "$DIFF_STAT" ]; then
  echo "ERROR: No changes vs main."
  exit 1
fi
```

### Step 2: Lint (if linter detected)

Auto-detect and run linter:
```bash
# Check for common linters: eslint, prettier, ruff, golangci-lint, clippy, etc.
# Run if found. Report issues but don't block.
```

If fixable errors found, offer to auto-fix.

### Step 3: Test

Auto-detect and run tests:
```bash
# npm test, pytest, go test ./..., cargo test, etc.
```

Report results. Warn if tests fail but don't block.

### Step 4: Security Quick-Check

Spawn the `security-auditor` agent on the diff:

```
Task: Quick security review of this diff — focus on OWASP top 10, hardcoded secrets, SQL injection, XSS.
Agent: security-auditor
Input: git diff main...HEAD
```

Report findings with severity levels.

### Step 5: Generate Changelog

Analyze commits on the branch:
```bash
git log main...HEAD --oneline
```

Categorize changes: features, fixes, refactors, docs, tests.

### Step 6: Create PR

```bash
gh pr create --title "{title}" --body "{body}"
```

Body includes:
- Summary of changes (from commit analysis)
- Test results
- Security findings (if any)
- Changelog

## Output

```
## PR Ready Report

### Pre-flight
- [x] Branch: feature/add-auth
- [x] Lint: 0 errors
- [x] Tests: 14/14 passing
- [⚠] Security: 1 warning (low severity)

### PR Created
{pr_url}

### Changelog
**Features**
- Added JWT authentication middleware

**Fixes**
- Fixed token expiry validation
```

## Presets

```
/devkit:pr-ready
/devkit:pr-ready --skip-security
/devkit:pr-ready --draft
```

## Rules

- Never force-push or modify commit history
- Security findings are advisory, not blocking
- If tests fail, warn but still offer to create PR as draft
- Auto-detect all tooling — don't assume any specific stack
- Use `gh` CLI for PR creation
