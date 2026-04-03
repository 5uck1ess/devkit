#!/bin/bash
# devkit Stop hook — final quality gate before session ends
#
# Checks for common issues that indicate incomplete work:
# - Uncommitted changes left in the working tree
# - Merge conflict markers in tracked files
# - TODO/FIXME markers introduced in the current diff
#
# Hook input (JSON on stdin):
#   .tool_name  = "Stop"
#   .session_id = current session ID (if available)
#
# Exit codes:
#   0 + permissionDecision "allow"  → session may end
#   0 + permissionDecision "ask"    → prompt user before ending
#   1                               → hook error (allow by default)

set -euo pipefail

WARNINGS=""

# Check for uncommitted changes
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  DIRTY=$(git status --porcelain 2>/dev/null | head -5)
  if [ -n "$DIRTY" ]; then
    WARNINGS="${WARNINGS}Uncommitted changes detected. "
  fi

  # Check for merge conflict markers in staged/modified files
  CHANGED_FILES=$(git diff --name-only HEAD 2>/dev/null || true)
  if [ -n "$CHANGED_FILES" ]; then
    CONFLICTS=$(git diff --name-only -z HEAD 2>/dev/null | xargs -0 grep -l -- '<<<<<<< ' 2>/dev/null | head -3 || true)
    if [ -n "$CONFLICTS" ]; then
      WARNINGS="${WARNINGS}Merge conflict markers found in: ${CONFLICTS}. "
    fi

    # Check for new TODO/FIXME in diff (not in the whole file, just new lines)
    NEW_TODOS=$(git diff HEAD 2>/dev/null | grep '^+' | grep -iE '(TODO|FIXME|HACK|XXX):' | head -3 || true)
    if [ -n "$NEW_TODOS" ]; then
      WARNINGS="${WARNINGS}New TODO/FIXME markers in diff. "
    fi
  fi
fi

# If warnings found, prompt the user
if [ -n "$WARNINGS" ]; then
  jq -n --arg reason "$WARNINGS" '{
    hookSpecificOutput: {
      hookEventName: "Stop",
      permissionDecision: "ask",
      permissionDecisionReason: ("Quality gate: " + $reason + "Continue anyway?")
    }
  }'
  exit 0
fi

# All clear
jq -n '{
  hookSpecificOutput: {
    hookEventName: "Stop",
    permissionDecision: "allow"
  }
}'
exit 0
