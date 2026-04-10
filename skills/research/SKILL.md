---
name: research
description: Research workflow — use when asked to research a topic, investigate options, compare approaches, or find the best solution to a technical question. NOT for "deep research" or "validate" requests — those go to deep-research. For complex or high-stakes questions where correctness is critical, use deep-research instead.
---

# Research

Deterministic research workflow: clarify → decompose → parallel search → summarize → follow-up → synthesize.

## Invoke

Ensure the devkit engine is installed, then run the workflow:

```bash
bash "$(find ~/.claude/plugins -path '*/devkit/scripts/ensure-engine.sh' 2>/dev/null | head -1)"
```

```bash
devkit workflow run research "{input}"
```

The YAML workflow (`workflows/research.yml`) enforces the step sequence deterministically. Claude handles thinking within each step; the engine owns the order.

If the engine cannot be installed, tell the user: "The devkit engine binary is required for deterministic workflow execution. Install manually from https://github.com/5uck1ess/devkit/releases" Do NOT fall back to manual steps — the engine is required for determinism.

## Rules

- Clarify before searching — don't waste searches on a vague question
- Decompose with explicit goals — no vague "broad search"
- Summarize immediately — never carry raw fetched content forward
- Track corroboration — every claim notes how many sources support it
- Cite sources — every finding links to where it came from
- Escalate when warranted — ask the user, never auto-escalate
