#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    print_error "Please run as root (use sudo)"
    exit 1
fi

print_msg "Starting Linux Traffic Monitor installation..."

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
    VERSION=$VERSION_ID
else
    print_error "Cannot detect OS. /etc/os-release not found."
    exit 1
fi

print_msg "Detected OS: $OS $VERSION"

# Install dependencies based on OS
install_dependencies() {
    case $OS in
        ubuntu|debian)
            print_msg "Installing dependencies for Ubuntu/Debian..."
            apt-get update
            apt-get install -y libpcap-dev golang-go git wget
            ;;
        centos|rhel|fedora)
            print_msg "Installing dependencies for CentOS/RHEL/Fedora..."
            yum install -y libpcap-devel golang git wget
            ;;
        arch|manjaro)
            print_msg "Installing dependencies for Arch Linux..."
            pacman -Sy --noconfirm libpcap go git wget
            ;;
        *)
            print_error "Unsupported OS: $OS"
            print_msg "Please install manually: libpcap-dev, golang, git"
            exit 1
            ;;
    esac
}

# Check Go version
check_go_version() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        print_msg "Go version: $GO_VERSION"
    else
        print_error "Go is not installed properly"
        exit 1
    fi
}

# Download and build
build_monitor() {
    INSTALL_DIR="/opt/linux-traffic-monitor"
    
    print_msg "Creating installation directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"
    
    # Check if already cloned
    if [ -d ".git" ]; then
        print_msg "Repository already exists, pulling latest changes..."
        git pull
    else
        print_msg "Cloning repository..."
        git clone https://github.com/xiaoxinmm/linux-traffic-monitor.git .
    fi
    
    print_msg "Building traffic monitor..."
    go build -o traffic-monitor main.go
    
    if [ ! -f "traffic-monitor" ]; then
        print_error "Build failed"
        exit 1
    fi
    
    print_msg "Build successful!"
}

# Create systemd service
create_service() {
    print_msg "Creating systemd service..."
    
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
    install_dependencies
    check_go_version
    build_monitor
    create_service
    
    print_msg ""
    print_msg "============================================"
    print_msg "Installation completed successfully!"
    print_msg "============================================"
    print_msg ""
    print_msg "To start the monitor:"
    print_msg "  sudo systemctl start traffic-monitor"
    print_msg ""
    print_msg "To enable auto-start on boot:"
    print_msg "  sudo systemctl enable traffic-monitor"
    print_msg ""
    print_msg "To check status:"
    print_msg "  sudo systemctl status traffic-monitor"
    print_msg ""
    print_msg "To view logs:"
    print_msg "  sudo journalctl -u traffic-monitor -f"
    print_msg ""
    print_msg "Web interface will be available at:"
    print_msg "  http://$(hostname -I | awk '{print $1}'):8080"
    print_msg ""
    
    # Ask if user wants to start now
    read -p "Do you want to start the monitor now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        systemctl start traffic-monitor
        sleep 2
        systemctl status traffic-monitor --no-pager
        print_msg ""
        print_msg "Monitor started! Access the dashboard at http://$(hostname -I | awk '{print $1}'):8080"
    fi
}

main
