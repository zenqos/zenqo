#!/bin/sh
set -e

REPO="zenqos/zenqo"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    echo "Please download manually from https://github.com/$REPO/releases"
    exit 1
    ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    echo "Please download manually from https://github.com/$REPO/releases"
    exit 1
    ;;
esac

# Fetch latest version
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)

if [ -z "$VERSION" ]; then
  echo "Failed to fetch latest version. Check your internet connection."
  exit 1
fi

FILENAME="zenqo_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

echo "Installing zenqo $VERSION ($OS/$ARCH)..."

# Download and extract
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "$TMP/$FILENAME"
tar -xzf "$TMP/$FILENAME" -C "$TMP"

# Install binary
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/zenqo" "$INSTALL_DIR/zenqo"
else
  echo "Installing to $INSTALL_DIR (requires sudo)..."
  sudo mv "$TMP/zenqo" "$INSTALL_DIR/zenqo"
fi
chmod +x "$INSTALL_DIR/zenqo"

echo ""
echo "zenqo $VERSION installed to $INSTALL_DIR/zenqo"
echo ""
echo "Get started:"
echo "  zenqo new my-app"
echo "  cd my-app && zenqo dev"
