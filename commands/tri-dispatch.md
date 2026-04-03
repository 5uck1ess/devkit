---
name: tri:dispatch
description: Dispatch a task to all three agents (Claude, Codex, Gemini) in parallel and compare results. Claude uses native background agent, others via plugin or CLI.
---

# Triple-Agent Dispatch

Send the same task to Claude, Codex, and Gemini in parallel. Compare outputs.

## When to use

- Comparing approaches to a problem
- Getting multiple implementation ideas
- Validating a solution across models

## Detect Available Agents

Check for plugins first (preferred), then fall back to CLI:

```bash
# Plugin detection (preferred — structured job management)
HAS_CODEX_PLUGIN=$(/codex:status >/dev/null 2>&1 && echo "yes" || echo "no")
HAS_GEMINI_PLUGIN=$(/gemini:status >/dev/null 2>&1 && echo "yes" || echo "no")

# CLI fallback detection
HAS_CODEX_CLI=$(command -v codex && echo "yes" || echo "no")
HAS_GEMINI_CLI=$(command -v gemini && echo "yes" || echo "no")
```

Run with whatever is available. Claude always runs. Prefer plugin over CLI.

## Concurrency & Budget

- **Concurrency limit:** Max 3 parallel agents.
- **Token budget:** ~300k tokens across all agents.
- **Rate limiting:** If API throttles, stagger agent launches with brief delays.

## Execution (Hybrid, Graceful Degradation)

**[PARALLEL]** Launch all available agents concurrently:

**CRITICAL:** Pass the full prompt and any relevant context inline to each agent. Worktree-isolated agents cannot see the latest commits or local state.

### Claude — always runs (native background agent)

Spawn the `researcher` agent as a background task with the full prompt inline:

```
Task: {user's task — full prompt inlined here}
Agent: researcher
```

<!-- The orchestrator MUST inline the complete task prompt. The agent runs in a worktree. -->

### Codex — if available

```
/codex:rescue --effort high --background "$PROMPT"
```

Retrieve result with `/codex:result` when done. Omit `--model` to use the account default.

### Gemini — if available

**Plugin (preferred):**

```
/gemini:rescue --background "$PROMPT"
```

Retrieve result with `/gemini:result` when done. Omit `--model` to use the account default.

**CLI fallback (only if plugin not installed):**

```bash
if [ "$HAS_GEMINI_CLI" = "yes" ]; then
  gemini -p "$PROMPT" -y \
    --output-format text > /tmp/tri-dispatch-gemini.txt 2>/dev/null &
fi

wait
```

## Output

```
## Triple Dispatch: {summary}

### Claude (researcher agent)
{agent result}

### Codex
{output}

### Gemini
{output}

### Analysis
- Where they agree: ...
- Where they differ: ...
- Recommended approach: ...
```

## Rules

- Claude always runs as native background agent for token efficiency
- Codex and Gemini are optional — run if installed, skip gracefully if not
- Report which agents participated (e.g., "2/3 agents" or "Claude only")
- If only Claude is available, still provide the full report format
- If one fails, report the others
- Clean up temp files after
- For file-modifying tasks, use Codex sandbox `full` instead of `read-only`
