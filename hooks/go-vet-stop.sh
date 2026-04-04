#!/bin/bash
# devkit Stop hook — enforces go vet + race detector on Go changes
#
# When Go files were modified in the session, runs go vet and
# go test -race to catch concurrency bugs before session completes.
#
# Stop hook schema:
#   { "decision": "approve" | "block", "reason": "string" }

set -euo pipefail

# Check if any Go files were modified
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
GO_CHANGES=$(cd "$REPO_ROOT" && {
  git diff --name-only HEAD 2>/dev/null
  git diff --name-only --cached 2>/dev/null
  git diff --name-only 2>/dev/null
} | grep '\.go$' | sort -u)

if [ -z "$GO_CHANGES" ]; then
  jq -n '{ decision: "approve" }'
  exit 0
fi

# Find the Go module root (directory containing go.mod)
GO_MOD_DIR=""
for candidate in "$REPO_ROOT" "$REPO_ROOT/src" "$REPO_ROOT/cmd"; do
  if [ -f "$candidate/go.mod" ]; then
    GO_MOD_DIR="$candidate"
    break
  fi
done

if [ -z "$GO_MOD_DIR" ]; then
  # No go.mod found — can't run vet, approve and move on
  jq -n '{ decision: "approve" }'
  exit 0
fi

# Run go vet
VET_OUTPUT=$(cd "$GO_MOD_DIR" && go vet ./... 2>&1) || true
if [ -n "$VET_OUTPUT" ]; then
  jq -n --arg msg "go vet found issues in modified Go files. Fix before completing:\n$VET_OUTPUT" '{
    decision: "block",
    reason: $msg
  }'
  exit 0
fi

# Run go test -race on packages with changes (limited to 60s)
# Extract unique package directories from changed files
PACKAGES=""
while IFS= read -r file; do
  dir=$(dirname "$file")
  # Convert filesystem path to Go package path relative to module
  rel=$(echo "$dir" | sed "s|^${GO_MOD_DIR#$REPO_ROOT/}/||; s|^${GO_MOD_DIR#$REPO_ROOT/}$|.|")
  if [ "$rel" = "$dir" ]; then
    rel="./$(echo "$dir" | sed "s|^src/||")"
  fi
  PACKAGES="$PACKAGES ./$rel"
done <<< "$GO_CHANGES"
PACKAGES=$(echo "$PACKAGES" | tr ' ' '\n' | sort -u | tr '\n' ' ')

if [ -n "$PACKAGES" ]; then
  # Use perl alarm for POSIX-compatible timeout (macOS has no `timeout` command)
  RACE_OUTPUT=$(cd "$GO_MOD_DIR" && perl -e 'alarm 60; exec @ARGV' -- go test -race -count=1 $PACKAGES 2>&1) || RACE_EXIT=$?
  if [ "${RACE_EXIT:-0}" -ne 0 ]; then
    # Check if it's specifically a race condition
    if echo "$RACE_OUTPUT" | grep -qE 'DATA RACE|race detected'; then
      RACE_LINES=$(echo "$RACE_OUTPUT" | grep -A5 'DATA RACE' | head -20)
      jq -n --arg msg "Race condition detected in modified Go packages:\n$RACE_LINES" '{
        decision: "block",
        reason: $msg
      }'
      exit 0
    fi
    # Test failure but not a race — don't block on this hook (dirty-bit handles test coverage)
  fi
fi

jq -n '{ decision: "approve" }'
exit 0
