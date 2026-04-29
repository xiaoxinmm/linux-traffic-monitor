#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="xiaoxinmm/linux-traffic-monitor"
INSTALL_DIR="/opt/linux-traffic-monitor"
BINARY_NAME="traffic-monitor"

# Print colored message
print_msg() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    print_error "Please run as root (use sudo)"
    exit 1
fi

print_msg "Starting Linux Traffic Monitor installation..."

# Detect system architecture
detect_arch() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="arm"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    print_msg "Detected architecture: $ARCH"
}

# Detect OS
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        VERSION=$VERSION_ID
    else
        print_error "Cannot detect OS. /etc/os-release not found."
        exit 1
    fi
    print_msg "Detected OS: $OS $VERSION"
}

detect_arch
detect_os

# Install dependencies based on OS
install_dependencies() {
    print_step "Installing dependencies..."
    case $OS in
        ubuntu|debian)
            print_msg "Installing dependencies for Ubuntu/Debian..."
            apt-get update -qq
            apt-get install -y libpcap0.8 curl wget tar
            ;;
        centos|rhel|fedora)
            print_msg "Installing dependencies for CentOS/RHEL/Fedora..."
            yum install -y libpcap curl wget tar
            ;;
        arch|manjaro)
            print_msg "Installing dependencies for Arch Linux..."
            pacman -Sy --noconfirm libpcap curl wget tar
            ;;
        *)
            print_error "Unsupported OS: $OS"
            print_msg "Please install libpcap manually"
            exit 1
            ;;
    esac
}

# Get latest release version from GitHub
get_latest_version() {
    print_step "Fetching latest release version..."
    LATEST_VERSION=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$LATEST_VERSION" ]; then
        print_warning "Could not fetch latest version from GitHub"
        return 1
    fi

    print_msg "Latest version: $LATEST_VERSION"
    return 0
}

# Download precompiled binary
download_binary() {
    print_step "Downloading precompiled binary..."

    # Only amd64 has precompiled binaries
    if [ "$ARCH" != "amd64" ]; then
        print_warning "Precompiled binaries are only available for amd64 architecture"
        print_msg "Your architecture ($ARCH) requires building from source"
        return 1
    fi

    if ! get_latest_version; then
        return 1
    fi

    # Construct download URL
    BINARY_FILE="${BINARY_NAME}-${LATEST_VERSION}-linux-${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_VERSION}/${BINARY_FILE}"

    print_msg "Download URL: $DOWNLOAD_URL"

    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    # Download binary
    print_msg "Downloading from GitHub Releases..."
    if ! wget -q --show-progress "$DOWNLOAD_URL" -O "$BINARY_FILE" 2>&1; then
        print_warning "Failed to download precompiled binary"
        cd - > /dev/null
        rm -rf "$TMP_DIR"
        return 1
    fi

    # Extract binary
    print_msg "Extracting binary..."
    if ! tar -xzf "$BINARY_FILE"; then
        print_error "Failed to extract binary"
        cd - > /dev/null
        rm -rf "$TMP_DIR"
        return 1
    fi

    # Create installation directory
    mkdir -p "$INSTALL_DIR"

    # Move binary to installation directory
    if [ -f "$BINARY_NAME" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
        print_msg "Binary installed to $INSTALL_DIR/$BINARY_NAME"
    else
        print_error "Binary not found in archive"
        cd - > /dev/null
        rm -rf "$TMP_DIR"
        return 1
    fi

    # Cleanup
    cd - > /dev/null
    rm -rf "$TMP_DIR"

    return 0
}

# Install build dependencies for compiling from source
install_build_dependencies() {
    print_step "Installing build dependencies..."
    case $OS in
        ubuntu|debian)
            print_msg "Installing build dependencies for Ubuntu/Debian..."
            apt-get install -y libpcap-dev golang-go git
            ;;
        centos|rhel|fedora)
            print_msg "Installing build dependencies for CentOS/RHEL/Fedora..."
            yum install -y libpcap-devel golang git
            ;;
        arch|manjaro)
            print_msg "Installing build dependencies for Arch Linux..."
            pacman -Sy --noconfirm libpcap go git
            ;;
        *)
            print_error "Unsupported OS: $OS"
            exit 1
            ;;
    esac
}

# Check Go version
check_go_version() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        print_msg "Go version: $GO_VERSION"
        return 0
    else
        print_error "Go is not installed properly"
        return 1
    fi
}

# Build from source
build_from_source() {
    print_step "Building from source..."

    install_build_dependencies

    if ! check_go_version; then
        print_error "Go installation failed"
        return 1
    fi

    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"

    # Check if already cloned
    if [ -d ".git" ]; then
        print_msg "Repository already exists, pulling latest changes..."
        git pull
    else
        print_msg "Cloning repository..."
        if ! git clone "https://github.com/${GITHUB_REPO}.git" .; then
            print_error "Failed to clone repository"
            return 1
        fi
    fi

    print_msg "Building traffic monitor..."
    if ! go build -o "$BINARY_NAME" main.go; then
        print_error "Build failed"
        return 1
    fi

    if [ ! -f "$BINARY_NAME" ]; then
        print_error "Build failed - binary not found"
        return 1
    fi

    chmod +x "$BINARY_NAME"
    print_msg "Build successful!"
    return 0
}

# Create systemd service
create_service() {
    print_step "Creating systemd service..."

    cat > /etc/systemd/system/traffic-monitor.service << 'SERVICEEOF'
[Unit]
Description=Linux Traffic Monitor
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/linux-traffic-monitor
ExecStart=/opt/linux-traffic-monitor/traffic-monitor
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
SERVICEEOF

    systemctl daemon-reload
    print_msg "Systemd service created"
}

# Main installation
main() {
    echo ""
    echo "=========================================="
    echo "  Linux Traffic Monitor Installation"
    echo "=========================================="
    echo ""

    # Install basic dependencies
    install_dependencies

    # Try to download precompiled binary first
    print_msg "Attempting to install precompiled binary..."
    if download_binary; then
        print_msg "Successfully installed precompiled binary!"
    else
        print_warning "Precompiled binary not available or download failed"
        print_msg "Falling back to building from source..."

        if ! build_from_source; then
            print_error "Installation failed"
            exit 1
        fi
    fi

    # Verify binary exists and is executable
    if [ ! -x "$INSTALL_DIR/$BINARY_NAME" ]; then
        print_error "Binary not found or not executable at $INSTALL_DIR/$BINARY_NAME"
        exit 1
    fi

    # Create systemd service
    create_service

    echo ""
    print_msg "============================================"
    print_msg "Installation completed successfully!"
    print_msg "============================================"
    echo ""
    print_msg "To start the monitor:"
    print_msg "  sudo systemctl start traffic-monitor"
    echo ""
    print_msg "To enable auto-start on boot:"
    print_msg "  sudo systemctl enable traffic-monitor"
    echo ""
    print_msg "To check status:"
    print_msg "  sudo systemctl status traffic-monitor"
    echo ""
    print_msg "To view logs:"
    print_msg "  sudo journalctl -u traffic-monitor -f"
    echo ""
    print_msg "Web interface will be available at:"
    print_msg "  http://$(hostname -I | awk '{print $1}'):8080"
    echo ""

    # Ask if user wants to start now
    read -p "Do you want to start the monitor now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        systemctl start traffic-monitor
        sleep 2
        systemctl status traffic-monitor --no-pager
        echo ""
        print_msg "Monitor started! Access the dashboard at http://$(hostname -I | awk '{print $1}'):8080"
    fi
}

main
