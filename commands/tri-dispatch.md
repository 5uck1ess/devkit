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

## Execution (Hybrid)

### Claude — native background agent

Spawn the `researcher` agent as a background task:

```
Task: {user's task}
Agent: researcher
```

### Codex — CLI

```bash
codex exec -m gpt-5.4 \
  --sandbox read-only \
  --full-auto \
  --skip-git-repo-check \
  --dangerously-bypass-approvals-and-sandbox \
  "$PROMPT" > /tmp/tri-dispatch-codex.txt 2>/dev/null &
```

### Gemini — CLI

```bash
gemini -p "$PROMPT" -m gemini-3.1-pro -y \
  --output-format text > /tmp/tri-dispatch-gemini.txt 2>/dev/null &

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

- Claude uses native background agent for token efficiency
- Codex and Gemini run as CLI background processes
- If one fails, report the others
- Clean up temp files after
- For file-modifying tasks, use Codex sandbox `full` instead of `read-only`
