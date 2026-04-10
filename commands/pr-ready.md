---
description: Full PR preparation pipeline — validate branch, DRY review, lint, test, security, changelog, create PR.
---

Ensure the devkit engine is installed, then run the workflow:

```bash
bash "$(find ~/.claude/plugins -path '*/devkit/scripts/ensure-engine.sh' 2>/dev/null | head -1)"
```

```bash
devkit workflow run pr-ready
```

If the engine cannot be installed (no network, no write access), tell the user: "The devkit engine binary is required for deterministic workflow execution. Install manually from https://github.com/5uck1ess/devkit/releases" Do NOT fall back to manual steps.
