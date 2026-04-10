#!/usr/bin/env bash
# Install the devkit engine binary from GitHub releases.
# Called by skills/commands before running `devkit workflow run`.
#
# Usage: ./scripts/install-engine.sh [--check]
#   --check   Exit 0 if devkit is already on PATH, 1 if not (no install)

set -euo pipefail

REPO="5uck1ess/devkit"
BINARY="devkit"
INSTALL_DIR="${DEVKIT_INSTALL_DIR:-/usr/local/bin}"

# --check mode: just test if binary exists
if [ "${1:-}" = "--check" ]; then
  command -v "$BINARY" >/dev/null 2>&1
  exit $?
fi

# Skip if already installed
if command -v "$BINARY" >/dev/null 2>&1; then
  echo "devkit engine already installed: $(command -v "$BINARY")"
  exit 0
fi

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

EXT=""
case "$OS" in
  linux|darwin) ;;
  mingw*|msys*|cygwin*) OS="windows"; EXT=".exe" ;;
  *)                    printf "Unsupported OS: %s\n" "$OS"; exit 1 ;;
esac

ASSET="${BINARY}-${OS}-${ARCH}${EXT}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

printf "Downloading devkit engine (%s/%s)...\n" "$OS" "$ARCH"
TMPFILE="$(mktemp)"
trap 'rm -f "$TMPFILE"' EXIT

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$TMPFILE"
elif command -v wget >/dev/null 2>&1; then
  wget -q "$URL" -O "$TMPFILE"
else
  printf "Error: curl or wget required\n"
  exit 1
fi

chmod +x "$TMPFILE"

# Windows: install to user's local bin
if [[ "$OS" == "windows" ]]; then
  WIN_DIR="${LOCALAPPDATA:-$HOME/AppData/Local}/devkit"
  mkdir -p "$WIN_DIR"
  mv "$TMPFILE" "${WIN_DIR}/${BINARY}${EXT}"
  printf "Installed to %s/%s%s\n" "$WIN_DIR" "$BINARY" "$EXT"
  printf "Add to PATH: %s\n" "$WIN_DIR"
  exit 0
fi

# Unix: try INSTALL_DIR, fall back to ~/.local/bin
if [[ -w "$INSTALL_DIR" ]]; then
  mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
  printf "Installed to %s/%s\n" "$INSTALL_DIR" "$BINARY"
elif mkdir -p "$HOME/.local/bin" 2>/dev/null; then
  mv "$TMPFILE" "$HOME/.local/bin/${BINARY}"
  printf "Installed to %s/.local/bin/%s\n" "$HOME" "$BINARY"
  printf "Add to PATH: export PATH=\"\$HOME/.local/bin:\$PATH\"\n"
else
  printf "Cannot write to %s or ~/.local/bin\n" "$INSTALL_DIR"
  printf "Run with sudo or set DEVKIT_INSTALL_DIR to a writable directory\n"
  exit 1
fi
