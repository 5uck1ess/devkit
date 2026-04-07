---
description: ACH-enhanced deep research — perspective discovery, competing hypotheses, directed disconfirmation, evidence matrix, sensitivity analysis.
---

# Deep Research

Rigorous research for complex or high-stakes questions where correctness matters. Uses Analysis of Competing Hypotheses (ACH) to actively disprove answers rather than just confirming them.

Use regular `/devkit:research` for quick lookups. Use this when:
- The answer has real consequences (architecture decisions, tool selection, security)
- Multiple conflicting sources exist
- You need confidence calibration, not just an answer
- The user says "deep research", "validate", "make sure this is right"

## Step 0: Harness Detection

```bash
if command -v devkit >/dev/null 2>&1; then
  echo "Go harness detected — delegating to devkit workflow deep-research."
  devkit workflow deep-research "{input}"
  exit 0
fi
```

## Step 1: Clarify

```
The user wants to deep-research: {input}

Use AskUserQuestion to clarify:
- What specifically are we trying to learn?
- What constraints matter?
- What would a wrong answer cost?
- Any sources they already know about?

Restate the research question precisely.
```

## Step 2: Discover Perspectives

Search for 2-3 overview articles on the topic. Fetch with Jina Reader (`WebFetch https://r.jina.ai/{url}`). Extract major schools of thought, key voices, known debates. Summarize immediately — do not carry raw content forward.

## Step 3: Decompose into Sub-Questions

Break the question into 5-8 sub-questions with explicit retrieval goals and perspective labels. At least 2 queries must seek **disconfirming** evidence.

## Step 4: Search (Parallel Fan-Out)

**[PARALLEL]** Launch searches concurrently using the `researcher` agent (max 3 per batch):

```
Task: Execute web search for sub-question.
Agent: researcher
Input: Query + Goal + Perspective
```

## Budget

- **Token budget:** ~400k tokens.
- **Early exit:** Only if the question turns out trivial after perspective discovery.

## Step 5: Summarize and Extract Claims

Fetch top 5-8 URLs with Jina Reader. Extract atomic claims (subject-predicate-object). Summarize immediately. Aim for 3-8 claims per source.

## Step 6: Generate Competing Hypotheses

Generate 2-4 mutually exclusive hypotheses. Include at least one contrarian hypothesis.

## Step 7: Directed Disconfirmation

For EACH hypothesis, search specifically for evidence that would DISPROVE it. This is the critical ACH step — you're trying to kill each hypothesis, not confirm it.

## Step 7.5: Adversarial Debate (optional — use when hypotheses are close or stakes are high)

When two or more hypotheses survive disconfirmation with similar evidence profiles, run an adversarial refinement cycle to stress-test them:

1. **Advocate** — For each surviving hypothesis, write the strongest possible case. Cite specific evidence. Assume this hypothesis is correct and explain away contradictions.

2. **Critic** — For each advocacy, write a targeted attack. Find the weakest link in the argument. Identify what the advocate glossed over or explained away too easily. Name the single observation that would kill this hypothesis.

3. **Synthesize** — Given the advocacy and critique for all hypotheses, ask: Is there a composite hypothesis that accounts for more evidence than any individual one? Can the strongest elements be combined?

4. **Blind Judge** — Present the surviving candidates (original + any composite) with randomized labels (Candidate A, B, C — not in hypothesis order). Evaluate on:
   - Fewest unexplained observations
   - Most falsifiable (can be tested)
   - Least reliance on coincidence
   Pick a winner. If no clear winner, note the deadlock and carry both forward with explicit uncertainty.

Skip this step if: one hypothesis clearly dominates after Step 7, or the question is low-stakes.

## Step 8: Build Evidence Matrix

```
| Evidence | H1 | H2 | H3 |
|----------|----|----|-----|
| Claim [source] | CC | I | N |

CC=Strongly Consistent, C=Consistent, N=Neutral, I=Inconsistent, II=Strongly Inconsistent
```

Score by FEWEST inconsistencies (not most consistencies).

## Step 9: Sensitivity Check

Identify linchpin evidence — what single fact, if wrong, would change the conclusion?

## Step 10: Self-Critique

Review for: genuine disconfirmation effort, missed perspectives, source over-weighting, fairness to opposing views. One more search round if gaps found.

## Step 11: Synthesize

```
## Deep Research: {question}

### Direct Answer
{answer with confidence: HIGH / MEDIUM / LOW}

### Competing Hypotheses
{for each: statement, supporting evidence, disconfirming evidence, status}

### Evidence Matrix Summary
{matrix or prose summary}

### Key Findings
{CONFIRMED / CONTESTED / UNCORROBORATED claims with sources}

### Sensitivity Analysis
{linchpin evidence, confidence fragility}

### What Would Change This Conclusion
{specific evidence that would flip the answer}

### Recommendation
{what to do, why, with calibrated confidence}
```

## Rules

- Perspectives first — ground queries in real viewpoints
- Disconfirm > confirm — try to KILL hypotheses
- Summarize immediately — no raw content carried forward
- Evidence matrix is mandatory
- Sensitivity check is mandatory
- Self-critique before output
- Cite everything
- Be honest about uncertainty
- Use adversarial debate when hypotheses are close — don't just pick the first plausible one
- Blind judge evaluations use randomized labels to prevent ordering bias
