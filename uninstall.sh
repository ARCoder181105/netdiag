#!/bin/bash
set -e

INSTALL_DIR=${INSTALL_DIR:-"/usr/local/bin"}
BINARY_NAME="netdiag"

echo "🗑️  Uninstalling netdiag..."

# Check if binary exists
if [ ! -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    echo "⚠️  netdiag not found in $INSTALL_DIR"
    # Try to find it using 'which' just in case
    FOUND_PATH=$(which $BINARY_NAME || true)
    if [ -n "$FOUND_PATH" ]; then
        echo "🔎 Found at: $FOUND_PATH"
        INSTALL_DIR=$(dirname "$FOUND_PATH")
    else
        echo "❌ Could not find netdiag installed on your system."
        exit 1
    fi
fi

echo "📍 Removing from $INSTALL_DIR/$BINARY_NAME..."

# Remove binary (use sudo if needed)
if [ -w "$INSTALL_DIR" ]; then
    rm "$INSTALL_DIR/$BINARY_NAME"
else
    sudo rm "$INSTALL_DIR/$BINARY_NAME"
fi

echo "✅ netdiag uninstalled successfully!"