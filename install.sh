#!/bin/bash
# install.sh — Build and install apitest binary
# Usage:
#   ./install.sh          # install to ~/.local/bin (no sudo)
#   ./install.sh --global # install to /usr/local/bin (requires sudo)

set -e

BINARY_NAME="apitest"
SRC_DIR="src"
BUILD_DIR="bin"
VERSION="0.3.0"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

info() { echo -e "${GREEN}$1${NC}"; }
error() { echo -e "${RED}$1${NC}" >&2; exit 1; }

# Check Go is installed
if ! command -v go &> /dev/null; then
    error "Go is not installed. Install Go 1.21+ from https://go.dev/dl/"
fi

# Determine install directory
INSTALL_DIR="$HOME/.local/bin"
if [[ "$1" == "--global" || "$1" == "-g" ]]; then
    INSTALL_DIR="/usr/local/bin"
fi

# Create install dir if needed
mkdir -p "$INSTALL_DIR"

# Determine script location (handle running from project root or elsewhere)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Build
info "Building apitest v${VERSION}..."
cd "$SRC_DIR"
go build -ldflags "-s -w" -o "../${BUILD_DIR}/${BINARY_NAME}" ./cmd/main.go
cd ..
info "✓ Built ${BUILD_DIR}/${BINARY_NAME}"

# Install
if [[ "$INSTALL_DIR" == "/usr/local/bin" ]]; then
    sudo cp "${BUILD_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
else
    cp "${BUILD_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
fi

info "✓ Installed to ${INSTALL_DIR}/${BINARY_NAME}"

# Verify
if command -v "$BINARY_NAME" &> /dev/null; then
    echo ""
    $BINARY_NAME --version
    echo ""
    info "Ready! Run 'apitest help' to get started."
else
    echo ""
    echo "Binary installed to ${INSTALL_DIR}/${BINARY_NAME}"
    echo ""
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        echo "⚠  ${INSTALL_DIR} is not in your PATH."
        echo "   Add this to your ~/.bashrc or ~/.zshrc:"
        echo ""
        echo "   export PATH=\"${INSTALL_DIR}:\$PATH\""
        echo ""
        echo "   Then restart your terminal or run: source ~/.bashrc"
    fi
fi
