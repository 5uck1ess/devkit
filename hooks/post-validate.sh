#!/bin/bash
# devkit PostToolUse hook — validates work after Bash/Edit/Write execution
#
# Checks for common post-execution issues:
# - Bash commands that silently failed (non-zero exit hidden in piped output)
# - Edit/Write operations that created files outside the repo
# - Accidental secret/credential content in written files
#
# Hook input (JSON on stdin):
#   .tool_name       = "Bash" | "Edit" | "Write"
#   .tool_input      = the original tool input
#   .tool_output     = the tool's output/result
#
# Exit codes:
#   0 → allow (with optional feedback message)
#   1 → hook error (allow by default)

set -euo pipefail

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
TOOL_OUTPUT=$(echo "$INPUT" | jq -r '.tool_output // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')
CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // empty')

# --- Bash: check for suppressed errors ---
if [ "$TOOL_NAME" = "Bash" ]; then
  # Flag common error patterns in output that the agent might miss
  ERROR_MATCHES=$(printf '%s\n' "$TOOL_OUTPUT" | grep -iE 'permission denied|no such file or directory|command not found|segmentation fault|killed|out of memory' | head -3 || true)
  if [ -n "$ERROR_MATCHES" ]; then
    jq -n --arg msg "$ERROR_MATCHES" '{
      hookSpecificOutput: {
        hookEventName: "PostToolUse",
        permissionDecision: "allow",
        permissionDecisionReason: ("Warning: command output contains error signals — verify this was expected: " + $msg)
      }
    }'
    exit 0
  fi
fi

# --- Edit/Write: check for secrets in content ---
if [ "$TOOL_NAME" = "Edit" ] || [ "$TOOL_NAME" = "Write" ]; then
  CHECK_CONTENT="$CONTENT"
  if [ -z "$CHECK_CONTENT" ]; then
    CHECK_CONTENT=$(echo "$INPUT" | jq -r '.tool_input.new_string // empty')
  fi

  if [ -n "$CHECK_CONTENT" ]; then
    # Check for patterns that look like hardcoded secrets
    if echo "$CHECK_CONTENT" | grep -qE '(sk-[a-zA-Z0-9]{20,}|AKIA[A-Z0-9]{16}|ghp_[a-zA-Z0-9]{36}|-----BEGIN (RSA |EC )?PRIVATE KEY)'; then
      jq -n '{
        hookSpecificOutput: {
          hookEventName: "PostToolUse",
          permissionDecision: "allow",
          permissionDecisionReason: "WARNING: Written content appears to contain a hardcoded secret or API key. Use environment variables instead."
        }
      }'
      exit 0
    fi
  fi

  # Check for writes outside the git repo
  if [ -n "$FILE_PATH" ]; then
    REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || true)
    if [ -n "$REPO_ROOT" ]; then
      ABS_PATH=$(realpath -m "$FILE_PATH" 2>/dev/null || echo "$FILE_PATH")
      case "$ABS_PATH" in
        "$REPO_ROOT"/*)
          ;; # within repo, OK
        /tmp/*|/private/tmp/*)
          ;; # temp files, OK
        *)
          jq -n --arg file "$FILE_PATH" --arg repo "$REPO_ROOT" '{
            hookSpecificOutput: {
              hookEventName: "PostToolUse",
              permissionDecision: "allow",
              permissionDecisionReason: ("Note: file written outside repository root (" + $repo + "): " + $file)
            }
          }'
          exit 0
          ;;
      esac
    fi
  fi
fi

# All clear
exit 0
