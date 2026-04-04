#!/bin/bash
# devkit Stop hook — consolidated quality gate
#
# Replaces dirty-bit.sh + go-vet-stop.sh + old stop-gate.sh with a single hook.
#
# Phase 1: Basic checks (uncommitted changes, conflict markers, TODOs)
# Phase 2: Cross-domain test evidence (dirty-bit logic)
# Phase 3: Language-specific linter/vet (go vet, clippy, tsc, ruff)
#
# Stop hook schema:
#   { "decision": "approve" | "block", "reason": "string" }

set -euo pipefail

INPUT=$(cat)
TRANSCRIPT=$(echo "$INPUT" | jq -r '.transcript // empty')

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
CHANGED_FILES=$(cd "$REPO_ROOT" && {
  git diff --name-only HEAD 2>/dev/null
  git diff --name-only --cached 2>/dev/null
  git diff --name-only 2>/dev/null
} | sort -u)

if [ -z "$CHANGED_FILES" ]; then
  jq -n '{ decision: "approve" }'
  exit 0
fi

# ---------------------------------------------------------------------------
# Phase 1: Basic quality checks
# ---------------------------------------------------------------------------

# Merge conflict markers
CONFLICT_PATTERN='<''<<''<<''<< '
CONFLICTS=$(echo "$CHANGED_FILES" | while IFS= read -r f; do
  [ -f "$REPO_ROOT/$f" ] && grep -l -- "$CONFLICT_PATTERN" "$REPO_ROOT/$f" 2>/dev/null || true
done | head -3)
if [ -n "$CONFLICTS" ]; then
  jq -n --arg files "$CONFLICTS" '{
    decision: "block",
    reason: ("Merge conflict markers found in: " + $files)
  }'
  exit 0
fi

# ---------------------------------------------------------------------------
# Phase 2: Classify domains and check cross-domain test evidence
# ---------------------------------------------------------------------------
HAS_GO=false
HAS_TS=false
HAS_RUST=false
HAS_PYTHON=false
HAS_CONFIG=false
HAS_SQL=false

while IFS= read -r file; do
  case "$file" in
    *.go)                                HAS_GO=true ;;
    *.ts|*.tsx|*.js|*.jsx|*.mjs|*.cjs)   HAS_TS=true ;;
    *.rs)                                HAS_RUST=true ;;
    *.py)                                HAS_PYTHON=true ;;
    *.yml|*.yaml|*.json|*.toml|*.ini|*.env*) HAS_CONFIG=true ;;
    *.sql|*/migrations/*|*/migrate/*)    HAS_SQL=true ;;
  esac
done <<< "$CHANGED_FILES"

# Count code domains (exclude config — doesn't need its own test evidence)
CODE_DOMAINS=0
DOMAINS=""
$HAS_GO && CODE_DOMAINS=$((CODE_DOMAINS + 1)) && DOMAINS="$DOMAINS go"
$HAS_TS && CODE_DOMAINS=$((CODE_DOMAINS + 1)) && DOMAINS="$DOMAINS typescript"
$HAS_RUST && CODE_DOMAINS=$((CODE_DOMAINS + 1)) && DOMAINS="$DOMAINS rust"
$HAS_PYTHON && CODE_DOMAINS=$((CODE_DOMAINS + 1)) && DOMAINS="$DOMAINS python"
$HAS_SQL && CODE_DOMAINS=$((CODE_DOMAINS + 1)) && DOMAINS="$DOMAINS sql"
DOMAINS=$(echo "$DOMAINS" | xargs)

if [ "$CODE_DOMAINS" -gt 1 ]; then
  MISSING=""

  $HAS_GO && ! echo "$TRANSCRIPT" | grep -qiE '(go test|ALL_PASSING|ALL_TESTS_PASSING)' && MISSING="$MISSING go"
  $HAS_TS && ! echo "$TRANSCRIPT" | grep -qiE '(npm test|npx jest|npx vitest|yarn test|pnpm test|ALL_PASSING)' && MISSING="$MISSING typescript"
  $HAS_RUST && ! echo "$TRANSCRIPT" | grep -qiE '(cargo test|ALL_PASSING)' && MISSING="$MISSING rust"
  $HAS_PYTHON && ! echo "$TRANSCRIPT" | grep -qiE '(pytest|python.*-m.*test|ALL_PASSING)' && MISSING="$MISSING python"
  $HAS_SQL && ! echo "$TRANSCRIPT" | grep -qiE '(migrate|migration.*up|ALL_PASSING)' && MISSING="$MISSING sql"

  MISSING=$(echo "$MISSING" | xargs)
  if [ -n "$MISSING" ]; then
    jq -n --arg domains "$DOMAINS" --arg missing "$MISSING" '{
      decision: "block",
      reason: ("Cross-domain changes (touched: " + $domains + "). Missing test evidence for: " + $missing)
    }'
    exit 0
  fi
fi

# ---------------------------------------------------------------------------
# Phase 3: Language-specific vet/lint
# ---------------------------------------------------------------------------

# --- Go ---
if $HAS_GO; then
  GO_MOD_DIR=""
  for candidate in "$REPO_ROOT" "$REPO_ROOT/src" "$REPO_ROOT/cmd"; do
    [ -f "$candidate/go.mod" ] && GO_MOD_DIR="$candidate" && break
  done

  if [ -n "$GO_MOD_DIR" ]; then
    VET_OUTPUT=$(cd "$GO_MOD_DIR" && go vet ./... 2>&1) || true
    if [ -n "$VET_OUTPUT" ]; then
      jq -n --arg msg "go vet found issues:\n$VET_OUTPUT" '{ decision: "block", reason: $msg }'
      exit 0
    fi

    # Race detection on changed packages
    GO_PKGS=$(echo "$CHANGED_FILES" | grep '\.go$' | while IFS= read -r f; do
      dirname "$f" | sed "s|^src/|./|"
    done | sort -u | tr '\n' ' ')

    if [ -n "$GO_PKGS" ]; then
      RACE_OUTPUT=$(cd "$GO_MOD_DIR" && perl -e 'alarm 60; exec @ARGV' -- go test -race -count=1 $GO_PKGS 2>&1) || RACE_EXIT=$?
      if [ "${RACE_EXIT:-0}" -ne 0 ] && echo "$RACE_OUTPUT" | grep -qE 'DATA RACE|race detected'; then
        RACE_LINES=$(echo "$RACE_OUTPUT" | grep -A5 'DATA RACE' | head -20)
        jq -n --arg msg "Race condition detected:\n$RACE_LINES" '{ decision: "block", reason: $msg }'
        exit 0
      fi
    fi
  fi
fi

# --- Rust ---
if $HAS_RUST && command -v cargo >/dev/null 2>&1 && [ -f "$REPO_ROOT/Cargo.toml" ]; then
  CLIPPY_OUTPUT=$(cd "$REPO_ROOT" && cargo clippy --quiet 2>&1) || true
  if echo "$CLIPPY_OUTPUT" | grep -qE 'error\['; then
    CLIPPY_ERRORS=$(echo "$CLIPPY_OUTPUT" | grep -E 'error\[' | head -5)
    jq -n --arg msg "cargo clippy errors:\n$CLIPPY_ERRORS" '{ decision: "block", reason: $msg }'
    exit 0
  fi
fi

# --- TypeScript ---
if $HAS_TS && [ -f "$REPO_ROOT/tsconfig.json" ] && command -v npx >/dev/null 2>&1; then
  TSC_OUTPUT=$(cd "$REPO_ROOT" && npx tsc --noEmit 2>&1) || TSC_EXIT=$?
  if [ "${TSC_EXIT:-0}" -ne 0 ]; then
    TSC_ERRORS=$(echo "$TSC_OUTPUT" | grep -E 'error TS' | head -5)
    if [ -n "$TSC_ERRORS" ]; then
      jq -n --arg msg "TypeScript errors:\n$TSC_ERRORS" '{ decision: "block", reason: $msg }'
      exit 0
    fi
  fi
fi

# --- Python ---
if $HAS_PYTHON && command -v ruff >/dev/null 2>&1; then
  PY_FILES=$(echo "$CHANGED_FILES" | grep '\.py$' || true)
  if [ -n "$PY_FILES" ]; then
    RUFF_OUTPUT=$(cd "$REPO_ROOT" && echo "$PY_FILES" | xargs ruff check 2>&1) || true
    if echo "$RUFF_OUTPUT" | grep -qE '^[^ ]+\.py:[0-9]+'; then
      RUFF_ERRORS=$(echo "$RUFF_OUTPUT" | head -5)
      jq -n --arg msg "ruff found issues:\n$RUFF_ERRORS" '{ decision: "block", reason: $msg }'
      exit 0
    fi
  fi
fi

jq -n '{ decision: "approve" }'
exit 0
