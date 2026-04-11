---
name: deep-research
description: High-stakes research with Analysis of Competing Hypotheses, disconfirmation testing, evidence matrices, and sensitivity checks (~400k token budget) — use when the user needs to be SURE, not just well-informed. Trigger phrases include "deep research", "deeply investigate", "validate that X", "prove/disprove", "we're making a big decision", "correctness is critical", "I need to trust this answer", or "actively try to disprove this". Worth the extra cost when — the wrong answer has real consequences (architecture commitment, vendor choice, security claim, compliance question, public-facing claim), the user has already seen shallow answers and doesn't trust them, they explicitly ask for rigor or disconfirmation, or they're defending a position to stakeholders. Do NOT use for exploratory "what's out there" questions (use research), routine library comparisons where a single good answer suffices (use research), or quick factual lookups. This is the bias-catching, evidence-calibrating tier — overkill for casual exploration but essential when you can't afford to be wrong.
---

# Deep Research

ACH-enhanced deterministic research: clarify → perspectives → decompose → parallel search → extract claims → hypotheses → disconfirm → evidence matrix (with sensitivity check) → self-critique → synthesize.

Costs more tokens (~400k budget) but produces higher-confidence results by actively trying to disprove answers.

## Invoke

Start the workflow via the devkit engine:

Use the `devkit_start` tool with workflow: "deep-research" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.

## Rules

- Perspectives first — ground queries in real viewpoints, not LLM brainstorming
- Disconfirm > confirm — try to KILL hypotheses, not prove them
- Summarize immediately — never carry raw fetched content forward
- Evidence matrix is mandatory — no skipping the structured comparison
- Sensitivity check is mandatory — know how fragile your conclusion is
- Self-critique before output — catch your own biases
- Cite everything — every claim links to its source
- Confidence calibration — HIGH/MEDIUM/LOW based on evidence robustness, not gut feel
- Be honest about uncertainty — "I don't know" with good reasoning beats a confident wrong answer
