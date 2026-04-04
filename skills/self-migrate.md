---
name: self:migrate
description: Self-improvement loop for incremental codebase migrations. Iteratively migrates code with tests as the safety gate.
---

# Self-Improve: Migration

Automated, incremental migration loop. Each iteration migrates a small piece, runs tests, keeps if green, reverts if red.

## Parameters

1. **Target** — file or directory to migrate (required)
2. **Test command** — command that validates correctness (required)
3. **Migration** — what to migrate to (required, e.g., "TypeScript", "React hooks", "Python 3.12", "ES modules")
4. **Iterations** — max cycles (default: 20)
5. **Budget** — max USD (default: $3)

## Budget & Early Exit

- **Token budget:** ~400k tokens. Migrations can touch many files.
- **Early exit:** Stop when all target files are migrated — don't run remaining iterations.
- **Stuck detection:** If 3 consecutive iterations fail (tests break after migration), stop and report. The remaining files may need manual intervention. See the `stuck` skill.

## Step 1: Establish Baseline

```bash
git checkout -b self-migrate/$(date +%Y%m%d-%H%M%S)
{test_command} 2>&1 > /tmp/self-migrate-baseline.txt
```

All tests must pass before starting. Abort if baseline fails.

## Step 2: Run the Loop

For each iteration, spawn the `improver` agent:

```
Task: Migrate {target} toward {migration}.
Agent: improver
Context:
  - Iteration: {i} of {max}
  - Migration objective: {migration}
  - Iteration history: (cat /tmp/self-migrate-log.txt)
  - Remaining unmigrated files: {file_list}
  - Target: {target}
```

The improver agent:
1. Identifies the next file or component to migrate
2. Performs ONE migration step (one file or one closely-related group)
3. Updates imports/references as needed

Then the orchestrator:
```bash
RESULT=$({test_command} 2>&1)
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
  MIGRATED=$(git diff --name-only)
  echo "ITERATION $i: PASS — migrated $MIGRATED" >> /tmp/self-migrate-log.txt
  git add -A && git commit -m "self-migrate: iteration $i — ${migration}"
else
  echo "ITERATION $i: FAIL — reverting" >> /tmp/self-migrate-log.txt
  git checkout -- .
fi
```

Stop early if all target files are migrated.

## Step 3: Report

```
## Self-Migrate Report

**Target:** {target}
**Migration:** {migration}
**Iterations:** {completed} / {total}

### Migrated Files
- src/utils.js → src/utils.ts ✓
- src/api.js → src/api.ts ✓
- src/parser.js → (failed, reverted)

### Log
| # | Result | File(s) |
|---|--------|---------|
| 1 | PASS   | utils.js → utils.ts |
| 2 | PASS   | api.js → api.ts |
| 3 | FAIL   | parser.js — type errors |

### Next Steps
- Review: `git diff main...HEAD`
- Merge: `git checkout main && git merge self-migrate/{branch}`
```

## Presets

```
/self:migrate --target src/ --test "npm test" --migration "TypeScript strict mode"
/self:migrate --target app/ --test "pytest" --migration "Python 3.12 syntax (match statements, tomllib)"
/self:migrate --target components/ --test "npm test" --migration "React class components to hooks"
/self:migrate --target lib/ --test "go test ./..." --migration "Go 1.22 range-over-func"
```

## Rules

- Uses `improver` agent with worktree isolation
- Always branches first
- One file or closely-related group per iteration
- Tests must pass to keep — no exceptions
- Discard on any test failure
- Preserve existing behavior — migration, not refactoring
- Update imports and references in the same iteration
- Stop when all target files are migrated
