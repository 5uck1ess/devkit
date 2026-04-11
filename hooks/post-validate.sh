#!/bin/bash
# devkit PostToolUse hook — validates work after Bash/Edit/Write execution
#
# Checks for common post-execution issues:
# - Bash commands that silently failed (non-zero exit hidden in piped output)
# - Edit/Write operations that created files outside the repo
# - Accidental secret/credential content in written files
#
# PostToolUse hook schema:
#   { "hookSpecificOutput": { "hookEventName": "PostToolUse", "additionalContext": "string" } }

set -euo pipefail

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
TOOL_OUTPUT=$(echo "$INPUT" | jq -r '.tool_output // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')
CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // empty')

# --- Bash: check for suppressed errors ---
if [ "$TOOL_NAME" = "Bash" ]; then
  ERROR_MATCHES=$(printf '%s\n' "$TOOL_OUTPUT" | grep -iE 'permission denied|no such file or directory|command not found|segmentation fault|killed|out of memory' | head -3 || true)
  if [ -n "$ERROR_MATCHES" ]; then
    jq -n --arg msg "$ERROR_MATCHES" '{
      hookSpecificOutput: {
        hookEventName: "PostToolUse",
        additionalContext: ("Warning: command output contains error signals — verify this was expected: " + $msg)
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
    if echo "$CHECK_CONTENT" | grep -qE '(sk-[a-zA-Z0-9]{20,}|AKIA[A-Z0-9]{16}|ghp_[a-zA-Z0-9]{36}|-----BEGIN (RSA |EC )?PRIVATE KEY)'; then
      jq -n '{
        hookSpecificOutput: {
          hookEventName: "PostToolUse",
          additionalContext: "WARNING: Written content appears to contain a hardcoded secret or API key. Use environment variables instead."
        }
      }'
      exit 0
    fi
  fi

  # Check for writes outside the git repo. Must resolve .. segments and
  # symlinks so REPO_ROOT and ABS_PATH compare against a common canonical
  # form — GNU `realpath -m` isn't available on macOS BSD realpath, so we
  # do it portably here.
  if [ -n "$FILE_PATH" ]; then
    REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || true)
    if [ -n "$REPO_ROOT" ]; then
      # Canonicalize REPO_ROOT so a symlinked repo path (e.g. /Users/x/dev
      # → /Volumes/Work/dev) matches $(pwd -P) inside the repo.
      REPO_ROOT=$(cd "$REPO_ROOT" 2>/dev/null && pwd -P) || REPO_ROOT=""
    fi
    if [ -n "$REPO_ROOT" ]; then
      case "$FILE_PATH" in
        /*) ABS_PATH="$FILE_PATH" ;;
        *)  ABS_PATH="$(pwd)/$FILE_PATH" ;;
      esac
      # Normalize the absolute path: resolve .. / . and symlinks. Prefer
      # python3 (ubiquitous on macOS/Linux); fall back to a dirname+pwd
      # trick which works whenever the parent directory exists (the common
      # case for Write/Edit since the parent must already exist).
      if command -v python3 >/dev/null 2>&1; then
        NORMALIZED=$(python3 -c 'import os,sys; print(os.path.realpath(sys.argv[1]))' "$ABS_PATH" 2>/dev/null || true)
      else
        NORMALIZED=""
        _dir=$(dirname -- "$ABS_PATH")
        _base=$(basename -- "$ABS_PATH")
        if [ -d "$_dir" ]; then
          NORMALIZED="$(cd -- "$_dir" && pwd -P)/$_base"
        fi
      fi
      [ -n "$NORMALIZED" ] && ABS_PATH="$NORMALIZED"
      case "$ABS_PATH" in
        "$REPO_ROOT"/*|"$REPO_ROOT")
          ;; # within repo, OK
        /tmp/*|/private/tmp/*|/var/folders/*|/private/var/folders/*)
          ;; # temp files (/var/folders is macOS TMPDIR; /private/var/folders
             # is its realpath-resolved form since /var is a symlink to
             # /private/var on macOS), OK
        *)
          jq -n --arg file "$FILE_PATH" --arg repo "$REPO_ROOT" '{
            hookSpecificOutput: {
              hookEventName: "PostToolUse",
              additionalContext: ("Note: file written outside repository root (" + $repo + "): " + $file)
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
