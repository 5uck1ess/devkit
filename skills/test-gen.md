---
name: devkit:test-gen
description: Generate tests for code — use when asked to write tests, create a test suite, add test coverage, or generate unit/integration tests for a file or module.
---

# Test Generation

Generate tests for target code, run them, and iterate until they pass.

## Step 1: Analyze Target

Read the target files and detect:
- Language and test framework (jest, vitest, pytest, go test, cargo test, etc.)
- Existing test patterns and conventions
- Exports, public API, and key code paths
- Edge cases and error conditions

If test framework isn't obvious, check `package.json`, `pyproject.toml`, `go.mod`, `Cargo.toml`, etc.

## Step 2: Generate Tests

Spawn the `test-writer` agent:

```
Task: Generate comprehensive tests for {target}.
Agent: test-writer
Context:
  - Target: {target}
  - Framework: {detected_framework}
  - Existing test patterns: {patterns}
  - User instructions: {args}
```

The test-writer should:
1. Create test files matching project conventions
2. Cover happy paths, edge cases, error conditions
3. Use descriptive test names
4. Mock external dependencies only when necessary
5. Follow existing test patterns in the repo

## Step 3: Run and Fix

```bash
{test_command} 2>&1
```

If tests fail, send failures back to the test-writer agent to fix. Repeat up to 3 times.

## Step 4: Report

```
## Test Generation Report

**Target:** {target}
**Framework:** {framework}
**Tests created:** {count}
**Status:** all passing ✓ / {n} failing ✗

### Files Created
- tests/test_parser.py (12 tests)
- tests/test_api.py (8 tests)

### Coverage
- Lines: {line_coverage}%
- Branches: {branch_coverage}%

### Run
{test_command}
```

## Presets

```
/devkit:test-gen src/parser.ts
/devkit:test-gen lib/ --focus "error handling"
/devkit:test-gen src/api/ --unit-only
```

## Rules

- Match existing test conventions exactly (naming, location, style)
- Never modify source code — only create/edit test files
- Tests must actually run and pass
- Iterate up to 3 times to fix failures
- If a test can't be fixed, skip it and note in report
