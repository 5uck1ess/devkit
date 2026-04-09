---
description: ACH-enhanced deep research — delegates to deterministic YAML workflow.
---

# Deep Research

Rigorous research for complex or high-stakes questions where correctness matters. Uses Analysis of Competing Hypotheses (ACH) to actively disprove answers rather than just confirming them.

Use regular `/devkit:research` for quick lookups. Use this when:
- The answer has real consequences (architecture decisions, tool selection, security)
- Multiple conflicting sources exist
- You need confidence calibration, not just an answer
- The user says "deep research", "validate", "make sure this is right"

## Invoke

```
devkit workflow run deep-research "{input}"
```

If `devkit workflow` is not available, activate the `/devkit:deep-research` skill which contains a condensed fallback for manual execution.

The YAML workflow (`workflows/deep-research.yml`) enforces the full ACH sequence deterministically:
clarify → perspectives → decompose → parallel search → extract claims → hypotheses → disconfirm → evidence matrix → self-critique → synthesize.
