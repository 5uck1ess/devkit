#!/usr/bin/env bash
set -euo pipefail
# devkit PR gate hook — prompts to run pr-ready pipeline before creating a PR
# Runs on PreToolUse for Bash tool
#
# Detects `gh pr create` commands and asks the user if they want to run
# the full pr-ready pipeline first.

INPUT=$(cat)
COMMAND=$(printf '%s' "$INPUT" | jq -r '.tool_input.command // empty')

# Only trigger on gh pr create
printf '%s' "$COMMAND" | grep -qE 'gh[[:space:]]+pr[[:space:]]+create' || exit 0

# Check if pr-ready already ran recently (cooldown file). Per-user
# rather than /tmp-global so concurrent projects don't share state.
COOLDOWN_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/devkit"
PR_GATE_FILE="${COOLDOWN_DIR}/pr-gate-done"
mkdir -p "$COOLDOWN_DIR" 2>/dev/null || true

if [[ -f "$PR_GATE_FILE" ]]; then
  LAST=$(cat "$PR_GATE_FILE" 2>/dev/null || printf '0')
  NOW=$(date +%s)
  # Only arithmetic-compare if both values are numeric, else treat as expired.
  if [[ "$LAST" =~ ^[0-9]+$ ]] && (( NOW - LAST < 600 )); then
    # Pipeline already ran recently; skip the prompt.
    exit 0
  fi
fi

# Set cooldown so this only fires once per 10 minutes.
date +%s > "$PR_GATE_FILE" 2>/dev/null || true

# Ask the user
jq -n '{
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    permissionDecision: "ask",
    permissionDecisionReason: "PR creation detected — want to run the pr-ready pipeline first? (lint, test, security, changelog, doc-check, monitor). Say yes to run it, or approve to skip and create the PR directly."
  }
}'
exit 0
