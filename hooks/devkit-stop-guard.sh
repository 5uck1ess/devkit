#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob

# devkit-stop-guard: Stop hook that blocks session end during active
# workflows. Thin wrapper around `devkit-engine guard --stop`. Emits
# JSON ({"decision":"approve"} or {"decision":"block","reason":"..."})
# on stdout and always exits 0 — Stop hooks communicate via verdict
# payload, not exit code.
#
# Binary search order and no-binary fallback rationale: see
# devkit-guard.sh header comment.

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-}"
if [[ -z "$PLUGIN_ROOT" ]]; then
  printf 'devkit-stop-guard: CLAUDE_PLUGIN_ROOT unset — enforcement disabled\n' >&2
  printf '{"decision":"approve"}'
  exit 0
fi

BIN_DIR="$PLUGIN_ROOT/bin"

if [[ -x "$BIN_DIR/devkit-engine" ]]; then
  exec "$BIN_DIR/devkit-engine" guard --stop
fi

# Pick the highest semver via sort -V. See devkit-guard.sh for the
# rationale on why a naive string comparison is wrong (v2.1.9 would
# lexicographically sort above v2.1.10).
candidates=()
for candidate in "$BIN_DIR"/devkit-engine-v*; do
  [[ -x "$candidate" ]] && candidates+=("$candidate")
done
if (( ${#candidates[@]} > 0 )); then
  latest=$(printf '%s\n' "${candidates[@]}" | sort -V | tail -n1)
  if [[ -n "$latest" && -x "$latest" ]]; then
    exec "$latest" guard --stop
  fi
fi

# No engine binary: approve so the user is never wedged in an
# un-stoppable session. Loud stderr so the broken install is visible.
# Point at the real self-downloader at $BIN_DIR/devkit, not a
# non-existent `devkit install` subcommand.
printf 'devkit-stop-guard: ERROR no devkit-engine binary under %s — ' "$BIN_DIR" >&2
printf 'run `%s/devkit --version` once to cache the engine. ' "$BIN_DIR" >&2
printf 'Stop enforcement DISABLED.\n' >&2
printf '{"decision":"approve"}'
exit 0
