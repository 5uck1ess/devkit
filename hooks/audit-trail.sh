#!/bin/bash
# devkit audit trail hook — logs all Bash commands with timestamps
# Runs on PreToolUse for Bash tool
#
# Log location: .devkit/audit.log (gitignored)
# Format: ISO-8601 timestamp | working directory | command

set -euo pipefail

# Observational hook — jq parse failures degrade to "nothing to log",
# never to a hook error (fail-open contract, see rtk-rewrite.sh).
INPUT=$(cat || true)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty' 2>/dev/null || true)

# Skip if no command
[ -z "$COMMAND" ] && exit 0

# Ensure log directory exists
LOG_DIR=".devkit"
LOG_FILE="${LOG_DIR}/audit.log"

# First-run only: self-install .devkit/ in the nearest git repo's .gitignore
# so devkit's audit log never gets accidentally tracked. Without this, users
# whose project .gitignore doesn't already cover .devkit/ end up tracking
# audit.log, which then conflicts on stash/pull/merge because the hook
# rewrites it on every Bash call.
#
# Race safety: Claude Code can fire PreToolUse hooks in parallel when the
# model issues parallel Bash calls. We use an atomic noclobber marker inside
# .devkit/ as the init lock so only the first process does the gitignore
# work; concurrent callers fail the noclobber and skip it cleanly.
#
# Submodule note: git rev-parse --show-toplevel intentionally returns the
# submodule root when cwd is inside one — this is correct because .devkit/
# is created relative to cwd and therefore lives inside the submodule, so
# the submodule's own .gitignore is the right file to update.
INIT_MARKER="${LOG_DIR}/.gitignore-installed"
if [[ ! -f "$INIT_MARKER" ]]; then
  mkdir -p "$LOG_DIR" 2>/dev/null || true
  if ( set -C; : > "$INIT_MARKER" ) 2>/dev/null; then
    REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || true)
    if [[ -n "$REPO_ROOT" ]]; then
      GITIGNORE="${REPO_ROOT}/.gitignore"
      if [[ ! -f "$GITIGNORE" ]] || ! grep -qE '^\.devkit($|/)' "$GITIGNORE" 2>/dev/null; then
        if [[ -f "$GITIGNORE" ]] && [[ -n "$(tail -c 1 "$GITIGNORE" 2>/dev/null)" ]]; then
          printf '\n' >> "$GITIGNORE" 2>/dev/null || true
        fi
        printf '.devkit/\n' >> "$GITIGNORE" 2>/dev/null || true
      fi
    fi
  fi
fi

# If the log directory can't be created there is nothing to log into —
# exit clean rather than erroring on every Bash call.
mkdir -p "$LOG_DIR" 2>/dev/null || exit 0

# Truncate command for log (first line only, max 500 chars)
CMD_SHORT=$(echo "$COMMAND" | head -1 | cut -c1-500)

# Append timestamped entry. Tolerate write failures (read-only fs,
# permissions) — losing one audit line must not error every Bash call.
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) | $(pwd) | ${CMD_SHORT}" >> "$LOG_FILE" 2>/dev/null || true

# Rotate if log exceeds 10k lines
if [ -f "$LOG_FILE" ] && [ "$(wc -l < "$LOG_FILE" 2>/dev/null || echo 0)" -gt 10000 ]; then
  { tail -5000 "$LOG_FILE" > "${LOG_FILE}.tmp" && mv "${LOG_FILE}.tmp" "$LOG_FILE"; } 2>/dev/null || true
fi

# Always allow — this hook is observational only
exit 0
