#!/bin/bash
# devkit PR gate hook — prompts to run pr-ready pipeline before creating a PR
# Runs on PreToolUse for Bash tool
#
# Detects `gh pr create` commands and asks the user if they want to run
# the full pr-ready pipeline first.

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# Only trigger on gh pr create
echo "$COMMAND" | grep -qE 'gh\s+pr\s+create' || exit 0

# Check if pr-ready already ran this session (cooldown file)
PR_GATE_FILE="/tmp/devkit-pr-gate-done"
if [ -f "$PR_GATE_FILE" ]; then
  LAST=$(cat "$PR_GATE_FILE" 2>/dev/null)
  NOW=$(date +%s 2>/dev/null)
  if [ -n "$LAST" ] && [ -n "$NOW" ]; then
    ELAPSED=$(( NOW - LAST )) 2>/dev/null || ELAPSED=0
    if [ "$ELAPSED" -lt 600 ] 2>/dev/null; then
      # Pipeline already ran recently, allow the PR creation
      exit 0
    fi
  fi
fi

# Set cooldown so this only fires once
date +%s > "$PR_GATE_FILE" 2>/dev/null

# Ask the user
jq -n '{
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    permissionDecision: "ask",
    permissionDecisionReason: "PR creation detected — want to run /devkit:pr-ready first? (lint, test, security, DRY review, changelog). Say yes to run the pipeline, or approve to skip and create the PR directly."
  }
}'
exit 0
