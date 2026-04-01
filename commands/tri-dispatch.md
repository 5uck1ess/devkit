---
name: tri:dispatch
description: Dispatch a task to all three agents (Claude, Codex, Gemini) in parallel and compare results. Claude uses native background agent, others via CLI.
---

# Triple-Agent Dispatch

Send the same task to Claude, Codex, and Gemini in parallel. Compare outputs.

## When to use

- Comparing approaches to a problem
- Getting multiple implementation ideas
- Validating a solution across models

## Detect Available Agents

```bash
HAS_CODEX=$(command -v codex && echo "yes" || echo "no")
HAS_GEMINI=$(command -v gemini && echo "yes" || echo "no")
```

Run with whatever is available. Claude always runs.

## Execution (Hybrid, Graceful Degradation)

### Claude — always runs (native background agent)

Spawn the `researcher` agent as a background task:

```
Task: {user's task}
Agent: researcher
```

### Codex — if available

```
/codex:rescue --model gpt-5.4 --effort high --background "$PROMPT"
```

Retrieve result with `/codex:result` when done.

### Gemini — if available

```bash
if [ "$HAS_GEMINI" = "yes" ]; then
  gemini -p "$PROMPT" -m gemini-3.1-pro -y \
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
