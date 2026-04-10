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

if [[ -f "$INSTALLER" ]]; then
  bash "$INSTALLER"
else
  # Fallback: search plugin cache
  INSTALLER=$(find ~/.claude/plugins -path '*/devkit/scripts/install-engine.sh' 2>/dev/null | head -1)
  if [[ -n "$INSTALLER" ]]; then
    bash "$INSTALLER"
  else
    printf "Cannot find install-engine.sh.\n"
    printf "Install manually: https://github.com/5uck1ess/devkit/releases\n"
    exit 1
  fi
fi
