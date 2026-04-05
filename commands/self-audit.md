---
description: Automated self-audit — measure the codebase, rank improvement hypotheses by evidence, present actionable plan. Inspired by karpathy/autoresearch.
---

# Self-Audit

Systematic codebase audit inspired by [karpathy/autoresearch](https://github.com/karpathy/autoresearch): one file, one metric, keep or discard, loop. Applied to codebase quality instead of ML training.

## Philosophy

autoresearch gives an AI agent a training setup and lets it experiment autonomously — modify code, measure the metric, keep or revert, repeat ~100 times overnight. We apply the same pattern to codebase quality: measure everything with a single pass, form ranked hypotheses, then test one at a time with a clear keep/discard metric.

This command does NOT fix anything. It measures, analyzes, and presents a ranked hypothesis list. You decide which to test. One at a time.

## Step 1: Detect Stack

```bash
# Detect what's in this repo
HAS_GO=$([ -f go.mod ] || find . -maxdepth 2 -name go.mod -print -quit | grep -q . && echo yes || echo no)
HAS_TS=$([ -f tsconfig.json ] && echo yes || echo no)
HAS_JS=$([ -f package.json ] && echo yes || echo no)
HAS_PYTHON=$([ -f pyproject.toml ] || [ -f requirements.txt ] || [ -f setup.py ] && echo yes || echo no)
HAS_RUST=$([ -f Cargo.toml ] && echo yes || echo no)
HAS_TESTS=$([ -n "$(find . -name '*_test.go' -o -name '*.test.ts' -o -name '*.test.js' -o -name 'test_*.py' -o -name '*_test.rs' 2>/dev/null | head -1)" ] && echo yes || echo no)
HAS_CI=$([ -d .github/workflows ] || [ -f .gitlab-ci.yml ] || [ -f Jenkinsfile ] && echo yes || echo no)
HAS_DOCKER=$([ -f Dockerfile ] || [ -f docker-compose.yml ] && echo yes || echo no)
```

Report what was detected. Skip measurements for stacks not present.

## Step 2: Measure (the data collection phase)

Run ALL applicable measurements. Do not skip any. Collect raw numbers.

**[PARALLEL]** Run these measurement groups concurrently:

### Code Quality

```bash
# Go
go vet ./... 2>&1 | wc -l              # vet issues
go test -cover ./... 2>&1               # coverage per package
gofmt -l . 2>&1 | wc -l                # formatting issues

# TypeScript/JavaScript
npx tsc --noEmit 2>&1 | grep -c 'error TS'  # type errors
npx eslint . --format compact 2>&1 | wc -l   # lint issues

# Python
ruff check . 2>&1 | wc -l              # lint issues
mypy . 2>&1 | grep -c 'error:'         # type errors

# Rust
cargo clippy 2>&1 | grep -c 'error\['  # clippy errors
```

### Test Coverage

```bash
# Get per-package/per-file coverage — the NUMBERS matter
# Go: go test -cover ./...
# TS: npx jest --coverage --coverageReporters text-summary
# Python: pytest --cov --cov-report term-missing
# Rust: cargo tarpaulin --out stdout
```

Record: total coverage %, lowest-coverage packages, untested files.

### Security

```bash
# Dependency vulnerabilities
# Go: govulncheck ./...
# Node: npm audit
# Python: pip-audit or safety check
# Rust: cargo audit

# Hardcoded secrets
grep -rn 'password\s*=\s*"[^"]*"' --include='*.go' --include='*.ts' --include='*.py' . 2>/dev/null | head -5
grep -rn 'api_key\s*=\s*"[^"]*"' --include='*.go' --include='*.ts' --include='*.py' . 2>/dev/null | head -5
```

### Stale Code

```bash
# Dead files (not imported/referenced)
# Unused dependencies
# go mod tidy -diff (shows removable deps)
# npm prune --dry-run
# TODO/FIXME/HACK count
grep -rn 'TODO\|FIXME\|HACK\|XXX' --include='*.go' --include='*.ts' --include='*.py' --include='*.rs' . 2>/dev/null | wc -l
```

### Documentation

```bash
# README exists and is recent?
git log -1 --format='%ai' -- README.md 2>/dev/null
# API docs?
# Changelog?
# Are counts/claims in docs accurate? (run validate-counts if available)
```

### Git Health

```bash
# Recent commit frequency
git log --oneline --since='30 days ago' | wc -l
# Stale branches
git branch -r --merged main | grep -v main | wc -l
# Large files in history
git rev-list --objects --all | git cat-file --batch-check='%(objecttype) %(objectname) %(objectsize) %(rest)' | awk '/^blob/ {print $3, $4}' | sort -rn | head -5
```

## Step 3: Analyze (turn measurements into hypotheses)

```
For each measurement category, form hypotheses:

IF coverage < 50% in a package THEN "Improve test coverage in {package} — currently {N}%"
IF vet/lint issues > 0 THEN "Fix {N} vet/lint issues"
IF vulnerability count > 0 THEN "Patch {N} known vulnerabilities"
IF TODO count > 20 THEN "Triage {N} TODOs — many may be stale"
IF stale branches > 5 THEN "Clean up {N} merged branches"
IF no CI THEN "Add CI pipeline — no automated checks exist"
IF hardcoded secrets found THEN "Move {N} hardcoded secrets to environment variables"
IF lowest coverage package < 30% THEN "Package {name} has {N}% coverage — highest risk area"

For each hypothesis, estimate:
- IMPACT: How much does this improve the codebase? (high/medium/low)
- EFFORT: How hard is this to fix? (high/medium/low)
- EVIDENCE: What measurement supports this? (cite the number)
```

## Step 4: Rank and Present

```
## Self-Audit: {repo-name}

### Stack Detected
{languages, frameworks, CI, Docker, etc.}

### Raw Measurements
| Category | Metric | Value |
|----------|--------|-------|
| Coverage | Overall | X% |
| Coverage | Lowest package | {name} at X% |
| Lint | Issues | N |
| Security | Vulnerabilities | N |
| Stale | TODOs | N |
| Stale | Merged branches | N |
| Git | 30-day commits | N |

### Ranked Hypotheses (by impact/effort ratio)

| # | Hypothesis | Impact | Effort | Evidence |
|---|-----------|--------|--------|----------|
| 1 | {highest impact/effort ratio} | high | low | {measurement} |
| 2 | ... | ... | ... | ... |
| N | ... | ... | ... | ... |

### Recommended Next Steps
1. Test hypothesis #1 first — highest impact for lowest effort
2. Use `/devkit:self-improve` or `/devkit:self-test` to execute
3. Measure again after each fix to verify improvement
4. One change at a time — don't bundle

### What NOT to Do
- Don't fix anything from this report directly — use the suggested devkit command instead
- Don't fix everything at once — one hypothesis at a time
- Don't start with low-impact items just because they're easy
- Don't add features — this is about quality, not functionality
```

## Budget

- **Token budget:** ~200k tokens. Measurement is cheap; the value is in the analysis.
- **Early exit:** If the codebase is clean (no lint issues, >80% coverage, no vulns), say so and skip the hypothesis phase.

## Rules

- Measure EVERYTHING before forming opinions — no guessing
- Cite numbers — every hypothesis must reference a specific measurement
- Rank by impact/effort ratio — not by what's easiest or most interesting
- One at a time — never suggest bundling fixes
- Don't fix anything — this command only measures and analyzes
- Be honest — if the codebase is clean, say so. Don't manufacture problems.
- Suggest the right devkit command for each hypothesis (self-improve, self-test, self-lint, tri-security, etc.)
