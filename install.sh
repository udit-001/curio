#!/usr/bin/env bash
#
# curio — install script.
# Downloads the latest binary from GitHub Releases and installs it
# to the skill directory with SKILL.md.
#
set -euo pipefail

REPO="udit-001/curio"
SKILL_DIR="${SKILL_DIR:-$HOME/.agents/skills/curio}"

# Detect platform
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux)  OS="linux"  ;;
  darwin) OS="darwin" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

BINARY="curio-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${BINARY}"

echo "Downloading curio for ${OS}/${ARCH}..."
mkdir -p "$SKILL_DIR"

curl -sSL -o "${SKILL_DIR}/curio" "$URL"
chmod +x "${SKILL_DIR}/curio"

echo "Writing skill docs..."
"${SKILL_DIR}/curio" skills install --dir "$SKILL_DIR"

echo ""
"${SKILL_DIR}/curio" version
echo ""
echo "✓ Installed to ${SKILL_DIR}/"
echo "  Test: ${SKILL_DIR}/curio \"cats\" -n 2"
