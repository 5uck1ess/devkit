#!/bin/bash
# devkit PostToolUse hook — detects Go functions that always return nil error
#
# Scans written Go code for functions with error return types where
# every return statement returns nil for the error. This pattern
# silently swallows failures and is a top LLM-generated bug category.
#
# PostToolUse hook schema:
#   { "hookSpecificOutput": { "hookEventName": "PostToolUse", "additionalContext": "string" } }

set -euo pipefail

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')
CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // .tool_input.new_string // empty')

# Only check Go files on Edit/Write
case "$FILE_PATH" in
  *.go) ;;
  *) exit 0 ;;
esac
if [ "$TOOL_NAME" != "Edit" ] && [ "$TOOL_NAME" != "Write" ]; then
  exit 0
fi
[ -z "$CONTENT" ] && exit 0

# Use awk to find functions that return error but only ever return nil
# Strategy: track function signatures with error returns, then check
# if ALL return statements in that function use nil for the error position
WARNINGS=$(echo "$CONTENT" | awk '
  # Match function declarations that return error (last return type)
  /^func .*\)\s*(\(.*error\)|error)\s*\{/ {
    fname = $0
    sub(/\{.*/, "", fname)
    in_func = 1
    brace_depth = 0
    has_return = 0
    has_non_nil_err = 0
    # Count all braces on the declaration line itself
    line = $0
    for (j = 1; j <= length(line); j++) {
      c = substr(line, j, 1)
      if (c == "{") brace_depth++
      if (c == "}") brace_depth--
    }
    next
  }

  in_func {
    # Track brace depth
    line = $0
    for (i = 1; i <= length(line); i++) {
      c = substr(line, i, 1)
      if (c == "{") brace_depth++
      if (c == "}") brace_depth--
    }

    # Check return statements
    if ($0 ~ /return /) {
      has_return = 1
      # Check if error position is non-nil (not "nil" or "nil)")
      if ($0 !~ /,\s*nil\s*$/ && $0 !~ /return nil\s*$/ && $0 !~ /,\s*nil\s*\)/) {
        has_non_nil_err = 1
      }
    }

    # End of function
    if (brace_depth <= 0) {
      if (has_return && !has_non_nil_err) {
        # Strip leading whitespace from function name
        gsub(/^[[:space:]]+/, "", fname)
        print fname
      }
      in_func = 0
    }
  }
')

if [ -n "$WARNINGS" ]; then
  # Limit to first 3 functions to avoid noise
  FUNCS=$(echo "$WARNINGS" | head -3)
  COUNT=$(echo "$WARNINGS" | wc -l | tr -d ' ')
  MSG="Go nil-error pattern: ${COUNT} function(s) return error but only ever return nil. This silently swallows failures:"
  while IFS= read -r fn; do
    MSG="$MSG\n  - $fn"
  done <<< "$FUNCS"
  if [ "$COUNT" -gt 3 ]; then
    MSG="$MSG\n  ... and $((COUNT - 3)) more"
  fi

  jq -n --arg msg "$MSG" '{
    hookSpecificOutput: {
      hookEventName: "PostToolUse",
      additionalContext: $msg
    }
  }'
  exit 0
fi

exit 0
