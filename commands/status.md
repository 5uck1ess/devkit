---
name: devkit:status
description: Check devkit health — which external CLIs are installed, which agents are available, and which commands are ready to use.
---

# Devkit Status

Report on devkit installation health and available capabilities.

## Checks

### Plugins

```bash
echo "=== Plugin Status ==="
echo -n "codex plugin: " && (/codex:status >/dev/null 2>&1 && echo "installed" || echo "not installed")
echo -n "gemini plugin: " && (/gemini:status >/dev/null 2>&1 && echo "installed" || echo "not installed")
```

### External CLIs

```bash
echo "=== CLI Status ==="
echo -n "codex: " && (command -v codex && codex --version 2>/dev/null || echo "not installed")
echo -n "gemini: " && (command -v gemini && gemini --version 2>/dev/null || echo "not installed")
echo -n "gh: " && (command -v gh && gh --version 2>/dev/null | head -1 || echo "not installed")
echo -n "rtk: " && (command -v rtk && rtk --version 2>/dev/null || echo "not installed (optional — 60-90% token savings)")
echo ""
echo "=== RTK Status ==="
if command -v rtk >/dev/null 2>&1; then
  echo "installed: $(rtk --version 2>/dev/null)"
  echo "latest: check with 'brew outdated rtk' or 'rtk --version'"
  rtk gain 2>/dev/null | head -5 || echo "no session data yet"
else
  echo "not installed — install with: brew install rtk"
  echo "saves 60-90% tokens on Bash output (tests, git, ls, grep, etc.)"
fi
```

### Agent Availability

List all agents from `agents/` directory with their model and isolation settings.

### Command Availability

List all commands, marking which ones need external CLIs:

```
## Devkit Status

### Plugins
| Plugin | Status | Commands |
|--------|--------|----------|
| codex | ✓ installed | /codex:rescue, /codex:review, /codex:result |
| gemini | ✗ not installed | /gemini:rescue, /gemini:review, /gemini:result |

### External CLIs (fallback if plugins not installed)
| CLI | Status | Required by |
|-----|--------|------------|
| codex | ✓ installed (v1.2.0) | tri:* commands (fallback) |
| gemini | ✗ not installed | tri:* commands (fallback) |
| gh | ✓ installed (v2.40.0) | devkit:pr-ready |
| rtk | ✓ installed (v0.34.2) | token optimization (optional) |

### Commands
| Command | Status | Notes |
|---------|--------|-------|
| /tri:review | ⚠ partial | 2/3 agents available (no gemini) |
| /tri:dispatch | ⚠ partial | 2/3 agents available |
| /self:improve | ✓ ready | Claude-only |
| /self:test | ✓ ready | Claude-only |
| /self:lint | ✓ ready | Claude-only |
| /self:perf | ✓ ready | Claude-only |
| /self:migrate | ✓ ready | Claude-only |
| /devkit:test-gen | ✓ ready | Claude-only |
| /devkit:doc-gen | ✓ ready | Claude-only |
| /devkit:pr-ready | ✓ ready | Uses gh for PR creation |
| /devkit:onboard | ✓ ready | Claude-only |
| /devkit:changelog | ✓ ready | Claude-only |
| /devkit:workflow | ✓ ready | Claude-only |

### Agents
| Agent | Model | Isolation | Status |
|-------|-------|-----------|--------|
| reviewer | opus | worktree | ✓ |
| researcher | sonnet | worktree | ✓ |
| improver | opus | worktree | ✓ |
| test-writer | sonnet | worktree | ✓ |
| documenter | haiku | worktree | ✓ |
| security-auditor | opus | worktree | ✓ |

### Workflows
{list from workflows/ directory, or "none defined"}
```

## Rules

- Check actual CLI availability with `command -v`, not just PATH
- Report versions where possible
- Clearly indicate which features work without external CLIs
- Show tri:* as "partial" if some but not all CLIs are present
