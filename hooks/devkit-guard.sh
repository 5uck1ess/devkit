#!/usr/bin/env bash
set -euo pipefail

# devkit-guard: PreToolUse hook that enforces workflow step ordering.
# Reads $CLAUDE_PLUGIN_DATA/session.json. Blocks out-of-step actions.
# Exit 0 = allow, Exit 2 + stderr = hard block.

DATA_DIR="${CLAUDE_PLUGIN_DATA:-}"
if [[ -z "$DATA_DIR" ]]; then
  exit 0  # not in plugin context
fi

SESSION_FILE="${DATA_DIR}/session.json"
if [[ ! -f "$SESSION_FILE" ]]; then
  exit 0  # no active workflow
fi

# Read session state (fast: no jq dependency, use python or inline parsing)
STATUS=$(python3 -c "import json,sys; d=json.load(open('$SESSION_FILE')); print(d.get('status',''))" 2>/dev/null || echo "")
STEP_TYPE=$(python3 -c "import json,sys; d=json.load(open('$SESSION_FILE')); print(d.get('step_type',''))" 2>/dev/null || echo "")
ENFORCE=$(python3 -c "import json,sys; d=json.load(open('$SESSION_FILE')); print(d.get('enforce','hard'))" 2>/dev/null || echo "hard")
CURRENT_STEP=$(python3 -c "import json,sys; d=json.load(open('$SESSION_FILE')); print(d.get('current_step',''))" 2>/dev/null || echo "")

if [[ "$STATUS" != "running" ]]; then
  exit 0
fi

# Read tool name from stdin
INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | python3 -c "import json,sys; print(json.load(sys.stdin).get('tool_name',''))" 2>/dev/null || echo "")

# Command steps: block all tools except devkit_advance (via MCP)
if [[ "$STEP_TYPE" == "command" ]]; then
  case "$TOOL_NAME" in
    Bash|Edit|Write|Read|Glob|Grep|Agent)
      if [[ "$ENFORCE" == "hard" ]]; then
        printf 'BLOCKED: Command step "%s" in progress. Call devkit_advance to execute it and proceed.\n' "$CURRENT_STEP" >&2
        exit 2
      fi
      ;;
  esac
fi

# All other cases: allow
exit 0
