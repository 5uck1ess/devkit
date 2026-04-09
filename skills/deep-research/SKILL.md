---
name: deep-research
description: Deep research with Analysis of Competing Hypotheses — use when asked to do deep research, deeply investigate, validate claims, or when correctness is critical and the user wants rigorous analysis with disconfirmation testing.
---

# Deep Research

ACH-enhanced deterministic research: clarify → perspectives → decompose → search → extract claims → hypotheses → disconfirm → evidence matrix → self-critique → synthesize.

Costs more tokens (~400k budget) but produces higher-confidence results by actively trying to disprove answers.

## Invoke

Run the workflow via the devkit engine:

```
devkit workflow run deep-research "{input}"
```

The YAML workflow (`workflows/deep-research.yml`) enforces the full ACH sequence deterministically. Claude handles thinking within each step; the engine owns the order.

## Fallback (no engine)

If `devkit workflow` is not available, follow these steps manually:

1. **Clarify** — Use `AskUserQuestion` to sharpen the question; ask what a wrong answer would cost
2. **Discover perspectives** — Search for 2-3 overview articles; extract schools of thought, key voices, debates; summarize immediately
3. **Decompose** — 5-8 sub-questions with retrieval goals and perspective labels; at least 2 must seek disconfirming evidence
4. **Search** — Parallel fan-out using `researcher` agent (max 3 per batch); collect titles, URLs, snippets, dates
5. **Extract claims** — Fetch top 5-8 URLs via Jina Reader; extract atomic claims (3-8 per source); do NOT carry raw content forward
6. **Hypotheses** — Generate 2-4 competing hypotheses; include at least one contrarian; each must be testable
7. **Directed disconfirmation** — For EACH hypothesis, search for evidence that DISPROVES it; this is the critical ACH step
8. **Evidence matrix** — Rows = claims, columns = hypotheses; mark CC/C/N/I/II; score by FEWEST inconsistencies (not most consistencies)
9. **Sensitivity check** — Identify linchpin evidence; what single fact, if wrong, changes the conclusion?
10. **Self-critique** — Did you genuinely try to disprove? Missed perspectives? Over-weighting a source? One more search round if gaps found (loop max 2)
11. **Synthesize** — Direct answer with confidence (HIGH/MEDIUM/LOW) → hypotheses evaluated → evidence matrix → key findings → sensitivity analysis → recommendation

## Rules

- Perspectives first — ground queries in real viewpoints, not LLM brainstorming
- Disconfirm > confirm — try to KILL hypotheses, not prove them
- Summarize immediately — never carry raw fetched content forward
- Evidence matrix is mandatory — no skipping the structured comparison
- Sensitivity check is mandatory — know how fragile your conclusion is
- Self-critique before output — catch your own biases
- Cite everything — every claim links to its source
- Be honest about uncertainty — "I don't know" with good reasoning beats a confident wrong answer
