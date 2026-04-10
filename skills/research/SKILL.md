---
name: research
description: Research workflow — use when asked to research a topic, investigate options, compare approaches, or find the best solution to a technical question. NOT for "deep research" or "validate" requests — those go to deep-research. For complex or high-stakes questions where correctness is critical, use deep-research instead.
---

# Research

Deterministic research workflow: clarify → decompose → parallel search → summarize → follow-up → synthesize.

## Invoke

Start the workflow via the devkit engine:

Use the `devkit_start` tool with workflow: "research" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.

## Rules

- Clarify before searching — don't waste searches on a vague question
- Decompose with explicit goals — no vague "broad search"
- Summarize immediately — never carry raw fetched content forward
- Track corroboration — every claim notes how many sources support it
- Cite sources — every finding links to where it came from
- Escalate when warranted — ask the user, never auto-escalate
