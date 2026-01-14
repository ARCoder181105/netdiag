#!/bin/bash
set -e

# netdiag Installation Script
# Supports Linux and macOS (Intel & Apple Silicon)

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Emojis
CHECK="✓"
CROSS="✗"
ARROW="→"
ROCKET="🚀"

# Default values
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
NETDIAG_VERSION="${NETDIAG_VERSION:-latest}"
REPO="ARCoder181105/netdiag"
BINARY_NAME="netdiag"

# Print functions
print_success() {
    echo -e "${GREEN}${CHECK}${NC} $1"
}

print_error() {
    echo -e "${RED}${CROSS}${NC} $1"
}

print_info() {
    echo -e "${CYAN}${ARROW}${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}!${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$os" in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        msys*|mingw*|cygwin*)
            print_error "Windows detected. Please use install.ps1 instead."
            exit 1
            ;;
        *)
            print_error "Unsupported OS: $os"
            exit 1
            ;;
    esac

    case "$arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac

    print_info "Detected platform: $OS-$ARCH"
}

# Get latest version from GitHub
get_latest_version() {
    if [ "$NETDIAG_VERSION" = "latest" ]; then
        print_info "Fetching latest version..."
        NETDIAG_VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "")
        
        if [ -z "$NETDIAG_VERSION" ]; then
            print_warning "No releases found. Will build from source."
            return 1
        fi
        print_success "Latest version: $NETDIAG_VERSION"
    fi
    return 0
}

# Download pre-built binary
download_binary() {
    local download_url="https://github.com/$REPO/releases/download/$NETDIAG_VERSION/netdiag-$OS-$ARCH"
    if [ "$OS" = "darwin" ]; then
        download_url="${download_url}"
    fi

    print_info "Downloading netdiag $NETDIAG_VERSION for $OS-$ARCH..."
    
    local tmp_file="/tmp/netdiag-$OS-$ARCH"
    
    if curl -fsSL "$download_url" -o "$tmp_file"; then
        chmod +x "$tmp_file"
        BINARY_PATH="$tmp_file"
        return 0
    else
        print_warning "Failed to download binary"
        return 1
    fi
}

# Build from source
build_from_source() {
    print_info "Building from source..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.25 or higher from https://go.dev/dl/"
        exit 1
    fi

    local go_version=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Using Go version: $go_version"

    # Clone repository to temporary directory
    local tmp_dir=$(mktemp -d)
    print_info "Cloning repository to $tmp_dir..."
    
    if ! git clone "https://github.com/$REPO.git" "$tmp_dir" &> /dev/null; then
        print_error "Failed to clone repository"
        rm -rf "$tmp_dir"
        exit 1
    fi

    cd "$tmp_dir"
    
    # Build binary
    print_info "Building binary..."
    if go build -ldflags="-s -w" -o netdiag; then
        print_success "Build successful"
        BINARY_PATH="$tmp_dir/netdiag"
        chmod +x "$BINARY_PATH"
    else
        print_error "Build failed"
        rm -rf "$tmp_dir"
        exit 1
    fi
}

# Install binary
install_binary() {
    print_info "Installing to $INSTALL_DIR..."
    
    # Check if we need sudo
    if [ ! -w "$INSTALL_DIR" ]; then
        print_warning "Installing to $INSTALL_DIR requires sudo privileges"
        sudo mkdir -p "$INSTALL_DIR"
        sudo cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
        sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    else
        mkdir -p "$INSTALL_DIR"
        cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi

    print_success "Installed to $INSTALL_DIR/$BINARY_NAME"
}

# Set capabilities on Linux
set_capabilities() {
    if [ "$OS" = "linux" ]; then
        print_info "Setting ICMP capabilities (requires sudo)..."
        if command -v setcap &> /dev/null; then
            if sudo setcap cap_net_raw+ep "$INSTALL_DIR/$BINARY_NAME"; then
                print_success "ICMP capabilities granted"
            else
                print_warning "Failed to set capabilities. You may need to run 'sudo netdiag ping ...' for ICMP operations"
            fi
        else
            print_warning "setcap not found. You may need to run 'sudo netdiag ping ...' for ICMP operations"
        fi
    elif [ "$OS" = "darwin" ]; then
        print_warning "macOS requires sudo for ICMP operations. Use: sudo netdiag ping ..."
    fi
}

# Verify installation
verify_installation() {
    print_info "Verifying installation..."
    
    if command -v netdiag &> /dev/null; then
        local version=$(netdiag --version 2>&1 || echo "unknown")
        print_success "netdiag installed successfully!"
        echo ""
        echo -e "${CYAN}Version:${NC} $version"
        echo -e "${CYAN}Location:${NC} $(which netdiag)"
        echo ""
        echo -e "${GREEN}${ROCKET} Quick Start:${NC}"
        echo "  netdiag ping google.com"
        echo "  netdiag speedtest"
        echo "  netdiag scan localhost -p 1-1000"
        echo ""
        echo "Run 'netdiag --help' for more information"
    else
        print_error "Installation verification failed"
        print_info "Make sure $INSTALL_DIR is in your PATH"
        exit 1
    fi
}

# Main installation flow
main() {
    echo -e "${GREEN}${ROCKET} netdiag Installation Script${NC}"
    echo ""

    detect_platform
    
    # Try to download pre-built binary first
    if get_latest_version && download_binary; then
        print_success "Downloaded pre-built binary"
    else
        # Fall back to building from source
        build_from_source
    fi

    install_binary
    set_capabilities
    verify_installation
}

main
