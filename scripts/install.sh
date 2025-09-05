#!/bin/bash
set -e

# ToneClone CLI Universal Install Script
# Usage: curl -sSL https://raw.githubusercontent.com/toneclone/cli/main/scripts/install.sh | bash

# Configuration
GITHUB_REPO="toneclone/cli"
BINARY_NAME="toneclone"
INSTALL_DIR="${INSTALL_DIR:-}"
VERSION="${VERSION:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Utility functions
log() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    exit 1
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Darwin*) os="Darwin" ;;
        Linux*)  os="Linux" ;;
        CYGWIN*|MINGW*|MSYS*) os="Windows" ;;
        *) error "Unsupported operating system: $(uname -s)" ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64) arch="x86_64" ;;
        arm64|aarch64) arch="arm64" ;;
        armv7l) arch="armv7" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac

    # Handle special cases
    if [[ "$os" == "Darwin" && "$arch" == "arm64" ]]; then
        arch="ARM64"
    elif [[ "$os" == "Darwin" && "$arch" == "x86_64" ]]; then
        arch="x86_64"
    elif [[ "$os" == "Linux" && "$arch" == "arm64" ]]; then
        arch="ARM64"
    fi

    # Set archive extension
    local ext="tar.gz"
    if [[ "$os" == "Windows" ]]; then
        ext="zip"
    fi

    echo "${os}_${arch}.${ext}"
}

# Get latest release version from GitHub API
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | \
            grep '"tag_name":' | \
            sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | \
            grep '"tag_name":' | \
            sed -E 's/.*"([^"]+)".*/\1/'
    else
        error "Neither curl nor wget is available. Please install one of them."
    fi
}

# Determine installation directory
determine_install_dir() {
    if [[ -n "$INSTALL_DIR" ]]; then
        echo "$INSTALL_DIR"
        return
    fi

    # Try standard locations in order of preference
    local dirs=(
        "/usr/local/bin"
        "$HOME/.local/bin"
        "$HOME/bin"
    )

    for dir in "${dirs[@]}"; do
        if [[ -w "$dir" ]] || [[ ! -e "$dir" && -w "$(dirname "$dir")" ]]; then
            echo "$dir"
            return
        fi
    done

    # If no writable directory found, default to ~/.local/bin
    echo "$HOME/.local/bin"
}

# Create directory if it doesn't exist
ensure_dir() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        log "Creating directory: $dir"
        mkdir -p "$dir" || error "Failed to create directory: $dir"
    fi
}

# Check if directory is in PATH
check_path() {
    local dir="$1"
    if [[ ":$PATH:" != *":$dir:"* ]]; then
        warn "$dir is not in your PATH"
        echo "Add it to your PATH by adding this line to your shell profile:"
        echo "  export PATH=\"$dir:\$PATH\""
        echo ""
    fi
}

# Download and extract binary
install_binary() {
    local platform="$1"
    local version="$2"
    local install_dir="$3"
    
    # Create temporary directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf '$tmp_dir'" EXIT

    local archive_name="${BINARY_NAME}_${version#v}_${platform}"
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${archive_name}"

    log "Downloading $BINARY_NAME $version for $platform..."
    log "URL: $download_url"

    # Download the archive
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$download_url" -o "$tmp_dir/$archive_name" || \
            error "Failed to download $archive_name"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$download_url" -O "$tmp_dir/$archive_name" || \
            error "Failed to download $archive_name"
    else
        error "Neither curl nor wget is available"
    fi

    log "Extracting archive..."
    
    # Extract based on file type
    if [[ "$archive_name" == *.tar.gz ]]; then
        tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir" || \
            error "Failed to extract tar.gz archive"
    elif [[ "$archive_name" == *.zip ]]; then
        if command -v unzip >/dev/null 2>&1; then
            unzip -q "$tmp_dir/$archive_name" -d "$tmp_dir" || \
                error "Failed to extract zip archive"
        else
            error "unzip command not found, required for Windows archives"
        fi
    else
        error "Unsupported archive format: $archive_name"
    fi

    # Find the binary (it might be in a subdirectory)
    local binary_path
    binary_path=$(find "$tmp_dir" -name "$BINARY_NAME" -type f | head -1)
    
    if [[ -z "$binary_path" ]]; then
        error "Binary '$BINARY_NAME' not found in downloaded archive"
    fi

    # Ensure install directory exists
    ensure_dir "$install_dir"

    # Install the binary
    log "Installing to $install_dir/$BINARY_NAME..."
    cp "$binary_path" "$install_dir/$BINARY_NAME" || \
        error "Failed to copy binary to $install_dir"
    
    # Make executable
    chmod +x "$install_dir/$BINARY_NAME" || \
        error "Failed to make binary executable"
}

# Verify installation
verify_installation() {
    local install_dir="$1"
    local binary_path="$install_dir/$BINARY_NAME"
    
    if [[ ! -f "$binary_path" ]]; then
        error "Installation verification failed: $binary_path not found"
    fi
    
    if [[ ! -x "$binary_path" ]]; then
        error "Installation verification failed: $binary_path is not executable"
    fi
    
    log "Verifying installation..."
    local version_output
    version_output=$("$binary_path" --version 2>/dev/null) || \
        error "Installation verification failed: could not run $BINARY_NAME --version"
    
    success "Installation verified: $version_output"
}

# Main installation function
main() {
    log "ToneClone CLI Universal Installer"
    log "================================="
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    log "Detected platform: $platform"
    
    # Get version to install
    local target_version="$VERSION"
    if [[ "$target_version" == "latest" ]]; then
        log "Fetching latest release version..."
        target_version=$(get_latest_version)
        if [[ -z "$target_version" ]]; then
            error "Failed to get latest version from GitHub API"
        fi
    fi
    log "Target version: $target_version"
    
    # Determine installation directory
    local install_dir
    install_dir=$(determine_install_dir)
    log "Install directory: $install_dir"
    
    # Check for existing installation
    if [[ -f "$install_dir/$BINARY_NAME" ]]; then
        local existing_version
        existing_version=$("$install_dir/$BINARY_NAME" --version 2>/dev/null | grep -o 'version [^ ]*' | cut -d' ' -f2) || existing_version="unknown"
        warn "Found existing installation: $existing_version"
        
        if [[ "$existing_version" == "${target_version#v}" ]]; then
            log "Same version already installed. Skipping installation."
            success "ToneClone CLI $existing_version is already installed at $install_dir/$BINARY_NAME"
            return 0
        fi
        
        log "Updating from $existing_version to ${target_version#v}..."
    fi
    
    # Install binary
    install_binary "$platform" "$target_version" "$install_dir"
    
    # Verify installation
    verify_installation "$install_dir"
    
    # Check PATH
    check_path "$install_dir"
    
    success "ToneClone CLI ${target_version#v} installed successfully!"
    log "Run '$BINARY_NAME --help' to get started."
}

# Check if script is being sourced or executed
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi