---
description: Triple-agent debugging — independent root-cause hypotheses from Claude, Codex, and Gemini, then consensus fix.
---

Ensure the devkit engine is installed, then run the workflow:

```bash
command -v devkit >/dev/null 2>&1 || bash "$(dirname "$(find ~/.claude/plugins -path '*/devkit/scripts/install-engine.sh' 2>/dev/null | head -1)")/install-engine.sh"
```

```bash
devkit workflow run tri-debug
```

If the engine cannot be installed (no network, no write access), tell the user: "The devkit engine binary is required for deterministic workflow execution. Run `bash scripts/install-engine.sh` manually." Do NOT fall back to manual steps.
