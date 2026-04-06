---
description: Autonomous improvement loop inspired by karpathy/autoresearch — audit, fix, measure, keep or revert, repeat.
---

# Autoloop

Autonomous codebase improvement loop. Each cycle: audit → pick hypothesis → fix → measure → keep or revert → repeat.

Inspired by [karpathy/autoresearch](https://github.com/karpathy/autoresearch): one metric, keep or discard, loop.

## Step 0: Harness Detection

```bash
if command -v devkit >/dev/null 2>&1; then
  echo "Go harness detected — delegating to devkit workflow autoloop."
  devkit workflow autoloop "{input}"
  exit 0
fi
```

## Step 1: Gather Inputs

If the input doesn't contain a metric command, use `AskUserQuestion` to collect:

1. **Objective** — what to improve
2. **Metric command** — how to measure (auto-detect from stack if not provided)
3. **Direction** — higher-is-better or lower-is-better
4. **Iterations** — how many cycles (default 10)
5. **Scope** — file/package constraints (optional)

## Step 2: Baseline

Run the metric command. Record the starting number and direction.

## Step 3: Audit

Analyze the codebase for the single highest-impact change. Read the scratchpad to avoid repeating failed approaches. Output one hypothesis with target files.

## Step 4: Fix

Make the recommended change. Minimal, focused, no unrelated refactoring.

## Step 5: Measure

Run the EXACT same metric command. Record the new number.

## Step 6: Compare

Compare baseline vs measurement using the direction:
- higher-is-better: new > old → IMPROVED
- lower-is-better: new < old → IMPROVED
- Equal or failed → REGRESSED

## Step 7: Keep or Revert

- **IMPROVED** → `git add -A && git commit`, update scratchpad, update baseline to new number, loop back to Step 3
- **REGRESSED** → `git checkout -- . && git clean -fd`, update scratchpad with failure reason, loop back to Step 3

## Step 8: Report

After all iterations or budget exhausted:
- Starting vs final metric
- List of kept changes with impact
- List of reverted attempts with failure reason
- Net improvement
- Recommendation for next steps

## Budget

- **Token budget:** ~500k tokens. Each cycle costs ~30-50k tokens.
- **Iteration limit:** User-specified (default 10).
- Budget or iteration limit, whichever hits first, stops the loop.

## Rules

- Every change must be measured — no skipping the metric step
- Never keep a regression — always revert
- One hypothesis at a time — no bundling
- Use the scratchpad to prevent repeating failures
- The metric command must be identical in baseline and measure steps
- Update the baseline number after each kept change (so the next cycle compares against the new state, not the original)
