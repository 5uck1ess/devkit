#!/bin/bash
# devkit Stop hook — dirty-bit feedback loop
#
# Tracks which domains were modified and warns if the appropriate
# verification (tests/lint) hasn't been run for each touched domain.
#
# Domains: backend (Go/Python), frontend (TS/JS/JSX/TSX), config (YAML/JSON/TOML),
#          test files, SQL/migrations
#
# Stop hook schema:
#   { "decision": "approve" | "block", "reason": "string" }

set -euo pipefail

INPUT=$(cat)
TRANSCRIPT=$(echo "$INPUT" | jq -r '.transcript // empty')

# Get modified files from git
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
CHANGED_FILES=$(cd "$REPO_ROOT" && git diff --name-only HEAD 2>/dev/null; git diff --name-only --cached 2>/dev/null; git diff --name-only 2>/dev/null)

if [ -z "$CHANGED_FILES" ]; then
  jq -n '{ decision: "approve" }'
  exit 0
fi

# Classify changed files into domains
BACKEND=false
FRONTEND=false
CONFIG=false
SQL=false
DOMAINS_TOUCHED=""

while IFS= read -r file; do
  case "$file" in
    *.go|*.py|*.rb|*.java|*.rs)
      BACKEND=true ;;
    *.ts|*.tsx|*.js|*.jsx|*.vue|*.svelte)
      FRONTEND=true ;;
    *.yml|*.yaml|*.json|*.toml|*.ini|*.env*)
      CONFIG=true ;;
    *.sql|**/migrations/*|**/migrate/*)
      SQL=true ;;
  esac
done <<< "$CHANGED_FILES"

# Build list of touched domains
if [ "$BACKEND" = "true" ]; then DOMAINS_TOUCHED="$DOMAINS_TOUCHED backend"; fi
if [ "$FRONTEND" = "true" ]; then DOMAINS_TOUCHED="$DOMAINS_TOUCHED frontend"; fi
if [ "$CONFIG" = "true" ]; then DOMAINS_TOUCHED="$DOMAINS_TOUCHED config"; fi
if [ "$SQL" = "true" ]; then DOMAINS_TOUCHED="$DOMAINS_TOUCHED sql"; fi

# If only one domain or no code domains, approve
DOMAIN_COUNT=$(echo "$DOMAINS_TOUCHED" | wc -w | tr -d ' ')
if [ "$DOMAIN_COUNT" -le 1 ]; then
  jq -n '{ decision: "approve" }'
  exit 0
fi

# Multiple domains touched — check for test evidence per domain
MISSING_VERIFICATION=""

if [ "$BACKEND" = "true" ]; then
  if ! echo "$TRANSCRIPT" | grep -qiE '(go test|pytest|python.*test|cargo test|bundle exec.*test|ALL_PASSING|ALL_TESTS_PASSING)'; then
    MISSING_VERIFICATION="$MISSING_VERIFICATION backend"
  fi
fi

if [ "$FRONTEND" = "true" ]; then
  if ! echo "$TRANSCRIPT" | grep -qiE '(npm test|npx jest|npx vitest|yarn test|pnpm test|ALL_PASSING|ALL_TESTS_PASSING)'; then
    MISSING_VERIFICATION="$MISSING_VERIFICATION frontend"
  fi
fi

if [ "$SQL" = "true" ]; then
  if ! echo "$TRANSCRIPT" | grep -qiE '(migrate|migration.*up|schema.*applied|ALL_PASSING)'; then
    MISSING_VERIFICATION="$MISSING_VERIFICATION sql/migrations"
  fi
fi

# If everything verified, approve
if [ -z "$MISSING_VERIFICATION" ]; then
  jq -n '{ decision: "approve" }'
  exit 0
fi

# Multiple domains touched, some unverified — block
DOMAINS_MSG=$(echo "$DOMAINS_TOUCHED" | xargs)
MISSING_MSG=$(echo "$MISSING_VERIFICATION" | xargs)

jq -n --arg domains "$DOMAINS_MSG" --arg missing "$MISSING_MSG" '{
  decision: "block",
  reason: ("Cross-domain changes detected (touched: " + $domains + "). Missing test/verification evidence for: " + $missing + ". Run the relevant test suite for each domain before completing.")
}'
exit 0
