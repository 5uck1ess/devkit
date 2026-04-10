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

# Parse all session fields in a single python3 call (no jq dependency).
# Outputs tab-separated: status, step_type, enforce, current_step.
# Passes file path via sys.argv to prevent shell injection.
SESSION_DATA=$(python3 -c "
import json, sys
d = json.load(open(sys.argv[1]))
print('\t'.join([
    d.get('status', ''),
    d.get('step_type', ''),
    d.get('enforce', 'hard'),
    d.get('current_step', '')
]))
" "$SESSION_FILE" 2>/dev/null) || {
  # python3 unavailable or JSON corrupt — fail closed if session file exists
  printf 'BLOCKED: Cannot parse session state (python3 required). Remove %s to clear.\n' "$SESSION_FILE" >&2
  exit 2
}

IFS=$'\t' read -r STATUS STEP_TYPE ENFORCE CURRENT_STEP <<< "$SESSION_DATA"

if [[ "$STATUS" != "running" ]]; then
  exit 0
fi

# Read tool name from stdin
INPUT=$(cat)
TOOL_NAME=$(printf '%s' "$INPUT" | python3 -c "import json,sys; print(json.load(sys.stdin).get('tool_name',''))" 2>/dev/null || echo "")

# Command steps: block all standard tools — only MCP tools (devkit_advance) allowed
if [[ "$STEP_TYPE" == "command" && "$ENFORCE" == "hard" ]]; then
  case "$TOOL_NAME" in
    Bash|Edit|Write|Read|Glob|Grep|Agent|WebFetch|WebSearch|NotebookEdit|Skill)
      printf 'BLOCKED: Command step "%s" in progress. Call devkit_advance to execute it and proceed.\n' "$CURRENT_STEP" >&2
      exit 2
      ;;
  esac
fi

# All other cases: allow
exit 0
