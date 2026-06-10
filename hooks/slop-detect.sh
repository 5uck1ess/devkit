#!/bin/bash
# devkit slop detection hook — catches AI-generated code patterns on PostToolUse
# Runs on PostToolUse for Edit and Write tools
#
# Detects:
# - Excessive documentation ratio (JSDoc/docstring lines > function body lines)
# - Unnecessary comments that restate the code
# - Boilerplate null checks and type guards that add no value
#
# PostToolUse hook schema:
#   { "hookSpecificOutput": { "hookEventName": "PostToolUse", "additionalContext": "string" } }

set -euo pipefail

# Fail open on malformed stdin — advisory hook, never a blocker.
INPUT=$(cat || true)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty' 2>/dev/null || true)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null || true)

[ "$TOOL_NAME" = "Edit" ] || [ "$TOOL_NAME" = "Write" ] || exit 0
[ -z "$FILE_PATH" ] && exit 0
[ -f "$FILE_PATH" ] || exit 0

WARNINGS=""

# --- Doc/code ratio check ---
# For JS/TS/Python files, check if documentation blocks outweigh code
if echo "$FILE_PATH" | grep -qE '\.(js|jsx|ts|tsx|mjs|cjs)$'; then
  # Count JSDoc/block comment lines vs code lines
  # NB: grep -c prints "0" itself on no match (exiting 1), so the guard
  # must be `|| true` — `|| echo 0` would yield "0\n0" and break the
  # integer comparisons below.
  DOC_LINES=$(grep -cE '^\s*(\*|/\*\*|///|\s*\*)' "$FILE_PATH" 2>/dev/null || true)
  TOTAL_LINES=$(wc -l < "$FILE_PATH" 2>/dev/null | tr -d ' ' || echo 1)
  BLANK_LINES=$(grep -cE '^\s*$' "$FILE_PATH" 2>/dev/null || true)
  CODE_LINES=$(( ${TOTAL_LINES:-1} - ${DOC_LINES:-0} - ${BLANK_LINES:-0} ))
  [ "$CODE_LINES" -lt 1 ] && CODE_LINES=1

  if [ "$DOC_LINES" -gt 10 ] && [ "$DOC_LINES" -gt "$CODE_LINES" ]; then
    WARNINGS="${WARNINGS}Doc/code ratio: ${DOC_LINES} doc lines vs ${CODE_LINES} code lines — documentation outweighs code. "
  fi
fi

if echo "$FILE_PATH" | grep -qE '\.py$'; then
  DOC_LINES=$(grep -cE '^\s*("""|'\'''\'''\''|#)' "$FILE_PATH" 2>/dev/null || true)
  TOTAL_LINES=$(wc -l < "$FILE_PATH" 2>/dev/null | tr -d ' ' || echo 1)
  BLANK_LINES=$(grep -cE '^\s*$' "$FILE_PATH" 2>/dev/null || true)
  CODE_LINES=$(( ${TOTAL_LINES:-1} - ${DOC_LINES:-0} - ${BLANK_LINES:-0} ))
  [ "$CODE_LINES" -lt 1 ] && CODE_LINES=1

  if [ "$DOC_LINES" -gt 10 ] && [ "$DOC_LINES" -gt "$CODE_LINES" ]; then
    WARNINGS="${WARNINGS}Doc/code ratio: ${DOC_LINES} doc lines vs ${CODE_LINES} code lines — documentation outweighs code. "
  fi
fi

# --- Restating-the-obvious comments ---
# Comments that just repeat the next line of code (e.g., "// Set the value" above "setValue(x)")
OBVIOUS=$(grep -nE '^\s*//' "$FILE_PATH" 2>/dev/null | while read -r line; do
  LINENUM=$(echo "$line" | cut -d: -f1)
  COMMENT=$(echo "$line" | cut -d: -f2- | sed 's|^\s*//\s*||' | tr '[:upper:]' '[:lower:]' | tr -d ' ')
  NEXTLINE=$(sed -n "$((LINENUM + 1))p" "$FILE_PATH" 2>/dev/null | tr '[:upper:]' '[:lower:]' | tr -d ' /(){}_;')
  # If the comment (stripped) is a substring of the next line or vice versa (>5 chars)
  if [ ${#COMMENT} -gt 5 ] && [ ${#NEXTLINE} -gt 5 ]; then
    if echo "$NEXTLINE" | grep -qF "$COMMENT" 2>/dev/null || echo "$COMMENT" | grep -qF "$NEXTLINE" 2>/dev/null; then
      echo "$LINENUM"
    fi
  fi
done | head -3 || true)

if [ -n "$OBVIOUS" ]; then
  COUNT=$(echo "$OBVIOUS" | wc -l | tr -d ' ')
  WARNINGS="${WARNINGS}Found ${COUNT} comments that restate the code (lines: $(echo "$OBVIOUS" | tr '\n' ',' | sed 's/,$//')). "
fi

# --- Excessive type annotations in JS (not TS) ---
if echo "$FILE_PATH" | grep -qE '\.(js|jsx|mjs|cjs)$'; then
  JSDOC_TYPES=$(grep -cE '@(param|returns|type|typedef)\s' "$FILE_PATH" 2>/dev/null || true)
  FUNCTIONS=$(grep -cE '(function\s|=>|async\s)' "$FILE_PATH" 2>/dev/null || true)
  [ "${FUNCTIONS:-0}" -lt 1 ] 2>/dev/null && FUNCTIONS=1
  RATIO=$(( ${JSDOC_TYPES:-0} / ${FUNCTIONS:-1} ))
  if [ "$RATIO" -gt 4 ]; then
    WARNINGS="${WARNINGS}Excessive JSDoc type annotations in .js file (${JSDOC_TYPES} annotations for ${FUNCTIONS} functions) — consider using TypeScript instead. "
  fi
fi

# Report if warnings found
if [ -n "$WARNINGS" ]; then
  jq -n --arg msg "$WARNINGS" '{
    hookSpecificOutput: {
      hookEventName: "PostToolUse",
      additionalContext: ("Slop detection: " + $msg + "Consider trimming unnecessary documentation and comments.")
    }
  }'
  exit 0
fi

# All clear
exit 0
