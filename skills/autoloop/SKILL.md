---
name: autoloop
description: Autonomous improvement loop inspired by karpathy/autoresearch — use when asked to run an autoloop, auto loop, autonomous improvement, or run experiments overnight on the codebase.
---

# Autoloop

Autonomous codebase improvement inspired by [karpathy/autoresearch](https://github.com/karpathy/autoresearch). Audit the codebase, pick the highest-impact change, fix it, measure the result, keep or revert, repeat.

## Before Starting

Use `AskUserQuestion` to gather these inputs. Do NOT proceed without answers.

### 1. Objective

Ask: "What do you want to improve? (e.g., test coverage, lint errors, performance, security)"

### 2. Metric Command

Ask: "What command measures success? (e.g., `go test -cover ./...`, `npx jest --coverage`, `ruff check . | wc -l`)"

If the user doesn't have one, detect the stack and suggest:

| Stack | Default metric | Direction |
|-------|---------------|-----------|
| Go | `go test -cover ./...` | higher-is-better (coverage %) |
| TypeScript | `npx jest --coverage` | higher-is-better (coverage %) |
| Python | `pytest --cov` | higher-is-better (coverage %) |
| Rust | `cargo test` | higher-is-better (pass count) |
| Go (lint) | `go vet ./... 2>&1 \| wc -l` | lower-is-better (error count) |
| Any (lint) | `<linter> . 2>&1 \| wc -l` | lower-is-better (error count) |

Confirm with the user: "I'll use `<command>` with <direction>. Correct?"

### 3. Direction

If not obvious from the metric, ask: "Is higher or lower better for this metric?"

### 4. Iterations

Ask: "How many improvement cycles? (default: 10, max recommended: 50)"

### 5. Scope (optional)

Ask: "Any scope constraints? (e.g., only `src/engine/`, only `.py` files, or everything)"

If the user says "everything" or skips, leave scope open.

## Invoke the Workflow

Ensure the devkit engine is installed:

```bash
ENSURE="$(find ~/.claude/plugins ${APPDATA:+$APPDATA/.claude/plugins} ${LOCALAPPDATA:+$LOCALAPPDATA/.claude/plugins} -path '*/devkit/scripts/ensure-engine.sh' 2>/dev/null | head -1)"; [ -n "$ENSURE" ] && bash "$ENSURE" || { echo "devkit plugin not found — install from https://github.com/5uck1ess/devkit/releases"; exit 1; }
```

Assemble the input as a single string and invoke:

```bash
devkit workflow autoloop "<objective> | metric: <command> | direction: <higher/lower>-is-better | iterations: <N> | scope: <constraint or 'all'>"
```

If the engine cannot be installed, tell the user: "The devkit engine binary is required for deterministic workflow execution. Install manually from https://github.com/5uck1ess/devkit/releases" Do NOT fall back to manual steps — the engine is required for determinism.

## Rules

- Never skip the measurement step — every change must be measured
- Never keep a change that regressed the metric — always revert
- One hypothesis at a time — don't bundle changes
- Use the scratchpad (.devkit/scratchpads/current.md) to avoid repeating failed approaches
- Stop when budget is exhausted, iterations are done, or metric stops improving
- Be honest in the report — include reverted attempts, not just successes
