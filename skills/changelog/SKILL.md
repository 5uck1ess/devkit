---
name: changelog
description: Generate a structured changelog from git history — use when asked to create a changelog, release notes, or summarize what changed between versions/tags/branches.
---

# Changelog Generation

Generate a structured changelog from git commits between two refs.

## Parameters

1. **From** — start ref (default: last tag, or first commit if no tags)
2. **To** — end ref (default: HEAD)
3. **Format** — output format (default: markdown)

## Step 1: Gather Commits

```bash
# Find the range
FROM=${from:-$(git describe --tags --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)}
TO=${to:-HEAD}

git log ${FROM}..${TO} --oneline --no-merges
```

## Step 2: Categorize

Analyze each commit and categorize:

- **Features** — new functionality (`feat:`, `add:`, new files)
- **Fixes** — bug fixes (`fix:`, `bug:`, `patch:`)
- **Performance** — optimizations (`perf:`, benchmark improvements)
- **Refactoring** — code changes with no behavior change (`refactor:`, `chore:`)
- **Documentation** — doc changes (`docs:`, README, comments)
- **Tests** — test additions/changes (`test:`)
- **Breaking Changes** — anything that changes public API or behavior

## Step 3: Output

```
## Changelog: {from} → {to}

### Features
- Added JWT authentication middleware (#42)
- New `/api/health` endpoint

### Fixes
- Fixed race condition in cache invalidation (#38)
- Corrected timezone handling in date parser

### Performance
- Optimized database queries — 2x faster list endpoints

### Breaking Changes
- Removed deprecated `/api/v1/users` endpoint — use `/api/v2/users`

### Other
- Updated dependencies
- Added CI pipeline for ARM builds
```

## Presets

```
/devkit:changelog
/devkit:changelog --from v1.2.0 --to v1.3.0
/devkit:changelog --from main --to feature/auth
```

## Rules

- Use actual commit messages — don't fabricate changes
- Group related commits together
- Include PR/issue numbers if present in commit messages
- Highlight breaking changes prominently
- Skip merge commits and trivial changes (typo fixes, formatting)
- If conventional commits are used, respect the prefixes
