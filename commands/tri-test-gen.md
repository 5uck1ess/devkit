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

Check for plugins first (preferred), then fall back to CLI:

```bash
# Plugin detection (preferred — structured job management)
HAS_CODEX_PLUGIN=$(/codex:status >/dev/null 2>&1 && echo "yes" || echo "no")
HAS_GEMINI_PLUGIN=$(/gemini:status >/dev/null 2>&1 && echo "yes" || echo "no")

# CLI fallback detection
HAS_CODEX_CLI=$(command -v codex && echo "yes" || echo "no")
HAS_GEMINI_CLI=$(command -v gemini && echo "yes" || echo "no")
```

Prefer plugin over CLI.

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

## Concurrency & Budget

- **Concurrency limit:** Max 3 parallel agents.
- **Token budget:** ~400k tokens across all agents. Test generation can be verbose.
- **Rate limiting:** If API throttles, stagger agent launches.

## Step 4: Dispatch (Hybrid, Graceful Degradation)

**[PARALLEL]** Launch all available agents concurrently:

**CRITICAL:** All source code MUST be passed inline in each agent's prompt. Worktree-isolated agents cannot see the latest commits.

### Claude — always runs

Pass the source code inline — the agent runs in a worktree and cannot see recent changes.

```
Task: Generate tests for {target}.
Agent: test-writer
Input: {prompt}

{source_code — inlined here by the orchestrator}
```

<!-- The orchestrator MUST inline the source code here. The agent runs in a worktree and cannot fetch it. -->

### Codex — if available

```
/codex:rescue --effort high --background "{prompt} {source_code}"
```

Retrieve result with `/codex:result` when done. Omit `--model` to use the account default.

### Gemini — if available

**Plugin (preferred):**

```
/gemini:rescue --background "{prompt} {source_code}"
```

Retrieve result with `/gemini:result` when done. Omit `--model` to use the account default.

**CLI fallback (only if plugin not installed):**

```bash
if [ "$HAS_GEMINI_CLI" = "yes" ]; then
  gemini -p "{prompt} {source_code}" -y \
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
