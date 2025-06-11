#!/bin/bash

# pic installation script
# This script installs pic on Unix-like systems (macOS and Linux)

set -e

# Configuration
REPO_OWNER="kidandcat"
REPO_NAME="pic"
BINARY_NAME="pic"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_error() {
    echo -e "${RED}Error: $1${NC}" >&2
}

print_success() {
    echo -e "${GREEN}$1${NC}"
}

print_info() {
    echo -e "${BLUE}$1${NC}"
}

print_warning() {
    echo -e "${YELLOW}$1${NC}"
}

# Detect OS and architecture
detect_platform() {
    local os
    local arch

    # Detect OS
    case "$(uname -s)" in
        Darwin*)
            os="darwin"
            ;;
        Linux*)
            os="linux"
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}-${arch}"
}

# Check if running with sudo
check_sudo() {
    if [ "$EUID" -ne 0 ] && [ ! -w "$INSTALL_DIR" ]; then
        print_error "This script needs sudo privileges to install to $INSTALL_DIR"
        print_info "Please run: curl -sSfL https://${REPO_OWNER}.github.io/${REPO_NAME}/install.sh | sudo bash"
        exit 1
    fi
}

# Download the binary
download_binary() {
    local platform=$1
    local temp_dir=$(mktemp -d)
    local binary_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest/download/${BINARY_NAME}-${platform}"
    
    print_info "Downloading ${BINARY_NAME} for ${platform}..."
    
    if command -v curl >/dev/null 2>&1; then
        curl -sSfL "$binary_url" -o "${temp_dir}/${BINARY_NAME}" || {
            print_error "Failed to download ${BINARY_NAME}"
            rm -rf "$temp_dir"
            exit 1
        }
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$binary_url" -O "${temp_dir}/${BINARY_NAME}" || {
            print_error "Failed to download ${BINARY_NAME}"
            rm -rf "$temp_dir"
            exit 1
        }
    else
        print_error "Neither curl nor wget found. Please install one of them."
        rm -rf "$temp_dir"
        exit 1
    fi
    
    echo "$temp_dir"
}

# Install the binary
install_binary() {
    local temp_dir=$1
    
    print_info "Installing ${BINARY_NAME} to ${INSTALL_DIR}..."
    
    # Make binary executable
    chmod +x "${temp_dir}/${BINARY_NAME}"
    
    # Move to install directory
    if [ -w "$INSTALL_DIR" ]; then
        mv "${temp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo mv "${temp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi
    
    # Clean up
    rm -rf "$temp_dir"
}

# Check Chrome/Chromium installation
check_chrome() {
    print_info "Checking for Chrome/Chromium installation..."
    
    if command -v google-chrome >/dev/null 2>&1 || \
       command -v google-chrome-stable >/dev/null 2>&1 || \
       command -v chromium >/dev/null 2>&1 || \
       command -v chromium-browser >/dev/null 2>&1 || \
       [ -d "/Applications/Google Chrome.app" ] || \
       [ -d "/Applications/Chromium.app" ]; then
        print_success "Chrome/Chromium found!"
    else
        print_warning "Chrome or Chromium not found. Please install Chrome or Chromium to use ${BINARY_NAME}."
        print_info "Visit https://www.google.com/chrome/ to download Chrome."
    fi
}

# Verify installation
verify_installation() {
    if command -v ${BINARY_NAME} >/dev/null 2>&1; then
        print_success "${BINARY_NAME} installed successfully!"
        print_info "Version: $(${BINARY_NAME} --version 2>/dev/null || echo 'version command not available')"
        print_info "Run '${BINARY_NAME} --help' to get started"
    else
        print_error "Installation failed. ${BINARY_NAME} not found in PATH."
        print_info "You may need to add ${INSTALL_DIR} to your PATH."
        exit 1
    fi
}

# Main installation process
main() {
    print_info "Installing ${BINARY_NAME}..."
    
    # Check if already installed
    if command -v ${BINARY_NAME} >/dev/null 2>&1; then
        print_warning "${BINARY_NAME} is already installed at $(which ${BINARY_NAME})"
        read -p "Do you want to reinstall? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Installation cancelled."
            exit 0
        fi
    fi
    
    # Check sudo if needed
    check_sudo
    
    # Detect platform
    platform=$(detect_platform)
    print_info "Detected platform: ${platform}"
    
    # Download binary
    temp_dir=$(download_binary "$platform")
    
    # Install binary
    install_binary "$temp_dir"
    
    # Check Chrome
    check_chrome
    
    # Verify installation
    verify_installation
    
    print_success "Installation complete!"
    echo
    print_info "Quick start:"
    echo "  ${BINARY_NAME} example.com                    # Take a screenshot"
    echo "  ${BINARY_NAME} --pdf example.com              # Save as PDF"
    echo "  ${BINARY_NAME} --help                         # Show all options"
}

# Run main function
main "$@"