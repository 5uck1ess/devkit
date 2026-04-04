#!/bin/bash
# devkit Stop hook — final quality gate before session ends
#
# Checks for common issues that indicate incomplete work:
# - Uncommitted changes left in the working tree
# - Merge conflict markers in tracked files
# - TODO/FIXME markers introduced in the current diff
#
# Stop hook schema:
#   { "decision": "approve" | "block", "reason": "string" }

set -euo pipefail

# Cooldown: only warn once per 5 minutes to avoid spamming during conversation
COOLDOWN_FILE="/tmp/devkit-stop-gate-cooldown"
if [ -f "$COOLDOWN_FILE" ]; then
  LAST=$(cat "$COOLDOWN_FILE" 2>/dev/null || echo 0)
  NOW=$(date +%s)
  ELAPSED=$(( NOW - LAST ))
  if [ "$ELAPSED" -lt 300 ]; then
    jq -n '{ decision: "approve" }'
    exit 0
  fi
fi

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
    CONFLICT_PATTERN='<''<<''<<''<< '
    CONFLICTS=$(git diff --name-only -z HEAD 2>/dev/null | xargs -0 grep -l -- "$CONFLICT_PATTERN" 2>/dev/null | head -3 || true)
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

# If warnings found, block with reason and set cooldown
if [ -n "$WARNINGS" ]; then
  date +%s > "$COOLDOWN_FILE"
  jq -n --arg reason "Quality gate: ${WARNINGS}Continue anyway?" '{
    decision: "block",
    reason: $reason
  }'
  exit 0
fi

# All clear
jq -n '{
  decision: "approve"
}'
exit 0
