---
description: Automated self-audit — measure the codebase, rank improvement hypotheses by evidence, present actionable plan.
---

# Self-Audit

Evidence-based codebase health assessment: detect stack → measure → analyze → rank and present.

## Invoke

```
devkit workflow run self-audit
```

If `devkit workflow` is not available, follow this manually:

1. **Detect stack** — Auto-detect languages, frameworks, package managers, test runners, linters
2. **Measure** — Collect data across 6 dimensions:
   - Code quality (lint errors, warnings, complexity)
   - Test coverage (percentage, untested critical paths)
   - Security (vulnerability scan, secrets detection)
   - Stale code (dead code, unused exports, orphan files)
   - Documentation (README freshness, API docs, changelog)
   - Git health (branch age, large files, merge conflicts)
3. **Analyze** — Turn measurements into ranked improvement hypotheses with predicted impact and effort
4. **Report** — Present top 5-10 improvements sorted by impact/effort ratio, with specific file references

## Rules

- Measure first, hypothesize second — no guessing at problems
- Run actual tools — don't estimate metrics
- Rank by evidence strength, not gut feel
- Every recommendation must cite specific measurements
- Token budget: ~200k tokens
