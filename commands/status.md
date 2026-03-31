---
name: devkit:status
description: Check devkit health — which external CLIs are installed, which agents are available, and which commands are ready to use.
---

# Devkit Status

Report on devkit installation health and available capabilities.

## Checks

### External CLIs

```bash
echo "=== CLI Status ==="
echo -n "codex: " && (command -v codex && codex --version 2>/dev/null || echo "not installed")
echo -n "gemini: " && (command -v gemini && gemini --version 2>/dev/null || echo "not installed")
echo -n "gh: " && (command -v gh && gh --version 2>/dev/null | head -1 || echo "not installed")
```

### Agent Availability

List all agents from `agents/` directory with their model and isolation settings.

### Command Availability

List all commands, marking which ones need external CLIs:

```
## Devkit Status

### External CLIs
| CLI | Status | Required by |
|-----|--------|------------|
| codex | ✓ installed (v1.2.0) | tri:* commands |
| gemini | ✗ not installed | tri:* commands |
| gh | ✓ installed (v2.40.0) | devkit:pr-ready |

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
