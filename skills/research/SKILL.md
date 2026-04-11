---
name: research
description: Fast, standard research workflow (clarify → decompose → parallel search → summarize → synthesize) for exploratory questions where the user wants a good answer quickly. Use when the user asks to research a topic, investigate options, compare libraries/tools/approaches, find the best solution to a technical question, scope out a space they don't know yet, or get a grounded starting point before deciding. Worth using when the user is gathering context or narrowing down options — not when they've already committed and need validation. Do NOT use for high-stakes decisions where a wrong answer is expensive (use deep-research), validating specific claims under scrutiny (use deep-research), or simple factual lookups that don't need web searches (answer directly). Default to this over deep-research unless the user explicitly signals stakes with phrases like "we're making a big bet", "need to be sure", "validate", "deeply investigate", or "correctness is critical".
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
