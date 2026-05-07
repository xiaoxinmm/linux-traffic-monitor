#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/opt/linux-traffic-monitor"
SERVICE_NAME="traffic-monitor"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

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

echo ""
echo "=========================================="
echo "  Linux Traffic Monitor Uninstallation"
echo "=========================================="
echo ""

print_warning "This will completely remove Linux Traffic Monitor from your system."
print_warning "The following will be removed:"
echo "  - Systemd service: $SERVICE_FILE"
echo "  - Installation directory: $INSTALL_DIR"
echo "  - All traffic data and databases"
echo ""

read -p "Are you sure you want to continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_msg "Uninstallation cancelled"
    exit 0
fi

# Stop the service if running
print_step "Stopping service..."
if systemctl is-active --quiet "$SERVICE_NAME"; then
    systemctl stop "$SERVICE_NAME"
    print_msg "Service stopped"
else
    print_msg "Service is not running"
fi

# Disable the service if enabled
print_step "Disabling service..."
if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
    systemctl disable "$SERVICE_NAME"
    print_msg "Service disabled"
else
    print_msg "Service is not enabled"
fi

# Remove systemd service file
print_step "Removing systemd service file..."
if [ -f "$SERVICE_FILE" ]; then
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
    print_msg "Service file removed"
else
    print_msg "Service file not found"
fi

# Remove installation directory
print_step "Removing installation directory..."
if [ -d "$INSTALL_DIR" ]; then
    # Show directory size before removal
    DIR_SIZE=$(du -sh "$INSTALL_DIR" 2>/dev/null | cut -f1)
    print_msg "Directory size: $DIR_SIZE"

    rm -rf "$INSTALL_DIR"
    print_msg "Installation directory removed: $INSTALL_DIR"
else
    print_msg "Installation directory not found"
fi

# Check for any remaining processes
print_step "Checking for remaining processes..."
if pgrep -x "traffic-monitor" > /dev/null; then
    print_warning "Found running traffic-monitor processes, killing them..."
    pkill -9 "traffic-monitor"
    print_msg "Processes killed"
else
    print_msg "No remaining processes found"
fi

echo ""
print_msg "============================================"
print_msg "Uninstallation completed successfully!"
print_msg "============================================"
echo ""
print_msg "Linux Traffic Monitor has been completely removed from your system."
echo ""

# Optional: Ask if user wants to remove dependencies
echo ""
read -p "Do you want to remove installed dependencies (libpcap, etc.)? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_step "Removing dependencies..."

    # Detect OS
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
    else
        print_warning "Cannot detect OS, skipping dependency removal"
        exit 0
    fi

    case $OS in
        ubuntu|debian)
            print_msg "Removing dependencies for Ubuntu/Debian..."
            apt-get remove -y libpcap0.8 2>/dev/null || true
            apt-get autoremove -y
            ;;
        centos|rhel|fedora)
            print_msg "Removing dependencies for CentOS/RHEL/Fedora..."
            yum remove -y libpcap 2>/dev/null || true
            ;;
        arch|manjaro)
            print_msg "Removing dependencies for Arch Linux..."
            pacman -Rs --noconfirm libpcap 2>/dev/null || true
            ;;
        *)
            print_warning "Unsupported OS: $OS"
            print_msg "Please remove libpcap manually if needed"
            ;;
    esac

    print_msg "Dependencies removed"
else
    print_msg "Dependencies kept (libpcap may be used by other applications)"
fi

echo ""
print_msg "Thank you for using Linux Traffic Monitor!"
echo ""
