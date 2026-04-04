#!/bin/bash
# devkit PreToolUse hook — shell script portability check
#
# Flags non-portable constructs in shell scripts that break on macOS:
# - grep -P (Perl regex, BSD grep doesn't support it)
# - sed -i without '' (GNU vs BSD sed)
# - readlink -f (use realpath or manual resolution)
# - stat --format (GNU stat, not BSD)
# - xargs -d (GNU xargs, not BSD)
# - date -d (GNU date, not BSD)
#
# PreToolUse hook schema:
#   { "hookSpecificOutput": { "hookEventName": "PreToolUse", "permissionDecision": "ask", ... } }

set -euo pipefail

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')
CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // .tool_input.new_string // empty')

# Only check shell scripts
case "$FILE_PATH" in
  *.sh) ;;
  *) exit 0 ;;
esac

# Only check Edit/Write
if [ "$TOOL_NAME" != "Edit" ] && [ "$TOOL_NAME" != "Write" ]; then
  exit 0
fi

[ -z "$CONTENT" ] && exit 0

# Session dedup
SEEN_FILE="/tmp/devkit-shellcompat-seen-$$"

check_compat() {
  local pattern="$1"
  local message="$2"
  local key="${FILE_PATH}:${pattern}"

  if echo "$CONTENT" | grep -qE "$pattern"; then
    if [ -f "$SEEN_FILE" ] && grep -qF "$key" "$SEEN_FILE" 2>/dev/null; then
      return
    fi
    echo "$key" >> "$SEEN_FILE" 2>/dev/null

    jq -n --arg reason "$message" '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "ask",
        permissionDecisionReason: $reason
      }
    }'
    exit 0
  fi
}

check_compat 'grep\s+(-[a-zA-Z]*P|--perl-regexp)' \
  "Portability: grep -P (Perl regex) is unavailable on macOS — use grep -E, awk, or perl instead"

check_compat 'sed\s+-i\s+[^'"'"'"]' \
  "Portability: sed -i without '' breaks on macOS BSD sed — use sed -i '' for in-place edits"

check_compat 'readlink\s+-f\b' \
  "Portability: readlink -f is GNU-only — use realpath or manual loop on macOS"

check_compat 'stat\s+--format' \
  "Portability: stat --format is GNU-only — use stat -f on macOS"

check_compat 'xargs\s+-d\b' \
  "Portability: xargs -d is GNU-only — use tr + xargs or while-read on macOS"

check_compat 'date\s+-d\b' \
  "Portability: date -d is GNU-only — use date -j -f on macOS"

check_compat 'mktemp\s+--suffix' \
  "Portability: mktemp --suffix is GNU-only — use mktemp with template on macOS"

# All clear
exit 0
