---
name: tri:test-gen
description: Multi-agent test generation — each available agent generates tests independently, then merge for maximum coverage.
---

# Triple-Agent Test Generation

Generate tests from all available agents in parallel, then merge the best tests into a comprehensive suite.

## Step 1: Analyze Target

Read the target files and detect:
- Language, test framework, existing test patterns
- Public API surface and code paths to test

## Step 2: Detect Available Agents

```bash
HAS_CODEX=$(command -v codex && echo "yes" || echo "no")
HAS_GEMINI=$(command -v gemini && echo "yes" || echo "no")
```

## Step 3: Build the Prompt

```
Generate a comprehensive test suite for the following code.
- Cover happy paths, edge cases, error conditions, and boundary values.
- Use {framework} conventions.
- Match existing test patterns in the repo.
- Write tests that actually run — no placeholder assertions.

Target: {target_files}
Code: {source_code}
```

## Step 4: Dispatch (Hybrid, Graceful Degradation)

### Claude — always runs

```
Task: Generate tests for {target} using the test-writer agent.
Agent: test-writer
Input: {prompt} + {source_code}
```

### Codex — if available

```bash
if [ "$HAS_CODEX" = "yes" ]; then
  codex exec -m gpt-5.4 \
    --sandbox read-only \
    --full-auto \
    --skip-git-repo-check \
    --dangerously-bypass-approvals-and-sandbox \
    "{prompt}" > /tmp/tri-test-codex.txt 2>/dev/null &
fi
```

### Gemini — if available

```bash
if [ "$HAS_GEMINI" = "yes" ]; then
  gemini -p "{prompt}" -m gemini-3.1-pro -y \
    --output-format text > /tmp/tri-test-gemini.txt 2>/dev/null &
fi

wait
```

## Step 5: Merge & Deduplicate

Analyze test suites from all agents:
1. Identify unique test cases across all suites
2. Remove duplicates (same assertion, different wording)
3. Keep the best implementation of each test (clearest, most thorough)
4. Combine into one unified test file

## Step 6: Run & Fix

```bash
{test_command} 2>&1
```

If tests fail, fix them (up to 3 attempts).

## Step 7: Report

```
## Triple Test Generation: {target}

### Agents Used: {count}/3

### Test Contributions
| Agent | Tests Generated | Unique Tests Kept |
|-------|----------------|-------------------|
| Claude | 12 | 8 |
| Codex | 10 | 4 |
| Gemini | 8 | 3 |

### Final Suite
- **Total tests:** 15
- **All passing:** ✓
- **Coverage:** {coverage}%

### Files Created
- {test_file_1}
- {test_file_2}
```

## Rules

- Claude always runs — others are optional
- Merge the best from each, don't just concatenate
- Final suite must pass before reporting success
- Match existing project test conventions
- Clean up temp files after
