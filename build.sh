#!/usr/bin/env bash
#
# curio — build & install script.
# Cross-compiles for 4 platforms, copies the current platform's binary
# to the skill directory, and writes skill docs via `curio skills install`.
#
set -euo pipefail

VERSION="${VERSION:-dev}"
SKILL_DIR="${SKILL_DIR:-$HOME/.agents/skills/curio}"
DIST_DIR="dist"
BIN_DIR="bin"

PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
)

LDFLAGS="-s -w -X main.version=${VERSION}"

echo "Building curio ${VERSION}..."

# Detect git tag if VERSION is still "dev"
if [[ "$VERSION" == "dev" ]] && command -v git >/dev/null 2>&1; then
  TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
  if [[ -n "$TAG" ]]; then
    VERSION="$TAG"
    LDFLAGS="-s -w -X main.version=${VERSION}"
    echo "Using git tag: ${VERSION}"
  fi
fi

mkdir -p "$DIST_DIR" "$BIN_DIR"

for PLATFORM in "${PLATFORMS[@]}"; do
  GOOS="${PLATFORM%/*}"
  GOARCH="${PLATFORM#*/}"
  OUTPUT="${DIST_DIR}/curio-${GOOS}-${GOARCH}"

  echo "  → ${GOOS}/${GOARCH}"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -trimpath -ldflags "$LDFLAGS" \
    -o "$OUTPUT" ./cmd/curio

  chmod +x "$OUTPUT"
done

echo ""
echo "Built binaries:"
ls -lh "$DIST_DIR"/curio-* | awk '{printf "  %s  %s\n", $5, $9}'

# Install current platform's binary to skill dir
CURRENT_OS="$(go env GOOS)"
CURRENT_ARCH="$(go env GOARCH)"
CURRENT_BIN="${DIST_DIR}/curio-${CURRENT_OS}-${CURRENT_ARCH}"

if [[ -f "$CURRENT_BIN" ]]; then
  echo ""
  echo "Installing to ${SKILL_DIR}/..."
  mkdir -p "$SKILL_DIR"
  cp "$CURRENT_BIN" "${SKILL_DIR}/curio"
  chmod +x "${SKILL_DIR}/curio"

  # Write skill docs
  "${SKILL_DIR}/curio" skills install --dir "$SKILL_DIR"

  echo ""
  "${SKILL_DIR}/curio" version
  echo ""
  echo "✓ Installed. Test: ${SKILL_DIR}/curio \"cats\" -n 2"
fi
