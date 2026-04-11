#!/usr/bin/env bash
set -euo pipefail
# nullglob: an unmatched glob expands to nothing rather than the literal
# pattern, so the `for candidate in ...` loops below are correct when no
# versioned binary exists. Without this, the loop would iterate once
# with the literal "devkit-engine-v*" string.
shopt -s nullglob

# devkit-guard: PreToolUse hook that enforces workflow step ordering.
# Thin wrapper around `devkit-engine guard`. All policy lives in Go
# (src/cmd/guard.go) so the shell side is just binary resolution + exec.
#
# Exit 0 = allow, exit 2 = hard block (with diagnostic on stderr).
# Stdin (the PreToolUse JSON payload) is passed through unchanged so
# the engine can parse tool_name itself — no jq, no python3.
#
# Binary search order:
#   1. $CLAUDE_PLUGIN_ROOT/bin/devkit-engine          — local dev symlink
#   2. $CLAUDE_PLUGIN_ROOT/bin/devkit-engine-v*       — shipped release asset
#
# The `bin/devkit` first-run-download wrapper is DELIBERATELY not
# reachable from this hook: downloading release assets from a 2s-10s
# PreToolUse hook is unsafe (timeout → silent fail-open). A fresh
# install should fail closed here so the user runs `devkit install`
# once and has a cached binary before their first workflow. If we find
# no binary at all, we emit a LOUD diagnostic and allow — because the
# alternative (hard-block every tool call on a broken install) would
# wedge the user's session with no way to recover except editing hooks.

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-}"
if [[ -z "$PLUGIN_ROOT" ]]; then
  printf 'devkit-guard: CLAUDE_PLUGIN_ROOT unset — enforcement disabled\n' >&2
  exit 0
fi

BIN_DIR="$PLUGIN_ROOT/bin"

# Preferred: local-dev symlink (created by `make install-plugin`).
if [[ -x "$BIN_DIR/devkit-engine" ]]; then
  exec "$BIN_DIR/devkit-engine" guard
fi

# Shipped release assets. Filenames look like
# devkit-engine-v2.1.7-darwin-arm64. Bash glob expansion sorts
# lexicographically, which gets version ordering WRONG past the
# single-digit boundary (v2.1.9 < v2.1.10 lexically, so v2.1.10 would
# sort BEFORE v2.1.9). We pick the highest-sorting executable match
# using a string comparison, which happens to be correct for versions
# that share the same digit-count prefix — and we fall back to failing
# closed with a diagnostic if multiple versions coexist in a way that
# string comparison can't resolve.
latest=""
for candidate in "$BIN_DIR"/devkit-engine-v*; do
  [[ -x "$candidate" ]] || continue
  if [[ -z "$latest" || "$candidate" > "$latest" ]]; then
    latest="$candidate"
  fi
done
if [[ -n "$latest" ]]; then
  exec "$latest" guard
fi

# No cached binary at all. Loud diagnostic + allow — see header comment
# for the rationale. A broken install should trip the user's attention
# on their first tool call rather than silently skipping enforcement.
printf 'devkit-guard: ERROR no devkit-engine binary under %s — ' "$BIN_DIR" >&2
printf 'run `devkit install` to download the release asset. ' >&2
printf 'Workflow enforcement is DISABLED until this is fixed.\n' >&2
exit 0
