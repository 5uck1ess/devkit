#!/usr/bin/env bash
set -euo pipefail
# nullglob: an unmatched glob expands to nothing rather than the literal
# pattern, so the array-glob construct below is correct when no
# versioned binary exists.
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
# reachable from this hook: downloading release assets from a
# time-limited PreToolUse hook is unsafe (timeout → silent fail-open).
# When no binary is found we fail OPEN with a LOUD diagnostic so the
# user notices on their first tool call — blocking every tool call on
# a broken install would wedge the session with no recovery path
# except manually editing hooks.

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
# devkit-engine-v2.1.7-darwin-arm64. Pick the highest semver via
# `sort -V` (GNU coreutils; available on Ubuntu runners and recent
# macOS). A naive string comparison would pick v2.1.9 over v2.1.10
# because `9 > 1` lexicographically — sort -V understands version
# fields and orders them correctly.
candidates=()
for candidate in "$BIN_DIR"/devkit-engine-v*; do
  [[ -x "$candidate" ]] && candidates+=("$candidate")
done
if (( ${#candidates[@]} > 0 )); then
  latest=$(printf '%s\n' "${candidates[@]}" | sort -V | tail -n1)
  if [[ -n "$latest" && -x "$latest" ]]; then
    exec "$latest" guard
  fi
fi

# No cached binary at all. Loud diagnostic + allow — see header
# comment for the fail-open rationale. Point the user at the real
# self-downloader at $BIN_DIR/devkit (that wrapper handles the
# download + verify + cache flow on first run). There is no
# `devkit install` subcommand — `devkit --version` triggers the same
# cache-if-missing path with zero side effects.
printf 'devkit-guard: ERROR no devkit-engine binary under %s — ' "$BIN_DIR" >&2
printf 'run `%s/devkit --version` once to download and cache the engine. ' "$BIN_DIR" >&2
printf 'Workflow enforcement is DISABLED until this is fixed.\n' >&2
exit 0
