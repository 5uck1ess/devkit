---
description: Triple-agent code review — dispatches to Claude, Codex, and Gemini in parallel, consolidates findings.
---

Ensure the devkit engine is installed, then run the workflow:

```bash
ENSURE="$(find ~/.claude/plugins ${APPDATA:+$APPDATA/.claude/plugins} ${LOCALAPPDATA:+$LOCALAPPDATA/.claude/plugins} -path '*/devkit/scripts/ensure-engine.sh' 2>/dev/null | head -1)"; [ -n "$ENSURE" ] && bash "$ENSURE" || { echo "devkit plugin not found — install from https://github.com/5uck1ess/devkit/releases"; exit 1; }
```

```bash
devkit workflow run tri-review
```

If the engine cannot be installed (no network, no write access), tell the user: "The devkit engine binary is required for deterministic workflow execution. Install manually from https://github.com/5uck1ess/devkit/releases" Do NOT fall back to manual steps.
