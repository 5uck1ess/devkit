---
description: Triple-agent security audit — independent security reviews from Claude, Codex, and Gemini, consolidated with severity ranking.
---

# Triple-Agent Security Audit

Dispatch a security audit to 2-3 AI agents in parallel and consolidate with severity-ranked findings.

## Invoke

```
devkit workflow run tri-security "{scope or default}"
```

The YAML workflow dispatches by security domain (injection/auth/config) across model tiers. The fallback below uses external agents (Claude/Codex/Gemini) — the richer path when the engine is unavailable.

If `devkit workflow` is not available, follow this manually:

1. **Gather scope** — Determine audit scope: full repo, specific directory, or changed files only. Capture relevant code.
2. **Detect agents** — Check for Codex and Gemini availability. Claude always runs.
3. **Build prompt** — Include OWASP top 10 categories, language-specific patterns, auth/authz checks, input validation, secrets detection, dependency vulnerabilities
4. **Dispatch in parallel** — Launch all available agents concurrently with full scope context
5. **Consolidate** — Rank by severity (critical/high/medium/low), then by consensus count (2+ agents = high confidence)

## Rules

- Claude always runs — others are optional
- Consensus findings ranked higher than single-agent findings
- Sort by severity, then by consensus count
- Include specific file:line references
- Provide actionable fix for each finding
