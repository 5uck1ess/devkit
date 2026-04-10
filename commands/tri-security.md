---
description: Triple-agent security audit — independent security reviews from Claude, Codex, and Gemini, consolidated with severity ranking.
---

Ensure the devkit engine is installed, then run the workflow:

```bash
bash "$(find ~/.claude/plugins -path '*/devkit/scripts/ensure-engine.sh' 2>/dev/null | head -1)"
```

```bash
devkit workflow run tri-security
```

If the engine cannot be installed (no network, no write access), tell the user: "The devkit engine binary is required for deterministic workflow execution. Install manually from https://github.com/5uck1ess/devkit/releases" Do NOT fall back to manual steps.
