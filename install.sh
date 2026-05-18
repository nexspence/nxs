#!/bin/sh
set -e

REPO="nexspence/nxs"
BINARY="nxs"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)          ARCH="amd64" ;;
  aarch64|arm64)   ARCH="arm64" ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

VERSION=$(curl -sSf "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | cut -d'"' -f4)

TARBALL="${BINARY}_${VERSION#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$TARBALL"

echo "Installing nxs $VERSION..."
curl -sSfL "$URL" | tar -xz -C /usr/local/bin "$BINARY"
chmod +x /usr/local/bin/$BINARY
echo "Done. Run: nxs --help"
