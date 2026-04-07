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
6. **Guard** — optional safety-net command that must always pass (e.g., `npm test`, `tsc --noEmit`). Changes that improve the metric but break the guard are treated as regressions and reverted. Guard commands must be side-effect free (no generated files, snapshots, or caches that persist between runs).

## Step 2: Baseline

Run the metric command. Record the starting number and direction.

If a guard command is set, run it now. If the guard fails at baseline, stop and tell the user — the invariant must hold before the loop can start.

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

## Step 6.5: Guard Check (if guard command is set)

Guard is only evaluated when the metric improved. Regressions are already reverted in Step 7, so running the guard on them would be wasted work. If the metric improved, run the guard with a timeout:

```bash
timeout 120 {guard_command} 2>&1
```

- Guard passes (exit 0) → proceed to Step 7 as IMPROVED
- Guard fails or times out → treat as REGRESSED regardless of metric improvement. Log: "Metric improved but guard failed — reverting to protect invariant."

## Step 7: Keep or Revert

- **IMPROVED** → stage only modified files + `git commit`, update scratchpad, update baseline to new number, loop back to Step 3
- **REGRESSED** → `git checkout -- <modified files>` (no `git clean -fd` — protect untracked work), update scratchpad with failure reason, increment per-file failure counter (see Escalation below), loop back to Step 3

### Escalation: Repeated Failures

Track consecutive failures by primary modified file. After 3 failed attempts where the same file is the main edit target:

1. **Log** what was tried and why each approach failed
2. **Skip** — move the hypothesis to a "blocked" list in the scratchpad
3. **Pivot** — choose a completely different hypothesis targeting different files
4. **Report** — include blocked items in the final report with context for manual investigation

Never loop on the same failing approach. Each attempt must use a materially different strategy.

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
- Guard failures override metric improvements — never accept a change that breaks the guard
- 3 failures on the same file → skip and pivot, don't grind
