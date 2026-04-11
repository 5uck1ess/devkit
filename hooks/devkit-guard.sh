#!/usr/bin/env bash
set -euo pipefail

# devkit-guard: PreToolUse hook that enforces workflow step ordering.
# Reads $CLAUDE_PLUGIN_DATA/session.json. Blocks out-of-step actions.
# Exit 0 = allow, Exit 2 + stderr = hard block.
#
# Policy matrix:
#   step_type=command, enforce=hard → allow only devkit MCP + TodoWrite
#                                     (engine runs the command, not Claude)
#   step_type=prompt,  enforce=hard → allow Read/Grep/Glob/NotebookRead/
#                                     TodoWrite + devkit MCP. Forces the
#                                     agent to advance before any
#                                     write/bash/dispatch. Closes issue #63
#                                     drift hole.
#   step_type=prompt,  enforce=soft → allow everything, emit stderr nudge
#   step_type=parallel             → allow everything (engine dispatches)
#   stale session (see lib/read-session.sh) → allow + warn; do not enforce
#                                     against an orphaned state file.
#
# This hook uses an ALLOWLIST rather than a blocklist because the
# Claude Code tool surface evolves — Task, SlashCommand, ExitPlanMode,
# BashOutput, KillBash, TodoWrite, any mcp__* tool, and future names
# would silently bypass a blocklist of hardcoded names.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/read-session.sh
source "${SCRIPT_DIR}/lib/read-session.sh"

DATA_DIR="${CLAUDE_PLUGIN_DATA:-}"
if [[ -z "$DATA_DIR" ]]; then
  printf 'devkit-guard: CLAUDE_PLUGIN_DATA unset — enforcement disabled\n' >&2
  exit 0
fi

SESSION_FILE="${DATA_DIR}/session.json"

if ! parse_session_fields "$SESSION_FILE"; then
  # python3 unavailable or JSON corrupt — fail closed if session file
  # exists, otherwise fall through (no session = nothing to guard).
  if [[ -f "$SESSION_FILE" ]]; then
    printf 'BLOCKED: Cannot parse session state (python3 required or JSON corrupt). Remove %s to clear.\n' "$SESSION_FILE" >&2
    exit 2
  fi
  exit 0
fi

if [[ "$SESSION_STATUS" != "running" ]]; then
  exit 0
fi

if [[ "$SESSION_STALE" == "1" ]]; then
  printf 'devkit-guard: session %s idle past TTL — treating as orphaned (run devkit_start to reclaim)\n' "$SESSION_WORKFLOW" >&2
  exit 0
fi

# Read tool name from stdin. Matches PreToolUse payload format.
INPUT=$(cat)
TOOL_NAME=$(printf '%s' "$INPUT" | python3 -c "import json,sys; print(json.load(sys.stdin).get('tool_name',''))" 2>/dev/null) || {
  # Malformed payload — surface a diagnostic so the transcript shows
  # why the next veto lists an empty tool name, instead of letting the
  # BLOCKED message say "(attempted tool: )" with no hint.
  printf 'devkit-guard: could not parse tool name from PreToolUse payload (python3 or JSON error)\n' >&2
  TOOL_NAME=""
}

# Build a progress label for veto messages so the agent always sees
# workflow + position without another devkit_status round trip.
step_label() {
  if [[ -n "$SESSION_CURRENT_INDEX" && -n "$SESSION_TOTAL_STEPS" ]]; then
    local human_index=$((SESSION_CURRENT_INDEX + 1))
    printf '%s step %d/%d (%s)' "$SESSION_WORKFLOW" "$human_index" "$SESSION_TOTAL_STEPS" "$SESSION_CURRENT_STEP"
  else
    printf '%s (%s)' "$SESSION_WORKFLOW" "$SESSION_CURRENT_STEP"
  fi
}

# Command steps: allow ONLY the MCP tools needed to progress the
# workflow. Everything else is blocked, including future tools.
if [[ "$SESSION_STEP_TYPE" == "command" && "$SESSION_ENFORCE" == "hard" ]]; then
  case "$TOOL_NAME" in
    mcp__*devkit-engine*|mcp__devkit__*|devkit_advance|devkit_status|devkit_list|devkit_start)
      exit 0
      ;;
    TodoWrite)
      exit 0
      ;;
    *)
      printf 'BLOCKED: Command step "%s" in progress — the engine runs this step. Call devkit_advance to execute it. (attempted tool: %s)\n' "$(step_label)" "$TOOL_NAME" >&2
      exit 2
      ;;
  esac
fi

# Prompt steps under hard enforcement: allow read-only evidence tools
# plus devkit MCP. Blocks Write/Edit/Bash/Task/WebFetch/other MCP so
# the agent cannot drift into unrelated work between step 1 and
# devkit_advance. See issue #63.
if [[ "$SESSION_STEP_TYPE" == "prompt" && "$SESSION_ENFORCE" == "hard" ]]; then
  case "$TOOL_NAME" in
    mcp__*devkit-engine*|mcp__devkit__*|devkit_advance|devkit_status|devkit_list|devkit_start)
      exit 0
      ;;
    Read|Grep|Glob|TodoWrite|NotebookRead)
      exit 0
      ;;
    *)
      printf 'BLOCKED: devkit workflow %s is at a prompt step — gather evidence with Read/Grep/Glob then call devkit_advance. (attempted tool: %s)\n' "$(step_label)" "$TOOL_NAME" >&2
      exit 2
      ;;
  esac
fi

# Prompt steps under soft enforcement: allow everything, but inject a
# stderr nudge so the transcript shows the agent that a step is open.
# Soft nudge is idempotent — if the agent ignores it, Stop gate still
# blocks via devkit-stop-guard.sh.
if [[ "$SESSION_STEP_TYPE" == "prompt" && "$SESSION_ENFORCE" != "hard" ]]; then
  printf 'devkit-guard: %s is open — call devkit_advance when the step is complete.\n' "$(step_label)" >&2
  exit 0
fi

# Parallel steps: engine is dispatching, agent needs full tool access.
exit 0
