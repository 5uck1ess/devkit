#!/bin/bash
# devkit RTK hook — rewrites Bash commands through rtk for token savings
# Runs on PreToolUse for Bash tool. No-op if rtk is not installed.
#
# rtk rewrite exit-code protocol (from the rtk binary):
#   0 + stdout   rewrite found, no deny/ask rule matched → auto-allow
#   1            no RTK equivalent → pass through unchanged
#   2            deny rule matched → pass through (Claude Code native deny handles it)
#   3 + stdout   ask rule matched → rewrite but let Claude Code prompt the user
set -euo pipefail

command -v rtk >/dev/null 2>&1 || exit 0

INPUT=$(cat)
COMMAND=$(printf '%s' "$INPUT" | jq -r '.tool_input.command // empty')

[[ -z "$COMMAND" ]] && exit 0

set +e
REWRITTEN=$(rtk rewrite "$COMMAND" 2>/dev/null)
RC=$?
set -e

case "$RC" in
  0) DECISION="allow" ;;
  3) DECISION="ask"   ;;
  *) exit 0 ;;  # 1 (no equivalent), 2 (deny handled natively), or anything else → pass through
esac

[[ -z "$REWRITTEN" ]] && exit 0
[[ "$REWRITTEN" == "$COMMAND" ]] && exit 0

DESCRIPTION=$(printf '%s' "$INPUT" | jq -r '.tool_input.description // empty')

jq -n --arg cmd "$REWRITTEN" --arg desc "$DESCRIPTION" --arg decision "$DECISION" '{
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    permissionDecision: $decision,
    updatedInput: {
      command: $cmd,
      description: $desc
    }
  }
}'
