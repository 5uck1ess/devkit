---
name: deep-research
description: Deep research with Analysis of Competing Hypotheses — use when asked to do deep research, deeply investigate, validate claims, or when correctness is critical and the user wants rigorous analysis with disconfirmation testing.
---

# Deep Research

ACH-enhanced deterministic research: clarify → perspectives → decompose → parallel search → extract claims → hypotheses → disconfirm → evidence matrix (with sensitivity check) → self-critique → synthesize.

Costs more tokens (~400k budget) but produces higher-confidence results by actively trying to disprove answers.

## Invoke

Ensure the devkit engine is installed, then run the workflow:

```bash
bash "$(find ~/.claude/plugins -path '*/devkit/scripts/ensure-engine.sh' 2>/dev/null | head -1)"
```

```bash
devkit workflow run deep-research "{input}"
```

The YAML workflow (`workflows/deep-research.yml`) enforces the full ACH sequence deterministically. Claude handles thinking within each step; the engine owns the order.

If the engine cannot be installed, tell the user: "The devkit engine binary is required for deterministic workflow execution. Install manually from https://github.com/5uck1ess/devkit/releases" Do NOT fall back to manual steps — the engine is required for determinism.

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
