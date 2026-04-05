---
name: deep-research
description: Deep research with Analysis of Competing Hypotheses — use when asked to do deep research, deeply investigate, validate claims, or when correctness is critical and the user wants rigorous analysis with disconfirmation testing.
---

# Deep Research Workflow

ACH-enhanced research: clarify → discover perspectives → decompose → search → summarize → generate hypotheses → disconfirm → build evidence matrix → self-critique → synthesize.

This is the rigorous path. It costs more tokens but produces higher-confidence results by actively trying to disprove answers rather than just confirming them.

## Step 1: Clarify

```
The user wants to deep-research: {input}

Before searching, clarify:
- What specifically are we trying to learn?
- What constraints matter (language, framework, scale)?
- What would a wrong answer cost? (helps calibrate rigor)
- Any sources they already know about?

Restate the research question precisely.
```

Use `AskUserQuestion`. Don't proceed until the question is sharp.

## Step 2: Discover Perspectives

```
Before generating search queries, survey the landscape.
Search for 2-3 overview/survey articles on the topic.
Fetch them with Jina Reader and extract:

- What are the major schools of thought or approaches?
- Who are the key voices (companies, researchers, communities)?
- What are the known debates or controversies?
- What perspectives might be underrepresented?

This grounds our search in real viewpoints, not LLM brainstorming.
Summarize immediately — do not carry raw content forward.
```

## Step 3: Decompose into Sub-Questions

```
Using the perspectives discovered in Step 2, break the research question
into 5-8 sub-questions, each with an explicit retrieval goal.

Format:
- Query: <search query>
  Goal: <what this search should find>
  Perspective: <which viewpoint this represents>

Rules:
- At least one query per major perspective/school of thought
- At least 2 queries explicitly seeking DISCONFIRMING evidence
  (e.g., "problems with X", "X failures", "why X doesn't work", "X vs Y disadvantages")
- No two queries should return the same results
- Include at least one query targeting recent sources (last 12 months)
```

## Step 4: Search (Parallel Fan-Out)

**[PARALLEL]** Launch sub-question searches concurrently using the `researcher` agent (max 3 agents per batch):

```
Task: Execute web search for a specific sub-question.
Agent: researcher
Input: Query + Goal + Perspective from decomposition step
Collect: titles, URLs, key snippets, publication date if available
```

Run in batches of 3. Collect all results before proceeding.

## Budget

- **Token budget:** ~400k tokens. Deep research is expensive but thorough.
- **Early exit:** Only if the question turns out to be trivial after Step 2.

## Step 5: Summarize and Extract Claims

```
For the 5-8 most promising URLs, fetch clean content:
  WebFetch https://r.jina.ai/{url} with header Accept: text/markdown

For each source, extract ATOMIC CLAIMS — individual factual assertions:
- Claim: <specific assertion>
  Source: <URL>
  Recency: <publication date or "unknown">

CRITICAL: Summarize each page into atomic claims immediately.
Do NOT carry raw page content forward.
Aim for 3-8 claims per source.
```

## Step 6: Generate Competing Hypotheses

```
Based on the claims gathered, generate 2-4 COMPETING HYPOTHESES
that could answer the research question.

Rules:
- Hypotheses must be mutually exclusive or meaningfully different
- Include at least one "contrarian" hypothesis that challenges the obvious answer
- Each hypothesis should be a clear, testable statement
- Don't include hypotheses with no supporting evidence at all

Format:
- H1: <statement>
- H2: <statement>
- H3: <statement>
```

## Step 7: Directed Disconfirmation

```
For EACH hypothesis, search specifically for evidence that would DISPROVE it.

This is the critical ACH step. You are not looking for confirmation.
You are trying to KILL each hypothesis.

For each hypothesis:
- Search: "<hypothesis claim> wrong" or "problems with <approach>" or "<alternative> better than <hypothesis>"
- Fetch and summarize the most relevant disconfirming source
- Extract any new claims that contradict the hypothesis

If you cannot find disconfirming evidence for a hypothesis after genuine effort,
note that — it's a signal of strength, not a gap to fill.
```

## Step 8: Build Evidence Matrix

```
Build a matrix: rows = evidence/claims, columns = hypotheses.

For each cell, mark:
- CC  (Strongly Consistent) — evidence directly supports this hypothesis
- C   (Consistent) — evidence is compatible with this hypothesis
- N   (Neutral) — evidence is irrelevant to this hypothesis
- I   (Inconsistent) — evidence contradicts this hypothesis
- II  (Strongly Inconsistent) — evidence directly disproves this hypothesis

| Evidence | H1 | H2 | H3 |
|----------|----|----|-----|
| Claim 1 [source] | CC | I | N |
| Claim 2 [source] | N | CC | C |
| Claim 3 [source] | II | C | CC |
| ...      |    |    |     |

Then score each hypothesis:
- Count inconsistencies (I + II). MORE inconsistencies = WEAKER hypothesis.
- The surviving hypothesis is the one with the FEWEST inconsistencies,
  NOT the most consistencies. This is the key ACH insight.
```

## Step 9: Sensitivity Check

```
For the leading hypothesis, identify:
1. Which single piece of evidence, if wrong, would change the conclusion?
2. Are there any "linchpin" claims supported by only one source?
3. What new evidence would cause you to switch to a different hypothesis?

This tells us how fragile or robust the conclusion is.
```

## Step 10: Self-Critique

```
Before writing the final synthesis, review your own work:

1. Did I genuinely try to disprove each hypothesis, or did I softball the disconfirmation?
2. Are there perspectives I missed entirely?
3. Am I over-weighting recency or authority of any single source?
4. Would someone with the opposite view find my analysis fair?
5. Are there claims I'm treating as confirmed that are actually uncorroborated?

If you find gaps, do ONE more targeted search round to fill them.
Otherwise, proceed to synthesis.
```

## Step 11: Synthesize

```
## Deep Research: {question}

### Direct Answer
{clear answer with confidence level: HIGH / MEDIUM / LOW}

### Competing Hypotheses Evaluated

#### H1: {statement} — [REJECTED / SURVIVING / INCONCLUSIVE]
- Supporting evidence: {claims with sources}
- Disconfirming evidence: {claims with sources}
- Inconsistency count: X

#### H2: {statement} — [REJECTED / SURVIVING / INCONCLUSIVE]
- Supporting evidence: {claims with sources}
- Disconfirming evidence: {claims with sources}
- Inconsistency count: X

(repeat for each hypothesis)

### Evidence Matrix Summary
{the matrix from Step 8, or a prose summary if >10 rows}

### Key Findings
{top findings with corroboration status}
- CONFIRMED: {claim} — [source1], [source2]
- CONTESTED: {claim} — [source1] says X, [source2] says Y
- UNCORROBORATED: {claim} — [source] (single source)

### Sensitivity Analysis
- Linchpin evidence: {what single fact, if wrong, changes the answer}
- Confidence fragility: {HIGH = robust across evidence, LOW = depends on 1-2 sources}

### What Would Change This Conclusion
{specific evidence or events that would flip the answer}

### Recommendation
{what to do and why, with explicit confidence calibration}
```

## Rules

- Perspectives first — discover real viewpoints before generating queries
- Disconfirm, don't confirm — the goal is to DISPROVE hypotheses, not prove them
- Atomic claims — extract specific assertions, not vague summaries
- Summarize immediately — never carry raw fetched content into next steps
- Evidence matrix is mandatory — no skipping the structured comparison
- Sensitivity check is mandatory — know how fragile your conclusion is
- Self-critique before output — catch your own biases
- Cite everything — every claim links to its source
- Confidence calibration — HIGH/MEDIUM/LOW based on evidence robustness, not gut feel
- Be honest about uncertainty — "I don't know" with good reasoning beats a confident wrong answer
