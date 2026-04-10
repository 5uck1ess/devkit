#!/usr/bin/env bash
set -euo pipefail

# devkit-guard: PreToolUse hook that enforces workflow step ordering.
# Reads $CLAUDE_PLUGIN_DATA/session.json. Blocks out-of-step actions.
# Exit 0 = allow, Exit 2 + stderr = hard block.
#
# Policy: during a command step (workflow.yml `command:`), the engine —
# not Claude — executes the shell. The only tool Claude is allowed to
# call is devkit_advance (which triggers execution and returns the next
# step). Everything else is blocked so Claude cannot observe or
# interfere with the state of the step.
#
# This hook uses an ALLOWLIST rather than a blocklist because the
# Claude Code tool surface evolves — Task, SlashCommand, ExitPlanMode,
# BashOutput, KillBash, TodoWrite, any mcp__* tool, and future names
# would silently bypass a blocklist of hardcoded names.

DATA_DIR="${CLAUDE_PLUGIN_DATA:-}"
if [[ -z "$DATA_DIR" ]]; then
  printf 'devkit-guard: CLAUDE_PLUGIN_DATA unset — enforcement disabled\n' >&2
  exit 0  # not in plugin context
fi

SESSION_FILE="${DATA_DIR}/session.json"
if [[ ! -f "$SESSION_FILE" ]]; then
  exit 0  # no active workflow
fi

# Parse all session fields in a single python3 call (no jq dependency).
# Outputs tab-separated: status, step_type, enforce, current_step.
# Passes file path via sys.argv to prevent shell injection. Handles
# FileNotFoundError so a TOCTOU race (file cleared between -f and open)
# is treated as "no session," matching the pre-check intent.
SESSION_DATA=$(python3 -c "
import json, sys
try:
    d = json.load(open(sys.argv[1]))
except FileNotFoundError:
    print('\t'.join(['', '', '', '']))
    sys.exit(0)
print('\t'.join([
    d.get('status', ''),
    d.get('step_type', ''),
    d.get('enforce', 'hard'),
    d.get('current_step', '')
]))
" "$SESSION_FILE" 2>/dev/null) || {
  # python3 unavailable or JSON corrupt — fail closed if session file exists
  printf 'BLOCKED: Cannot parse session state (python3 required or JSON corrupt). Remove %s to clear.\n' "$SESSION_FILE" >&2
  exit 2
}

IFS=$'\t' read -r STATUS STEP_TYPE ENFORCE CURRENT_STEP <<< "$SESSION_DATA"

if [[ "$STATUS" != "running" ]]; then
  exit 0
fi

# Read tool name from stdin
INPUT=$(cat)
TOOL_NAME=$(printf '%s' "$INPUT" | python3 -c "import json,sys; print(json.load(sys.stdin).get('tool_name',''))" 2>/dev/null || echo "")

# Command steps: allow ONLY the MCP tools needed to progress the
# workflow. Everything else is blocked, including future tools the
# hook author hasn't heard of.
if [[ "$STEP_TYPE" == "command" && "$ENFORCE" == "hard" ]]; then
  case "$TOOL_NAME" in
    # MCP devkit tools — Claude uses these to drive the engine.
    mcp__*devkit*|devkit_advance|devkit_status|devkit_list|devkit_start)
      exit 0
      ;;
    # TodoWrite is a pure in-memory tracker with no side effects, allowed.
    TodoWrite)
      exit 0
      ;;
    *)
      printf 'BLOCKED: Command step "%s" in progress — the engine runs this step. Call devkit_advance to execute it. (attempted tool: %s)\n' "$CURRENT_STEP" "$TOOL_NAME" >&2
      exit 2
      ;;
  esac
fi

# Prompt/parallel steps: allow everything (Claude needs full tool access).
exit 0
