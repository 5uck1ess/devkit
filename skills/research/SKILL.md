---
name: research
description: Research workflow — use when asked to research a topic, investigate options, compare approaches, or find the best solution to a technical question. NOT for "deep research" or "validate" requests — those go to deep-research. For complex or high-stakes questions where correctness is critical, use deep-research instead.
---

# Research

Deterministic research workflow: clarify → decompose → parallel search → corroborate → synthesize.

## Invoke

Run the workflow via the devkit engine:

```
devkit workflow run research "{input}"
```

The YAML workflow (`workflows/research.yml`) enforces the step sequence deterministically. Claude handles thinking within each step; the engine owns the order.

## Fallback (no engine)

If `devkit workflow` is not available, follow these steps manually:

1. **Clarify** — Use `AskUserQuestion` to sharpen the question before searching
2. **Decompose** — Break into 3-5 sub-questions with explicit retrieval goals; include at least one disconfirming query
3. **Search** — Launch searches in parallel using the `researcher` agent (max 3); collect titles, URLs, snippets
4. **Summarize** — Fetch top URLs via Jina Reader (`WebFetch https://r.jina.ai/{url}`); extract 3-5 claims per source immediately; do NOT carry raw content forward
5. **Corroborate** — Mark each claim CONFIRMED (2+ sources) / UNCORROBORATED (1 source) / CONTESTED (sources disagree)
6. **Escalation check** — If 3+ CONTESTED claims or high-stakes domain, ask the user to upgrade to `/devkit:deep-research`. Never auto-escalate.
7. **Follow-up** — For UNCORROBORATED/CONTESTED claims, run one targeted search to resolve (loop max 2)
8. **Synthesize** — Direct answer → key findings with corroboration status → tradeoffs → open questions → recommendation with confidence level

## Rules

- Clarify before searching — don't waste searches on a vague question
- Summarize immediately — never carry raw fetched content forward
- Track corroboration — every claim notes how many sources support it
- Cite sources — every finding links to where it came from
- Escalate when warranted — ask the user, never auto-escalate
