#!/bin/bash
set -e

REPO="ARCoder181105/netdiag"
INSTALL_DIR=${INSTALL_DIR:-"/usr/local/bin"}
BINARY_NAME="netdiag"

# Detect OS and Arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

if [ "$ARCH" == "x86_64" ]; then ARCH="amd64"; fi
if [ "$ARCH" == "aarch64" ]; then ARCH="arm64"; fi

echo "🔮 Detecting system: $OS/$ARCH"

# Determine Version
if [ -z "$NETDIAG_VERSION" ]; then
    VERSION=$(curl -fsSL https://api.github.com/repos/$REPO/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
else
    VERSION=$NETDIAG_VERSION
fi

if [ -z "$VERSION" ]; then
    echo "❌ Could not detect latest version. Please check the repository."
    exit 1
fi

echo "⬇️  Downloading netdiag $VERSION..."
URL="https://github.com/$REPO/releases/download/$VERSION/netdiag-$OS-$ARCH"

if ! curl -fsSL -o "$BINARY_NAME" "$URL"; then
    echo "❌ Download failed. Binary for $OS/$ARCH might not exist."
    exit 1
fi

chmod +x "$BINARY_NAME"

echo "📦 Installing to $INSTALL_DIR..."
sudo mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"

if [ "$OS" == "linux" ]; then
    echo "🔧 Setting capabilities (ping/trace without sudo)..."
    sudo setcap cap_net_raw+ep "$INSTALL_DIR/$BINARY_NAME" || echo "⚠️  Warning: setcap failed. You may need sudo for ping."
fi

echo "✅ netdiag installed successfully!"