#!/bin/bash
# devkit RTK hook — rewrites Bash commands through rtk for token savings
# Runs on PreToolUse for Bash tool. No-op if rtk is not installed.

# Skip if rtk is not installed
command -v rtk >/dev/null 2>&1 || exit 0

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# Skip empty commands
[ -z "$COMMAND" ] && exit 0

# Try to rewrite through rtk
REWRITTEN=$(rtk rewrite "$COMMAND" 2>/dev/null) || exit 0

# If rewrite produced the same command, skip
[ "$REWRITTEN" = "$COMMAND" ] && exit 0

# Output the rewrite
jq -n --arg cmd "$REWRITTEN" '{
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    permissionDecision: "allow",
    updatedInput: {
      command: $cmd
    }
  }
}'
