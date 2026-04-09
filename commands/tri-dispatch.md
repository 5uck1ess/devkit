---
description: Dispatch a task to all available agents (Claude, Codex, Gemini) in parallel and compare results.
---

# Triple-Agent Dispatch

Send an arbitrary task to 2-3 AI agents in parallel and compare their results.

## Invoke

```
devkit workflow run tri-dispatch "{task_description}"
```

The YAML workflow uses model tiers (smart/general/fast) for parallelism. The fallback below uses external agents (Claude/Codex/Gemini) — the richer path when the engine is unavailable.

If `devkit workflow` is not available, follow this manually:

1. **Detect agents** — Check for Codex and Gemini availability. Claude always runs.
2. **Dispatch in parallel** — Launch all available agents with the same prompt. Claude uses native background agent; others use plugin or CLI.
3. **Collect results** — Wait for all agents to complete.
4. **Compare** — Present results side by side. Highlight consensus and divergence.

## Rules

- Claude always runs as native background agent
- Codex and Gemini are optional — skip gracefully
- Same prompt goes to all agents — no agent-specific modifications
- Report which agents participated
