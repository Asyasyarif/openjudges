#!/bin/bash
set -euo pipefail

# ============================================
# openjudges Installer
# Zero-config, multi-platform, auto-update ready
# ============================================

# Configuration
BINARY_NAME="openjudges"
REPO="Asyasyarif/openjudges"
INSTALL_MODE=${INSTALL_MODE:-"user"}  # Default: user mode
REQUESTED_VERSION=${VERSION:-""}
INCLUDE_PRERELEASE=false

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
ORANGE='\033[38;5;214m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

# Set install directory based on mode
if [ "$INSTALL_MODE" = "user" ]; then
    INSTALL_DIR="$HOME/.openjudges/bin"
else
    INSTALL_DIR="/usr/local/bin"
fi

# ============================================
# Argument Parsing
# ============================================

while [[ $# -gt 0 ]]; do
    case "$1" in
        --system)
            INSTALL_MODE="system"
            INSTALL_DIR="/usr/local/bin"
            shift
            ;;
        --version|-v)
            if [ -n "${2:-}" ]; then
                REQUESTED_VERSION="$2"
                shift 2
            else
                echo -e "${RED}Error:${NC} --version requires a version argument"
                exit 1
            fi
            ;;
        --prerelease|--pre)
            INCLUDE_PRERELEASE=true
            shift
            ;;
        --help|-h)
            cat <<EOF
${GREEN}openjudges Installer${NC}

${BLUE}Usage:${NC}
  install.sh [options]

${BLUE}Options:${NC}
  --system       Install to /usr/local/bin (requires sudo, system-wide)
  --version V    Install specific version (e.g., v1.2.0)
  --prerelease   Include pre-release versions when fetching latest
  --help, -h     Show this help message

${BLUE}Examples:${NC}
  curl -fsSL https://raw.githubusercontent.com/Asyasyarif/openjudges/main/scripts/install.sh | bash
  ./install.sh --version v1.2.0
  ./install.sh --prerelease
  sudo ./install.sh --system

${BLUE}Default:${NC}
  Installs to ~/.openjudges/bin (user-level, no sudo required)
  Automatically adds to PATH and configures shell

${BLUE}Supported Platforms:${NC}
  Linux (amd64, arm64, armv7, armv6)
  macOS (Intel, Apple Silicon)
  Windows (amd64, 386)
  FreeBSD (amd64, arm64)

${BLUE}More info:${NC}
  https://github.com/Asyasyarif/openjudges
EOF
            exit 0
            ;;
        *)
            echo -e "${ORANGE}Warning:${NC} Unknown option '$1'"
            shift
            ;;
    esac
done

# ============================================
# Cleanup Function
# ============================================

cleanup() {
    if [ -d "${TMP_DIR:-}" ]; then
        rm -rf "$TMP_DIR"
    fi
}
trap cleanup EXIT

# ============================================
# Shell Detection
# ============================================

detect_shell_and_config() {
    local shell_name=""
    local config_file=""

    if [ -n "${ZSH_VERSION:-}" ] || [[ "$SHELL" == *"zsh"* ]]; then
        shell_name="zsh"
        XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
        for file in "$HOME/.zshrc" "$HOME/.zshenv" "$XDG_CONFIG_HOME/zsh/.zshrc" "$XDG_CONFIG_HOME/zsh/.zshenv"; do
            if [ -f "$file" ]; then
                config_file="$file"
                break
            fi
        done
        # Create if doesn't exist
        [ -z "$config_file" ] && config_file="$HOME/.zshrc"

    elif [ -n "${BASH_VERSION:-}" ] || [[ "$SHELL" == *"bash"* ]]; then
        shell_name="bash"
        XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
        for file in "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile" "$XDG_CONFIG_HOME/bash/.bashrc"; do
            if [ -f "$file" ]; then
                config_file="$file"
                break
            fi
        done
        [ -z "$config_file" ] && config_file="$HOME/.bashrc"

    elif [[ "$SHELL" == *"fish"* ]]; then
        shell_name="fish"
        config_file="$HOME/.config/fish/config.fish"
        mkdir -p "$(dirname "$config_file")"
        [ ! -f "$config_file" ] && touch "$config_file"
    elif [[ "$SHELL" == *"ash"* ]] || [ "$(basename "$SHELL" 2>/dev/null)" = "ash" ]; then
        shell_name="ash"
        for file in "$HOME/.ashrc" "$HOME/.profile" "/etc/profile"; do
            if [ -f "$file" ]; then
                config_file="$file"
                break
            fi
        done
        [ -z "$config_file" ] && config_file="$HOME/.profile"
    else
        shell_name="sh"
        for file in "$HOME/.ashrc" "$HOME/.profile" "/etc/profile"; do
            if [ -f "$file" ]; then
                config_file="$file"
                break
            fi
        done
        [ -z "$config_file" ] && config_file="$HOME/.profile"
    fi

    echo "$shell_name|$config_file"
}

# ============================================
# PATH Configuration
# ============================================

add_to_path() {
    local config_file="$1"
    local install_dir="$2"
    local shell_name="$3"

    # Check if already in PATH
    if [[ ":$PATH:" == *":$install_dir:"* ]]; then
        return 1
    fi
    
    # Check if command already exists in config file
    local path_command=""
        if [ "$shell_name" = "fish" ]; then
            path_command="fish_add_path $install_dir"
        elif [ "$shell_name" = "ash" ] || [ "$shell_name" = "sh" ]; then
            path_command="export PATH=\"\$PATH:$install_dir\""
        else
            path_command="export PATH=\"\$PATH:$install_dir\""
        fi
    
    # Check if command already exists in file
    if [ -f "$config_file" ] && grep -Fxq "$path_command" "$config_file" 2>/dev/null; then
        return 1
    fi
    
    # Add to config file
    if [ -w "$config_file" ] || [ ! -f "$config_file" ]; then
        echo "" >> "$config_file"
        echo "# openjudges - Auto-added by installer on $(date)" >> "$config_file"
        echo "$path_command" >> "$config_file"
        return 0
    else
        return 1
    fi
}

# ============================================
# Version Checking
# ============================================

check_existing_version() {
	if command -v openjudges >/dev/null 2>&1; then
		# Try to get version from binary
		installed_version=$(openjudges --version 2>/dev/null || echo "unknown")

        if [ "$installed_version" = "$LATEST_RELEASE" ]; then
            echo ""
            echo -e "${GREEN}✓${NC} Latest version ${LATEST_RELEASE} already installed"
            echo ""
			echo -e "${BLUE}Available commands:${NC}"
			echo -e "  ${CYAN}openjudges update${NC}       # Check for and apply updates"
			echo -e "  ${CYAN}openjudges --version${NC}    # Show current version"
			echo -e "  ${CYAN}openjudges --help${NC}       # Show all commands"
            echo ""
            exit 0
        fi

        echo ""
        echo -e "${ORANGE}⚠${NC} Upgrading from ${installed_version} to ${LATEST_RELEASE}"
    fi
}

# ============================================
# Platform Detection
# ============================================

echo -e "${BLUE}==>${NC} Installing ${BINARY_NAME}..."

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
    darwin*)  OS='darwin' ;;
    linux*)   OS='linux' ;;
    freebsd*) OS='freebsd' ;;
    netbsd*)  OS='netbsd' ;;
    openbsd*) OS='openbsd' ;;
    msys*|cygwin*|mingw*) OS='windows' ;;
    *)        echo -e "${RED}Error:${NC} Unsupported OS: ${OS}"; exit 1 ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64) ARCH='amd64' ;;
    arm64|aarch64) ARCH='arm64' ;;
    armv7l) ARCH='arm'; ARM_VERSION=7 ;;
    armv6l) ARCH='arm'; ARM_VERSION=6 ;;
    i386|i686) ARCH='386' ;;
    *)      echo -e "${RED}Error:${NC} Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

# Handle Rosetta 2 (macOS Intel binaries on ARM)
if [ "$OS" = "darwin" ] && [ "$ARCH" = "amd64" ]; then
    rosetta_flag=$(sysctl -n sysctl.proc_translated 2>/dev/null || echo 0)
    if [ "$rosetta_flag" = "1" ]; then
        echo -e "${ORANGE}==>${NC} Detected Rosetta 2 translation, using arm64 binary"
        ARCH='arm64'
    fi
fi

# Handle musl libc (Alpine Linux)
IS_MUSL=false
if [ "$OS" = "linux" ]; then
    if [ -f /etc/alpine-release ]; then
        IS_MUSL=true
    fi
    if command -v ldd >/dev/null 2>&1; then
        if ldd --version 2>&1 | grep -qi musl; then
            IS_MUSL=true
        fi
    fi
    if [ "$IS_MUSL" = true ]; then
        echo -e "${BLUE}==>${NC} Detected musl libc (Alpine Linux)"
    fi
fi

# Construct target string
TARGET="${OS}-${ARCH}"
if [ -n "${ARM_VERSION:-}" ]; then
    TARGET="${TARGET}v${ARM_VERSION}"
fi

echo -e "${BLUE}==>${NC} Detected platform: ${TARGET}"

# ============================================
# Version Fetching
# ============================================

if [ -n "$REQUESTED_VERSION" ]; then
    LATEST_RELEASE="$REQUESTED_VERSION"
    echo -e "${BLUE}==>${NC} Installing specific version: ${LATEST_RELEASE}"

    # Verify release exists
    http_status=$(curl -sI -o /dev/null -w "%{http_code}" "https://github.com/${REPO}/releases/tag/${LATEST_RELEASE}" 2>/dev/null || echo "000")
    if [ "$http_status" = "404" ]; then
        echo -e "${RED}Error:${NC} Release ${LATEST_RELEASE} not found"
        echo -e "${BLUE}==>${NC} Available releases: https://github.com/${REPO}/releases"
        exit 1
    fi
else
    echo -e "${BLUE}==>${NC} Fetching latest version info..."
    if [ "$INCLUDE_PRERELEASE" = true ]; then
        # Get all releases including pre-releases, take the first one
        LATEST_RELEASE=$(curl -s https://api.github.com/repos/${REPO}/releases 2>/dev/null | grep '"tag_name":' | head -n1 | sed -E 's/.*"([^"]+)".*/\1/' || echo "")
        if [ -n "$LATEST_RELEASE" ]; then
            echo -e "${BLUE}==>${NC} Including pre-releases..."
        fi
    else
        # Get only stable releases (excludes pre-releases)
        LATEST_RELEASE=$(curl -s https://api.github.com/repos/${REPO}/releases/latest 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "")
    fi
fi

if [ -z "$LATEST_RELEASE" ]; then
    echo -e "${RED}Error:${NC} Could not find release for ${REPO}"
    exit 1
fi

echo -e "${BLUE}==>${NC} Target version: ${LATEST_RELEASE}"

# Check if already installed
check_existing_version

# ============================================
# Download & Install
# ============================================

# Determine archive extension based on OS
if [ "$OS" = "windows" ] || [ "$OS" = "darwin" ]; then
    ARCHIVE_EXT=".zip"
else
    ARCHIVE_EXT=".tar.gz"
fi

# Check required tools
if [ "$ARCHIVE_EXT" = ".tar.gz" ]; then
    if ! command -v tar >/dev/null 2>&1; then
        echo -e "${RED}Error:${NC} 'tar' is required but not installed."
        exit 1
    fi
else
    if ! command -v unzip >/dev/null 2>&1; then
        echo -e "${RED}Error:${NC} 'unzip' is required but not installed."
        exit 1
    fi
fi

# Construct download URL
ARCHIVE_FILENAME="${BINARY_NAME}_${TARGET}${ARCHIVE_EXT}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_RELEASE}/${ARCHIVE_FILENAME}"

# Create temp directory
TMP_DIR=$(mktemp -d || { echo -e "${RED}Error:${NC} Failed to create temp directory"; exit 1; })
ARCHIVE_PATH="${TMP_DIR}/${ARCHIVE_FILENAME}"
TMP_BIN="${TMP_DIR}/${BINARY_NAME}"
[ "$OS" = "windows" ] && TMP_BIN="${TMP_BIN}.exe"

# Download with progress
echo -e "${BLUE}==>${NC} Downloading ${ARCHIVE_FILENAME}..."

# Use curl progress bar if TTY, otherwise silent
if [ -t 2 ]; then
    # Show progress bar for TTY
    if ! curl -# -L "$DOWNLOAD_URL" -o "$ARCHIVE_PATH"; then
        echo -e "\n${RED}Error:${NC} Download failed"
        echo -e "${BLUE}==>${NC} URL: ${DOWNLOAD_URL}"
        exit 1
    fi
else
    # Silent download for non-TTY (pipes, scripts)
    if ! curl -s -L "$DOWNLOAD_URL" -o "$ARCHIVE_PATH"; then
        echo -e "${RED}Error:${NC} Download failed"
        echo -e "${BLUE}==>${NC} URL: ${DOWNLOAD_URL}"
        exit 1
    fi
fi

# Extract archive
echo -e "${BLUE}==>${NC} Extracting archive..."
if [ "$ARCHIVE_EXT" = ".tar.gz" ]; then
    tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"
else
    unzip -q "$ARCHIVE_PATH" -d "$TMP_DIR"
fi

# Verify binary exists
if [ ! -f "$TMP_BIN" ]; then
    echo -e "${RED}Error:${NC} Binary not found in archive"
    exit 1
fi

# Make executable
chmod +x "$TMP_BIN"

# Install
BINARY_INSTALL_NAME="$BINARY_NAME"
[ "$OS" = "windows" ] && BINARY_INSTALL_NAME="${BINARY_NAME}.exe"

if [ "$INSTALL_MODE" = "user" ]; then
    mkdir -p "$INSTALL_DIR"
    echo -e "${BLUE}==>${NC} Installing to ${INSTALL_DIR}..."
    if ! mv "$TMP_BIN" "${INSTALL_DIR}/${BINARY_INSTALL_NAME}"; then
        echo -e "${RED}Error:${NC} Failed to move binary to ${INSTALL_DIR}"
        exit 1
    fi

    # Auto PATH setup
    echo -e "${BLUE}==>${NC} Setting up PATH..."
    shell_info=$(detect_shell_and_config)
    shell_name=$(echo "$shell_info" | cut -d'|' -f1)
    config_file=$(echo "$shell_info" | cut -d'|' -f2)

    if add_to_path "$config_file" "$INSTALL_DIR" "$shell_name"; then
        echo -e "${GREEN}✓${NC} Added ${INSTALL_DIR} to PATH in ${config_file}"
        echo ""
        echo -e "${ORANGE}==>${NC} Please run one of the following to activate:"
        echo -e "   ${CYAN}source $config_file${NC}"
        echo -e "   ${CYAN}Restart your terminal${NC}"
    else
        echo -e "${GREEN}✓${NC} Already in PATH"
    fi
    
    # GitHub Actions support
    if [ -n "${GITHUB_ACTIONS:-}" ] && [ "${GITHUB_ACTIONS}" = "true" ]; then
        echo "$INSTALL_DIR" >> "$GITHUB_PATH"
        echo -e "${GREEN}✓${NC} Added ${INSTALL_DIR} to \$GITHUB_PATH"
    fi
else
    echo -e "${BLUE}==>${NC} Installing to ${INSTALL_DIR} (requires sudo)..."
    mkdir -p "$INSTALL_DIR"
    if sudo mv "$TMP_BIN" "${INSTALL_DIR}/${BINARY_INSTALL_NAME}"; then
        echo -e "${GREEN}✓${NC} Successfully installed ${BINARY_NAME}!"
    else
        echo -e "${RED}Error:${NC} Failed to move binary to ${INSTALL_DIR}"
        exit 1
    fi
    
    # GitHub Actions support
    if [ -n "${GITHUB_ACTIONS:-}" ] && [ "${GITHUB_ACTIONS}" = "true" ]; then
        echo "$INSTALL_DIR" >> "$GITHUB_PATH"
        echo -e "${GREEN}✓${NC} Added ${INSTALL_DIR} to \$GITHUB_PATH"
    fi
fi

# ============================================
# Success Message
# ============================================

echo ""
echo -e "${GREEN}✓${NC} Installation complete!"
echo ""
echo -e "${BLUE}Quick Start:${NC}"
echo -e "  ${CYAN}openjudges${NC}              # Start interactive menu"
echo -e "  ${CYAN}openjudges --help${NC}       # Show all commands"
echo -e "  ${CYAN}openjudges update${NC}       # Update to latest version"
echo ""
echo -e "${BLUE}Documentation:${NC} https://github.com/Asyasyarif/openjudges"
echo ""
