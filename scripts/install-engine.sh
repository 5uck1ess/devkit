#!/usr/bin/env bash
# Install the devkit engine binary from GitHub releases.
# Called by skills/commands before running `devkit workflow run`.
#
# Usage: ./scripts/install-engine.sh [--check] [--upgrade]
#   --check    Exit 0 if devkit is on PATH, 1 if not (no install)
#   --upgrade  Re-download even if devkit is already installed

set -euo pipefail

REPO="5uck1ess/devkit"
BINARY="devkit"
INSTALL_DIR="${DEVKIT_INSTALL_DIR:-/usr/local/bin}"

# --check mode: just test if binary exists
if [[ "${1:-}" == "--check" ]]; then
  command -v "$BINARY" >/dev/null 2>&1
  exit $?
fi

# Skip if already installed (unless --upgrade)
if [[ "${1:-}" != "--upgrade" ]] && command -v "$BINARY" >/dev/null 2>&1; then
  printf "devkit engine already installed: %s\n" "$(command -v "$BINARY")"
  exit 0
fi

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       printf "Unsupported architecture: %s\n" "$ARCH"; exit 1 ;;
esac

EXT=""
case "$OS" in
  linux|darwin) ;;
  # Windows: only reachable via MSYS2, Git Bash, or Cygwin
  mingw*|msys*|cygwin*) OS="windows"; EXT=".exe" ;;
  *)                    printf "Unsupported OS: %s\n" "$OS"; exit 1 ;;
esac

ASSET="${BINARY}-${OS}-${ARCH}${EXT}"
BASE_URL="https://github.com/${REPO}/releases/latest/download"

printf "Downloading devkit engine (%s/%s)...\n" "$OS" "$ARCH"
TMPDIR_CLEAN="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_CLEAN"' EXIT
TMPFILE="${TMPDIR_CLEAN}/${ASSET}"
TMPCHECKSUM="${TMPDIR_CLEAN}/checksums.txt"

# Download binary — use -fSL (capital S keeps error messages visible)
if command -v curl >/dev/null 2>&1; then
  curl -fSL "${BASE_URL}/${ASSET}" -o "$TMPFILE" || { printf "Download failed: %s/%s\nCheck network and that the release exists.\n" "$BASE_URL" "$ASSET"; exit 1; }
  curl -fSL "${BASE_URL}/checksums.txt" -o "$TMPCHECKSUM" || { printf "Warning: could not download checksums.txt\n"; TMPCHECKSUM=""; }
elif command -v wget >/dev/null 2>&1; then
  wget -q "${BASE_URL}/${ASSET}" -O "$TMPFILE" || { printf "Download failed: %s/%s\nCheck network and that the release exists.\n" "$BASE_URL" "$ASSET"; exit 1; }
  wget -q "${BASE_URL}/checksums.txt" -O "$TMPCHECKSUM" || { printf "Warning: could not download checksums.txt\n"; TMPCHECKSUM=""; }
else
  printf "Error: curl or wget required\n"
  exit 1
fi

# Validate download is non-empty
if [[ ! -s "$TMPFILE" ]]; then
  printf "Downloaded file is empty — release may not exist for %s/%s\n" "$OS" "$ARCH"
  exit 1
fi

# Verify checksum if checksums.txt was downloaded
if [[ -n "$TMPCHECKSUM" ]] && [[ -s "$TMPCHECKSUM" ]]; then
  EXPECTED=$(awk -v asset="$ASSET" '$2 == asset || $2 == "./"asset {print $1}' "$TMPCHECKSUM" | head -1)
  if [[ -n "$EXPECTED" ]]; then
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL=$(sha256sum "$TMPFILE" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL=$(shasum -a 256 "$TMPFILE" | awk '{print $1}')
    else
      printf "Warning: cannot verify checksum (no sha256sum or shasum)\n"
      ACTUAL="$EXPECTED"
    fi
    if [[ "$EXPECTED" != "$ACTUAL" ]]; then
      printf "Checksum mismatch! Expected %s, got %s\n" "$EXPECTED" "$ACTUAL"
      exit 1
    fi
    printf "Checksum verified.\n"
  fi
else
  printf "Warning: checksums.txt not available — skipping integrity check\n"
fi

chmod +x "$TMPFILE"

# Windows: install to user's local bin (MSYS2/Git Bash/Cygwin only)
if [[ "$OS" == "windows" ]]; then
  WIN_DIR="${LOCALAPPDATA:-$HOME/AppData/Local}/devkit"
  mkdir -p "$WIN_DIR"
  mv "$TMPFILE" "${WIN_DIR}/${BINARY}${EXT}" || { printf "Failed to install to %s\n" "$WIN_DIR"; exit 1; }
  printf "Installed to %s/%s%s\n" "$WIN_DIR" "$BINARY" "$EXT"
  printf "Add to PATH: %s\n" "$WIN_DIR"
  exit 0
fi

# Unix: try INSTALL_DIR, fall back to ~/.local/bin
if [[ -w "$INSTALL_DIR" ]]; then
  mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}" || { printf "Failed to install to %s\n" "$INSTALL_DIR"; exit 1; }
  printf "Installed to %s/%s\n" "$INSTALL_DIR" "$BINARY"
elif mkdir -p "$HOME/.local/bin"; then
  mv "$TMPFILE" "$HOME/.local/bin/${BINARY}" || { printf "Failed to install to %s/.local/bin\n" "$HOME"; exit 1; }
  printf "Installed to %s/.local/bin/%s\n" "$HOME" "$BINARY"
  # Make binary available in current session
  export PATH="$HOME/.local/bin:$PATH"
  if ! command -v "$BINARY" >/dev/null 2>&1; then
    printf "Warning: installed to ~/.local/bin but it's not on PATH.\n"
    printf "Add to your shell profile: export PATH=\"\$HOME/.local/bin:\$PATH\"\n"
  fi
else
  printf "Cannot write to %s or ~/.local/bin\n" "$INSTALL_DIR"
  printf "Run with sudo or set DEVKIT_INSTALL_DIR to a writable directory\n"
  exit 1
fi
