---
description: Triple-agent debugging — independent root-cause hypotheses from Claude, Codex, and Gemini, then consensus fix.
---

# Triple-Agent Debug

Dispatch a bug report to 2-3 AI agents in parallel, get independent root-cause hypotheses, and consolidate a fix.

## Invoke

```
devkit workflow run tri-debug "{bug_description}"
```

The YAML workflow uses model tiers (smart/general/fast) for parallelism. The fallback below uses external agents (Claude/Codex/Gemini) — the richer path when the engine is unavailable.

If `devkit workflow` is not available, follow this manually:

1. **Gather context** — Collect error messages, stack traces, reproduction steps, and relevant code
2. **Detect agents** — Check for Codex and Gemini availability. Claude always runs.
3. **Build prompt** — Include: error output, relevant source code, recent changes (`git log -5`), and reproduction steps
4. **Dispatch in parallel** — Launch all available agents concurrently with the full context
5. **Consolidate** — Consensus diagnosis (2+ agents agree) → high confidence. Unique hypotheses → worth investigating.

## Rules

- Claude always runs as native background agent
- Codex and Gemini are optional — skip gracefully if not installed
- Each agent works independently — don't share findings between agents
- Report consensus vs unique diagnoses
- Include specific file:line references and suggested fixes
