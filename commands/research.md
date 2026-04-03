---
name: devkit:research
description: Deep research workflow — clarify question, parallel web search, analyze sources, synthesize findings.
---

# Research Workflow

Complete research lifecycle: clarify → search (broad + specific + alternatives) → analyze → follow-up → synthesize.

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

## Step 2: Search (Three Perspectives)

Run three search strategies using the `researcher` agent.

**[PARALLEL]** Launch all three searches concurrently (max 3 agents):

### Broad Search

```
Task: Web search with broad queries to find general sources.
Agent: researcher
Input: Research question + broad search terms
Collect: titles, URLs, key snippets
```

### Specific Search

```
Task: Web search with specific technical queries targeting documentation, benchmarks, or comparisons.
Agent: researcher
Input: Research question + specific/technical search terms (different from broad)
```

### Alternative Viewpoints

```
Task: Web search for contrarian or alternative viewpoints.
Agent: researcher
Input: "problems with", "alternatives to", "vs" queries related to the topic
```

All three run in parallel. Collect results before proceeding to analysis.

## Budget

- **Token budget:** ~300k tokens. Web search and reading can be expensive.
- **Early exit:** If the first search pass answers the question clearly, skip follow-up loops.

## Step 3: Analyze

```
Read the 3-5 most promising URLs in depth.

For each URL, fetch clean Markdown using Jina Reader:
  WebFetch https://r.jina.ai/{url} with header Accept: text/markdown

This strips boilerplate (nav, ads, footers) and returns clean article content.
If Jina fails for a URL, fall back to raw WebFetch on the original URL.

Extract key findings, compare approaches, note contradictions.

Sources from:
- Broad search results
- Specific search results
- Alternative viewpoints
```

## Step 4: Follow-Up

```
Are there gaps in the research? If the initial sources were weak
or missing key perspectives, search and read more to fill gaps.
Use Jina Reader (WebFetch https://r.jina.ai/{url}) for any new sources.
```

Loop up to 3 times until research is thorough enough.

## Step 5: Synthesize

```
## Research: {question}

### Direct Answer
{clear answer to the research question}

### Key Findings
{findings with source URLs}

### Tradeoffs
{comparison between approaches}

### Open Questions
{what couldn't be resolved}

### Recommendation
{what to do and why}
```

## Rules

- Clarify before searching — don't waste searches on a vague question
- Multiple search strategies — broad, specific, and contrarian
- Read actual sources — don't summarize from snippets alone
- Note contradictions — if sources disagree, say so
- Follow up on gaps — loop if the initial pass missed something
- Cite sources — every finding should link to where it came from
- Recommend — don't just dump information, give a clear recommendation
