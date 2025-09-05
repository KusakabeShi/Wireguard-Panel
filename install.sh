#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   print_error "This script must be run as root"
   exit 1
fi

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        print_error "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

print_info "Detected architecture: $ARCH"

# Create directory
print_info "Creating /etc/wireguard-panel directory..."
mkdir -p /etc/wireguard-panel

# Download the binary
BINARY_URL="https://github.com/KusakabeShi/Wireguard-Panel/releases/latest/download/wg-panel-linux-$ARCH"
print_info "Downloading WireGuard Panel binary from $BINARY_URL..."

if ! curl -L -o /usr/local/sbin/wg-panel "$BINARY_URL"; then
    print_error "Failed to download WireGuard Panel binary"
    exit 1
fi

# Make binary executable
chmod +x /usr/local/sbin/wg-panel

# Download systemd service file
SERVICE_URL="https://raw.githubusercontent.com/KusakabeShi/Wireguard-Panel/refs/heads/main/wireguard-panel.service"
print_info "Downloading systemd service file from $SERVICE_URL..."

if ! curl -L -o /usr/lib/systemd/system/wireguard-panel.service "$SERVICE_URL"; then
    print_error "Failed to download systemd service file"
    exit 1
fi

# Reload systemd daemon
print_info "Reloading systemd daemon..."
systemctl daemon-reload

# Generate initial config
print_info "Generating initial configuration..."
cd /etc/wireguard-panel

# Run the binary once to generate config
if ! /usr/local/sbin/wg-panel --help >/dev/null 2>&1; then
    print_warn "Binary test failed, but continuing with installation..."
fi

# Check if config.json exists, if not, create a basic one
if [[ ! -f "/etc/wireguard-panel/config.json" ]]; then
    print_info "Creating initial configuration file..."
    /usr/local/sbin/wg-panel > /tmp/wg-panel-init.log 2>&1 &
    WG_PANEL_PID=$!
    
    # Wait a few seconds for config generation
    sleep 3
    
    # Kill the process
    kill $WG_PANEL_PID 2>/dev/null || true
    wait $WG_PANEL_PID 2>/dev/null || true
    
    # Check if config was created
    if [[ -f "/etc/wireguard-panel/config.json" ]]; then
        print_info "Configuration file created successfully"
        
        # Try to extract password from logs
        if [[ -f "/tmp/wg-panel-init.log" ]]; then
            PASSWORD=$(grep -oP '(?<=password: )[^\s]+' /tmp/wg-panel-init.log 2>/dev/null || true)
            if [[ -n "$PASSWORD" ]]; then
                print_info "Generated admin password: $PASSWORD"
            fi
            rm -f /tmp/wg-panel-init.log
        fi
    else
        print_warn "Configuration file was not created automatically"
        print_info "You may need to run the service manually first: systemctl start wireguard-panel"
    fi
fi

# Enable and start service
print_info "Enabling wireguard-panel service..."
systemctl enable wireguard-panel

print_info "Starting wireguard-panel service..."
if systemctl start wireguard-panel; then
    print_info "WireGuard Panel service started successfully"
else
    print_warn "Service start may have failed. Check with: systemctl status wireguard-panel"
fi

print_info "Installation completed!"
print_info "You can:"
print_info "  - Check service status: systemctl status wireguard-panel"
print_info "  - View logs: journalctl -u wireguard-panel -f"
print_info "  - Stop service: systemctl stop wireguard-panel"
print_info "  - Restart service: systemctl restart wireguard-panel"

# Show final status
sleep 2
systemctl status wireguard-panel --no-pager --lines=10 || true