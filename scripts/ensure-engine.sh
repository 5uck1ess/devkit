#!/usr/bin/env bash
# Ensure the devkit engine binary is on PATH.
# Finds and runs install-engine.sh if devkit is not installed.
# Called by commands and skills before `devkit workflow run`.

set -euo pipefail

if command -v devkit >/dev/null 2>&1; then
  exit 0
fi

# Find install-engine.sh relative to this script (works when called from plugin cache)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALLER="${SCRIPT_DIR}/install-engine.sh"

if [[ ! -f "$INSTALLER" ]]; then
  # Fallback: search plugin cache (Unix: ~/.claude, Windows: $APPDATA/.claude)
  for PLUGIN_ROOT in "$HOME/.claude/plugins" "${APPDATA:+$APPDATA/.claude/plugins}" "${LOCALAPPDATA:+$LOCALAPPDATA/.claude/plugins}"; do
    [[ -z "$PLUGIN_ROOT" ]] && continue
    [[ -d "$PLUGIN_ROOT" ]] || continue
    INSTALLER=$(find "$PLUGIN_ROOT" -path '*/devkit/scripts/install-engine.sh' 2>/dev/null | head -1)
    [[ -n "$INSTALLER" ]] && break
  done
fi

if [[ -z "$INSTALLER" ]] || [[ ! -f "$INSTALLER" ]]; then
  printf "Cannot find install-engine.sh.\n"
  printf "Install manually: https://github.com/5uck1ess/devkit/releases\n"
  exit 1
fi

# Source instead of subprocess so PATH exports propagate
# shellcheck disable=SC1090
source "$INSTALLER"

# Verify devkit is now available (handles ~/.local/bin PATH addition)
if ! command -v devkit >/dev/null 2>&1; then
  # Last resort: check common install locations directly
  for dir in /usr/local/bin "$HOME/.local/bin" "${LOCALAPPDATA:-}/devkit"; do
    if [[ -x "${dir}/devkit" ]]; then
      export PATH="${dir}:${PATH}"
      break
    fi
  done
  if ! command -v devkit >/dev/null 2>&1; then
    printf "devkit installed but not on PATH. Add the install directory to your PATH.\n"
    exit 1
  fi
fi
