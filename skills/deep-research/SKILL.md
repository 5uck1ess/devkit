---
name: deep-research
description: Deep research with Analysis of Competing Hypotheses — use when asked to do deep research, deeply investigate, validate claims, or when correctness is critical and the user wants rigorous analysis with disconfirmation testing.
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
