---
description: Triple-agent code review — dispatches to Claude, Codex, and Gemini in parallel, consolidates findings.
---

# Triple-Agent Review

Dispatch the same code review to 2-3 AI agents in parallel and consolidate results by consensus.

## Invoke

```
devkit workflow run tri-review "{prompt or default}"
```

If `devkit workflow` is not available, follow this manually:

1. **Gather context** — Capture diff via `git diff main...HEAD` (fall back to `HEAD~1..HEAD` or `--cached`). Write to temp file. Warn if >5000 lines.
2. **Build prompt** — Use custom prompt if provided, otherwise default: bugs, security, DRY violations, unnecessary complexity, missing edge cases.
3. **Detect agents** — Check for Codex plugin/CLI and Gemini plugin/CLI. Claude always runs.
4. **Dispatch in parallel** — Launch all available agents concurrently. Pass diff inline in each prompt — worktree-isolated agents can't see latest commits, so the provided diff is the ONLY source of truth. Do NOT instruct agents to read files from the worktree to verify changes. Claude uses native background agent; Codex/Gemini use plugin (preferred) or CLI fallback.
5. **Consolidate** — Consensus findings (2+ agents) ranked higher. Unique findings listed per agent.

## Rules

- Claude always runs as native background agent (not `claude -p`)
- Codex and Gemini are optional — skip gracefully if not installed
- Diff MUST be passed inline — agents can't fetch it themselves
- Report which agents participated
- Sort by severity, then consensus count
