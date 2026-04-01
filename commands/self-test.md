---
name: self:test
description: Self-improvement loop targeting test coverage. Iteratively generates and improves tests until a coverage target is met or iterations exhausted.
---

# Self-Improve: Test Coverage

Automated loop that generates tests, runs them, measures coverage, and iterates until the target is hit.

## Parameters

1. **Target** — file or directory to generate tests for (required)
2. **Test command** — command that runs tests and reports coverage (required)
3. **Coverage target** — percentage to aim for (default: 80)
4. **Iterations** — max cycles (default: 10)
5. **Budget** — max USD (default: $2)

## Budget & Early Exit

- **Token budget:** ~300k tokens. If approaching limit, reduce remaining iteration count.
- **Early exit:** Stop immediately when coverage target is met — don't run remaining iterations.
- **Stuck detection:** If 3 consecutive iterations fail (tests break), stop and report. See the `stuck` skill.

## Step 1: Detect Test Framework

```bash
# Auto-detect from package.json, pyproject.toml, go.mod, Cargo.toml, etc.
# Identify existing test files, patterns, and coverage tooling
```

If no test command provided, infer from project config. Ask user to confirm.

## Step 2: Establish Baseline

```bash
git checkout -b self-test/$(date +%Y%m%d-%H%M%S)
BASELINE=$({test_command} 2>&1)
echo "$BASELINE" > /tmp/self-test-baseline.txt
```

Extract current coverage percentage from output. If no tests exist yet, baseline is 0%.

## Step 3: Run the Loop

For each iteration, spawn the `test-writer` agent as a background task:

```
Task: Generate or improve tests for {target} to increase coverage.
Agent: test-writer
Context:
  - Iteration: {i} of {max}
  - Current coverage: {current}%
  - Target coverage: {coverage_target}%
  - Iteration history: (cat /tmp/self-test-log.txt)
  - Target file(s): {target}
  - Existing tests: {test_files}
```

The test-writer agent:
1. Reads the target source and existing tests
2. Identifies uncovered code paths
3. Writes or improves ONE test file

Then the orchestrator:
```bash
RESULT=$({test_command} 2>&1)
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
  COVERAGE=$(echo "$RESULT" | grep -oE '[0-9]+\.?[0-9]*%' | tail -1)
  echo "ITERATION $i: PASS — coverage $COVERAGE" >> /tmp/self-test-log.txt
  git add -A && git commit -m "self-test: iteration $i — coverage $COVERAGE"
else
  echo "ITERATION $i: FAIL — tests broke, reverting" >> /tmp/self-test-log.txt
  git checkout -- .
fi
```

Stop early if coverage target is met.

## Step 4: Report

```
## Self-Test Report

**Target:** {target}
**Coverage:** {baseline}% → {final}%  (target: {coverage_target}%)
**Iterations:** {completed} / {total}

### Log
| # | Result | Coverage | Change |
|---|--------|----------|--------|
| 1 | PASS   | 45%      | Added unit tests for parser |
| 2 | FAIL   | —        | Tests broke — reverted |
| 3 | PASS   | 62%      | Added edge case tests |

### Next Steps
- Review: `git diff main...HEAD`
- Merge: `git checkout main && git merge self-test/{branch}`
- Discard: `git checkout main && git branch -D self-test/{branch}`
```

## Rules

- Uses `test-writer` agent with worktree isolation
- Always branches first — never modifies main
- One test file per iteration
- Discard on failure — `git checkout -- .`
- Stop early if target coverage reached
- Match existing test conventions (file naming, framework, patterns)
- Never modify source code — only test files
