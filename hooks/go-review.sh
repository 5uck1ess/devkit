#!/bin/bash
# devkit PostToolUse hook — Go code quality patterns
#
# Checks written Go code for common LLM-generated bug patterns:
# 1. Accessing result fields in error paths
# 2. Goroutines reading shared maps without protection
# 3. Functions that always return nil error
# 4. Unsanitized user input in filepath operations
#
# PostToolUse hook schema:
#   { "hookSpecificOutput": { "hookEventName": "PostToolUse", "additionalContext": "string" } }

set -euo pipefail

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')
CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // .tool_input.new_string // empty')

# Only check Go files
case "$FILE_PATH" in
  *.go) ;;
  *) exit 0 ;;
esac

# Only check Edit/Write
if [ "$TOOL_NAME" != "Edit" ] && [ "$TOOL_NAME" != "Write" ]; then
  exit 0
fi

if [ -z "$CONTENT" ]; then
  exit 0
fi

WARNINGS=""

# Pattern 1: Accessing result after error check
# Detects: if err != nil { ... } followed by result.Something on the same or next few lines
if echo "$CONTENT" | grep -qE 'if err != nil' && echo "$CONTENT" | grep -qP 'err != nil[\s\S]{0,200}(result\.|res\.)'; then
  # More specific: check if result is accessed INSIDE the error block
  if echo "$CONTENT" | grep -qP 'if err != nil \{[^}]*(result\.|res\.)[^}]*\}'; then
    WARNINGS="$WARNINGS\n- Possible result field access inside error path (result may be zero-value when err != nil)"
  fi
fi

# Pattern 2: Goroutines with shared map access
if echo "$CONTENT" | grep -qE 'go func' && echo "$CONTENT" | grep -qE 'map\[string\]'; then
  if ! echo "$CONTENT" | grep -qE '(sync\.Mutex|sync\.RWMutex|sync\.Map|snapshot|Snap)'; then
    WARNINGS="$WARNINGS\n- Goroutines detected with map usage but no visible mutex/snapshot — verify concurrent map access is safe"
  fi
fi

# Pattern 3: filepath.Join with unsanitized variable
if echo "$CONTENT" | grep -qE 'filepath\.Join.*\b(name|input|arg|param|user)'; then
  if ! echo "$CONTENT" | grep -qE '(regexp|Regexp|MustCompile|MatchString|ValidateName|sanitize)'; then
    WARNINGS="$WARNINGS\n- filepath.Join with potentially unsanitized input — validate before constructing paths"
  fi
fi

if [ -n "$WARNINGS" ]; then
  MSG=$(printf "Go code quality check:%b" "$WARNINGS")
  jq -n --arg msg "$MSG" '{
    hookSpecificOutput: {
      hookEventName: "PostToolUse",
      additionalContext: $msg
    }
  }'
  exit 0
fi

# All clear
exit 0
