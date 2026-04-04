---
description: Full PR preparation pipeline — necessity check, DRY review, lint, test, security, changelog, and create PR.
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

### Step 2: Necessity Check

Before anything else, evaluate whether this branch justifies a PR.

Review the full diff and commit history:
```bash
git log main...HEAD --oneline
git diff main...HEAD --stat
git diff main...HEAD
```

Answer these questions:
1. **Does this solve a real problem?** — Is there a bug fix, feature request, or measurable improvement? Or is this churn (renaming for style, reshuffling without functional change)?
2. **Is the scope right?** — Does the branch do one coherent thing, or is it a grab-bag of unrelated changes that should be separate PRs?
3. **Does this duplicate existing capability?** — Is there already a command, function, or tool in the codebase that does the same thing? Search for overlapping functionality.
4. **Is the approach justified?** — Could the same result be achieved with a simpler change (config tweak, one-line fix, existing tool)?

Report a verdict:
- **Necessary** — clear justification, proceed with pipeline
- **Questionable** — concerns exist, list them and ask the user before proceeding
- **Unnecessary** — this branch adds no real value, explain why and recommend against PR

If the verdict is "Unnecessary", stop the pipeline and explain. If "Questionable", use `AskUserQuestion` to confirm before continuing.

### Step 3: DRY Review

**[PARALLEL with Step 4-5]** Spawn the `reviewer` agent:

```
Task: DRY and code quality review of this branch's changes.
Agent: reviewer
Input: git diff main...HEAD

Review the diff for:

1. **Duplication within the diff** — Are there repeated blocks of text, logic, or structure
   in the new/changed code that should be extracted or consolidated?

2. **Duplication against existing code** — Do the changes duplicate functionality that
   already exists elsewhere in the codebase? Search for similar patterns, function names,
   and logic in the repo.

3. **Abstraction quality** — If the changes introduce shared code or utilities, is the
   abstraction well-named, properly parameterized, and not over-engineered?

4. **Rule of Three** — Flag duplication only when there are 3+ instances. Two similar
   blocks are fine. Don't suggest premature abstractions.

Apply the DRY principle correctly: DRY is about knowledge, not code. Two functions with
identical code but representing different concerns are NOT a violation. Only flag
duplication where a change in one copy would require changing the other.

Report:
- DRY violations found (with file paths and line numbers)
- Suggestions for extraction or consolidation
- Cases where duplication is acceptable and why
```

### Step 4: Lint (if linter detected)

Auto-detect and run linter:
```bash
# Check for common linters: eslint, prettier, ruff, golangci-lint, clippy, etc.
# Run if found. Report issues but don't block.
```

If fixable errors found, offer to auto-fix.

### Step 5: Test

Auto-detect and run tests:
```bash
# npm test, pytest, go test ./..., cargo test, etc.
```

Report results. Warn if tests fail but don't block.

### Step 6: Security Quick-Check

Spawn the `security-auditor` agent on the diff:

```
Task: Quick security review of this diff — focus on OWASP top 10, hardcoded secrets, SQL injection, XSS.
Agent: security-auditor
Input: git diff main...HEAD
```

Report findings with severity levels.

### Step 7: Generate Changelog

Analyze commits on the branch:
```bash
git log main...HEAD --oneline
```

Categorize changes: features, fixes, refactors, docs, tests.

### Step 8: Create PR

```bash
gh pr create --title "{title}" --body "{body}"
```

Body includes:
- Summary of changes (from commit analysis)
- Necessity justification (from Step 2)
- DRY review findings (from Step 3)
- Test results
- Security findings (if any)
- Changelog

## Output

```
## PR Ready Report

### Pre-flight
- [x] Branch: feature/add-auth
- [x] Necessity: Justified — adds JWT auth required by PROJ-142
- [x] DRY: No violations found
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
/devkit:pr-ready --skip-necessity
/devkit:pr-ready --draft
```

## Rules

- Never force-push or modify commit history
- Necessity check gates the pipeline — unnecessary branches don't get PRs
- DRY review uses the Rule of Three — don't flag premature abstractions
- Security findings are advisory, not blocking
- If tests fail, warn but still offer to create PR as draft
- Auto-detect all tooling — don't assume any specific stack
- Use `gh` CLI for PR creation
- Steps 3, 4, and 5 (DRY, lint, test) run in parallel when possible
