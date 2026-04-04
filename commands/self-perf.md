---
description: Hypothesis-driven performance investigation — analyze, hypothesize, test one theory at a time, measure against baseline.
---

# Self-Improve: Performance

Structured performance optimization that forms hypotheses from evidence before changing code. Tests one theory at a time against a stable baseline.

## Parameters

1. **Benchmark command** — command that outputs a measurable metric (required)
2. **Target** — file or directory scope (default: entire project)
3. **Metric name** — what the benchmark measures, e.g. "request latency ms" (required)
4. **Goal** — target metric value, e.g. "< 200" (optional)
5. **Iterations** — max investigation cycles (default: 10)
6. **Budget** — max USD (default: $5)

## Budget & Early Exit

- **Token budget:** ~400k tokens. Investigation is more expensive than blind optimization.
- **Early exit:** Stop when goal is met or no viable hypotheses remain.
- **Stuck detection:** If 3 consecutive hypotheses fail, escalate to user. See the `stuck` skill.

## Step 1: Establish Baseline

Run the benchmark at least 3 times for stability:

```bash
git checkout -b perf/$(date +%Y%m%d-%H%M%S)

echo "=== Baseline Run 1 ===" && {benchmark_command}
echo "=== Baseline Run 2 ===" && {benchmark_command}
echo "=== Baseline Run 3 ===" && {benchmark_command}
```

Record the median result as the baseline. Reject runs with >20% variance — the benchmark isn't stable enough for meaningful optimization.

Save baseline:
```bash
mkdir -p .devkit/perf
cat > .devkit/perf/baseline.json << 'BASELINE'
{
  "metric": "{metric_name}",
  "value": {median_value},
  "unit": "{unit}",
  "runs": [{run1}, {run2}, {run3}],
  "variance_pct": {variance},
  "commit": "{commit_hash}",
  "timestamp": "{iso8601}"
}
BASELINE
```

## Step 2: Gather Evidence

Before forming hypotheses, collect data from multiple sources:

### 2a. Git History Analysis

```bash
# Find performance-related commits
git log --all --oneline --grep="perf" --grep="slow" --grep="optimize" --grep="cache" --grep="latency" --grep="memory" | head -20

# Find recent changes to hot paths
git log --oneline -20 -- {target}

# Find large commits that may have introduced regressions
git log --oneline --diff-filter=M --stat | head -30
```

### 2b. Code Path Analysis

Spawn the `researcher` agent:

```
Task: Analyze the critical code paths in {target} for performance.
Agent: researcher
Focus on:
  - Hot loops and recursive calls
  - I/O operations (file, network, database)
  - Memory allocation patterns (large objects, frequent allocations)
  - Synchronous operations that could be async
  - Missing caching opportunities
  - N+1 query patterns
  - Unnecessary serialization/deserialization
  - Redundant computation
Report: list of suspicious code paths with file:line references
```

### 2c. Profiling Data (if available)

```bash
# Check for existing profiling tools
command -v perf >/dev/null 2>&1 && echo "perf available"
command -v hyperfine >/dev/null 2>&1 && echo "hyperfine available"
[ -f flamegraph.svg ] && echo "flamegraph found"
```

If profiling tools are available, run a quick profile to identify actual hotspots.

## Step 3: Form Hypotheses

Based on the evidence, form up to 5 ranked hypotheses:

```
## Hypotheses

| # | Hypothesis | Evidence | Confidence | Expected Impact |
|---|-----------|----------|------------|-----------------|
| 1 | N+1 queries in getUserOrders | researcher found loop with individual DB calls at orders.ts:45 | High | 3-5x faster |
| 2 | Missing cache for config parsing | parseConfig called 12x per request, git shows it was recently changed | Medium | 20-30% faster |
| 3 | Synchronous file reads in middleware | blocking I/O in request path at middleware.ts:23 | Medium | 10-20% faster |
| 4 | Large JSON serialization in logging | JSON.stringify on full request objects at logger.ts:67 | Low | 5-10% faster |
| 5 | Regex compilation on every call | new RegExp() inside loop at validator.ts:12 | Low | 5% faster |
```

Rules for hypotheses:
- Each must cite specific evidence (file:line, git commit, profiler output)
- Each must predict the expected impact (not just "faster")
- Confidence is based on evidence strength, not gut feel
- Order by confidence * expected impact (highest first)

## Step 4: Test Hypotheses (One at a Time)

For each hypothesis, starting from highest-ranked:

### 4a. Implement the Fix

Spawn the `improver` agent:

```
Task: Optimize {target} based on this hypothesis:
  Hypothesis: {hypothesis_description}
  Evidence: {evidence}
  File: {file_path}:{line}
Agent: improver
Constraints:
  - Change ONLY what this hypothesis addresses
  - Do not refactor unrelated code
  - Preserve all existing behavior
  - Keep the change as small as possible
```

### 4b. Verify Correctness

```bash
# Run tests first — optimization must not break anything
{test_command} || echo "TESTS FAILED — reverting"
```

If tests fail, revert and move to next hypothesis.

### 4c. Measure Impact

Run benchmark 3 times again:

```bash
echo "=== Post-fix Run 1 ===" && {benchmark_command}
echo "=== Post-fix Run 2 ===" && {benchmark_command}
echo "=== Post-fix Run 3 ===" && {benchmark_command}
```

### 4d. Evaluate

Compare median against baseline:

```bash
IMPROVEMENT=$(( (BASELINE - NEW_MEDIAN) * 100 / BASELINE ))
```

Decision:
- **Improvement matches or exceeds prediction** → Keep. Commit.
  ```bash
  git add -A && git commit -m "perf: {hypothesis summary} ({improvement}% improvement)"
  ```
- **Improvement exists but below prediction** → Keep if >5% improvement, otherwise revert.
- **No improvement or regression** → Revert. Log why hypothesis was wrong.
  ```bash
  git checkout -- .
  ```

Log the result:
```bash
echo "HYPOTHESIS {n}: {PASS|FAIL} — predicted {predicted}%, actual {actual}%" >> .devkit/perf/investigation.log
echo "  Evidence: {evidence}" >> .devkit/perf/investigation.log
echo "  Lesson: {why it worked or didn't}" >> .devkit/perf/investigation.log
```

### 4e. Update Baseline

If the fix was kept, the new median becomes the baseline for subsequent hypotheses.

## Step 5: Report

```
## Performance Investigation Report

**Target:** {target}
**Metric:** {metric_name}
**Baseline:** {baseline_value} {unit}
**Final:** {final_value} {unit}
**Total improvement:** {total_improvement}%
**Goal:** {goal} — {met|not met}

### Hypotheses Tested
| # | Hypothesis | Predicted | Actual | Result |
|---|-----------|-----------|--------|--------|
| 1 | N+1 queries in getUserOrders | 3-5x | 3.2x | PASS — kept |
| 2 | Missing cache for config | 20-30% | 22% | PASS — kept |
| 3 | Sync file reads | 10-20% | 2% | FAIL — below threshold, reverted |
| 4 | JSON serialization | 5-10% | — | SKIPPED — goal already met |

### Investigation Log
{contents of .devkit/perf/investigation.log}

### Commits
| Commit | Hypothesis | Improvement |
|--------|-----------|-------------|
| abc1234 | Batch N+1 queries | 3.2x |
| def5678 | Add config cache | 22% |

### Remaining Hypotheses (untested)
- Large JSON serialization in logging — estimated 5-10%
- Regex compilation on every call — estimated 5%

### Next Steps
- Review: `git diff main...HEAD`
- Merge: `git checkout main && git merge perf/{branch}`
```

## Presets

```
/self:perf --target src/api/ --benchmark "wrk -t4 -c100 -d5s http://localhost:3000" --metric "p99 latency ms" --goal "< 200"
/self:perf --target lib/parser.go --benchmark "go test -bench=. -benchtime=3s" --metric "ns/op" --goal "< 1000"
```

## Rules

- Always branch first
- Never skip the evidence-gathering step — blind optimization is guessing
- Test ONE hypothesis at a time — never bundle changes
- Run benchmarks 3x minimum — single runs are unreliable
- Reject benchmarks with >20% variance
- Tests must pass before measuring — broken code isn't faster code
- Log every hypothesis outcome with the lesson learned
- Revert on regression or no improvement — don't keep dead changes
- Stop when goal is met — don't over-optimize
- The improver agent runs in worktree isolation
- The researcher agent runs in worktree isolation for code analysis
