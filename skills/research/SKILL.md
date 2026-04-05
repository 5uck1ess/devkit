---
name: research
description: Research workflow — use when asked to research a topic, do a deep dive, investigate options, compare approaches, or find the best solution to a technical question. For complex or high-stakes questions where correctness is critical, use deep-research instead.
---

# Research Workflow

Complete research lifecycle: clarify → decompose → search → summarize → corroborate → synthesize.

## Step 1: Clarify

```
The user wants to research: {input}

Before searching, clarify:
- What specifically are we trying to learn?
- What constraints matter (language, framework, scale)?
- Any sources they already know about?

Restate the research question precisely.
```

Use `AskUserQuestion` to ask these questions explicitly. Don't proceed until the question is clear.

## Step 2: Decompose into Sub-Questions

```
Break the research question into 3-5 sub-questions, each with an explicit retrieval goal.

Format:
- Query: <search query>
  Goal: <what this search should find — e.g., "official docs on X", "benchmarks comparing X vs Y", "known problems with X">

Rules:
- Each query must target a DIFFERENT angle (definition, evidence, criticism, alternatives, recency)
- No two queries should return the same results
- Include at least one query seeking disconfirming evidence or criticism
```

## Step 3: Search (Parallel Fan-Out)

**[PARALLEL]** Launch all sub-question searches concurrently using the `researcher` agent (max 3 agents):

```
Task: Execute web search for a specific sub-question.
Agent: researcher
Input: Query + Goal from decomposition step
Collect: titles, URLs, key snippets
```

All searches run in parallel. Collect results before proceeding.

## Budget

- **Token budget:** ~200k tokens.
- **Early exit:** If the first search pass clearly answers the question, skip follow-up.

## Step 4: Summarize Sources

```
For the 3-5 most promising URLs, fetch clean content:
  WebFetch https://r.jina.ai/{url} with header Accept: text/markdown

CRITICAL: Immediately summarize each page into 3-5 key claims with source attribution.
Do NOT carry raw page content forward — summarize first, then discard the raw text.
This prevents context overflow on large pages.

If Jina fails for a URL, fall back to raw WebFetch on the original URL.
```

## Step 5: Corroborate

```
For each key claim from Step 4:
- Count how many independent sources support it
- Flag any claims supported by only 1 source as "uncorroborated"
- Flag any claims where sources contradict each other

Mark claims as:
- CONFIRMED (2+ independent sources agree)
- UNCORROBORATED (only 1 source)
- CONTESTED (sources disagree)
```

## Step 6: Follow-Up

```
Review claims marked UNCORROBORATED or CONTESTED.
For each, run one targeted search to either confirm or resolve the conflict.
Use Jina Reader for any new sources, summarize immediately.

If all key claims are confirmed or the question is answered, say "RESEARCH_COMPLETE".
```

Loop up to 2 times.

## Step 7: Synthesize

```
## Research: {question}

### Direct Answer
{clear answer to the research question}

### Key Findings
{findings with source URLs and corroboration status}
- CONFIRMED: {claim} — [source1], [source2]
- UNCORROBORATED: {claim} — [source] (single source only)
- CONTESTED: {claim} — [source1] says X, [source2] says Y

### Tradeoffs
{comparison between approaches}

### Open Questions
{what couldn't be resolved, any CONTESTED claims without resolution}

### Recommendation
{what to do and why, noting confidence level}
```

## Rules

- Clarify before searching — don't waste searches on a vague question
- Decompose into sub-questions with explicit goals — no vague "broad search"
- Summarize immediately — never carry raw fetched content forward
- Track corroboration — every claim should note how many sources support it
- Surface contradictions — if sources disagree, say so explicitly
- Follow up on gaps — loop if key claims are uncorroborated
- Cite sources — every finding links to where it came from
- Recommend — don't just dump information, give a clear recommendation with confidence level
- For complex/high-stakes questions, suggest the user run `/devkit:deep-research` instead
