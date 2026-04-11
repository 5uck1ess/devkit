---
name: status
description: Check devkit health — which external CLIs (codex, gemini, gh, rtk, sg, gcli) are installed, which agents are available, which skills are ready to use. Use when asked about devkit status, "is devkit working?", "what's installed?", "what devkit capabilities do I have?", diagnosing devkit setup issues, or running /devkit:status. Read-only diagnostic, safe to auto-invoke.
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

### Devkit Engine

Use the `devkit_status` tool to check workflow progress.

### External CLIs

```bash
echo "=== CLI Status ==="
echo -n "codex: " && (command -v codex && codex --version 2>/dev/null || echo "not installed")
echo -n "gemini: " && (command -v gemini && gemini --version 2>/dev/null || echo "not installed")
echo -n "gh: " && (command -v gh && gh --version 2>/dev/null | head -1 || echo "not installed")
echo -n "rtk: " && (command -v rtk && rtk --version 2>/dev/null || echo "not installed (optional — 60-90% token savings)")
echo -n "gcli: " && (command -v gcli >/dev/null 2>&1 && echo "installed" || echo "not installed (optional — Google Workspace: Gmail, Calendar, Drive)")
echo -n "sg: " && (command -v sg >/dev/null 2>&1 && (sg --version 2>/dev/null || echo "installed") || echo "not installed (optional — AST-based repo mapping)")
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

### Skill Availability

List all skills from `skills/` directory, marking which ones need external CLIs:

```
## Devkit Status

### Plugins
| Plugin | Status | Skills |
|--------|--------|----------|
| codex | ✓ installed | /codex:rescue, /codex:review, /codex:result |
| gemini | ✗ not installed | /gemini:rescue, /gemini:review, /gemini:result |

### External CLIs (fallback if plugins not installed)
| CLI | Status | Required by |
|-----|--------|------------|
| codex | ✓ installed (v1.2.0) | tri:* skills (fallback) |
| gemini | ✗ not installed | tri:* skills (fallback) |
| gh | ✓ installed (v2.40.0) | pr-ready skill |
| rtk | ✓ installed (v0.34.2) | token optimization (optional) |

### Skills
| Skill | Status | Notes |
|-------|--------|-------|
| /tri-review | ⚠ partial | 2/3 agents available (no gemini) |
| /tri-debug | ⚠ partial | 2/3 agents available |
| /tri-security | ⚠ partial | 2/3 agents available |
| /devkit:status | ✓ ready | This skill |
| /devkit:setup-rules | ✓ ready | One-time setup |

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
