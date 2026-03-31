---
name: self:perf
description: Self-improvement loop targeting performance. Iteratively optimizes code with a benchmark command as the gate.
---

# Self-Improve: Performance

Automated loop that profiles, optimizes, benchmarks, and keeps only changes that improve performance.

## Parameters

1. **Target** — file or directory to optimize (required)
2. **Benchmark** — command that measures performance and exits 0 on success (required)
3. **Objective** — what to optimize (e.g., "reduce p99 latency", "improve throughput") (required)
4. **Iterations** — max cycles (default: 10)
5. **Budget** — max USD (default: $2)

## Step 1: Establish Baseline

```bash
git checkout -b self-perf/$(date +%Y%m%d-%H%M%S)
BASELINE=$({benchmark_command} 2>&1)
echo "$BASELINE" > /tmp/self-perf-baseline.txt
echo "$BASELINE" > /tmp/self-perf-best.txt
```

## Step 2: Run the Loop

For each iteration, spawn the `improver` agent:

```
Task: Optimize {target} for: {objective}
Agent: improver
Context:
  - Iteration: {i} of {max}
  - Baseline benchmark: (cat /tmp/self-perf-baseline.txt)
  - Current best: (cat /tmp/self-perf-best.txt)
  - Iteration history: (cat /tmp/self-perf-log.txt)
  - Target file(s): {target}
```

The improver agent:
1. Reads the target code and benchmark results
2. Identifies ONE optimization opportunity
3. Applies the change

Then the orchestrator:
```bash
RESULT=$({benchmark_command} 2>&1)
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
  echo "ITERATION $i: PASS" >> /tmp/self-perf-log.txt
  echo "$RESULT" >> /tmp/self-perf-log.txt
  echo "$RESULT" > /tmp/self-perf-best.txt
  git add -A && git commit -m "self-perf: iteration $i — passed"
else
  echo "ITERATION $i: FAIL — reverting" >> /tmp/self-perf-log.txt
  git checkout -- .
fi
```

## Step 3: Report

```
## Self-Perf Report

**Target:** {target}
**Objective:** {objective}
**Iterations:** {completed} / {total}

### Baseline → Final
{baseline_metrics} → {final_metrics}

### Log
| # | Result | Change |
|---|--------|--------|
| 1 | PASS   | Replaced O(n²) loop with hash map |
| 2 | FAIL   | Caching broke correctness |
| 3 | PASS   | Batched DB queries |

### Next Steps
- Review: `git diff main...HEAD`
- Merge: `git checkout main && git merge self-perf/{branch}`
```

## Presets

```
/self:perf --target src/api/ --benchmark "wrk -t4 -c100 -d5s http://localhost:3000" --objective "reduce p99 latency"
/self:perf --target lib/parser.go --benchmark "go test -bench=. -benchtime=3s" --objective "improve throughput"
```

## Rules

- Uses `improver` agent with worktree isolation
- Always branches first
- One optimization per iteration
- Benchmark must pass (exit 0) to keep
- Discard on failure
- Never sacrifice correctness for speed — run tests too if available
