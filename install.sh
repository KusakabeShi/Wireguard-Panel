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

# Function to detect OS
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        OS_FAMILY=$ID_LIKE
    elif [[ -f /etc/redhat-release ]]; then
        OS="rhel"
        OS_FAMILY="rhel fedora"
    elif [[ -f /etc/debian_version ]]; then
        OS="debian"
        OS_FAMILY="debian"
    else
        print_error "Cannot detect operating system"
        exit 1
    fi
}

# Function to install system requirements
install_requirements() {
    print_info "Installing required system tools..."
    
    detect_os
    
    if [[ "$OS_FAMILY" == *"debian"* ]] || [[ "$OS" == "ubuntu" ]] || [[ "$OS" == "debian" ]]; then
        print_info "Detected Debian/Ubuntu system"
        apt-get update
        if ! apt-get install -y iproute2 wireguard-tools iptables; then
            print_error "Failed to install required packages. Please install manually:"
            print_error "apt-get install iproute2 wireguard-tools iptables"
            exit 1
        fi
    elif [[ "$OS_FAMILY" == *"rhel"* ]] || [[ "$OS" == "rhel" ]] || [[ "$OS" == "centos" ]] || [[ "$OS" == "rocky" ]] || [[ "$OS" == "almalinux" ]] || [[ "$OS" == "fedora" ]]; then
        print_info "Detected RHEL/CentOS/Rocky/Fedora system"
        if command -v dnf &> /dev/null; then
            if ! dnf install -y iproute wireguard-tools iptables; then
                print_error "Failed to install required packages. Please install manually:"
                print_error "dnf install iproute wireguard-tools iptables"
                exit 1
            fi
        elif command -v yum &> /dev/null; then
            if ! yum install -y iproute wireguard-tools iptables; then
                print_error "Failed to install required packages. Please install manually:"
                print_error "yum install iproute wireguard-tools iptables"
                exit 1
            fi
        else
            print_error "Neither dnf nor yum found"
            exit 1
        fi
    else
        print_warn "Unknown operating system: $OS"
        print_warn "Please manually install required tools:"
        print_warn "  - iproute2 (ip command)"
        print_warn "  - wireguard-tools (wg, wg-quick)"
        print_warn "  - iptables (iptables, ip6tables, iptables-save, ip6tables-save)"
    fi
    
    print_info "System tools installation completed"
}

# Function to show IP forwarding warning
show_ip_forwarding_warning() {
    print_warn "IMPORTANT: IP forwarding must be enabled for WireGuard to function properly"
    print_warn "Please manually configure IP forwarding by running:"
    print_warn "  echo 'net.ipv4.ip_forward = 1' > /etc/sysctl.d/99-wireguard.conf"
    print_warn "  echo 'net.ipv6.conf.all.forwarding = 1' >> /etc/sysctl.d/99-wireguard.conf"
    print_warn "  sysctl -p /etc/sysctl.d/99-wireguard.conf"
}

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

# Install system requirements
install_requirements

# Create directory
print_info "Creating /etc/wireguard-panel directory..."
mkdir -p /etc/wireguard-panel

# Stop existing service if it's running to allow binary overwrite
if systemctl cat wireguard-panel.service &>/dev/null; then
    if systemctl is-active --quiet wireguard-panel; then
        print_info "WireGuard Panel service is running. Stopping it for the upgrade..."
        systemctl stop wireguard-panel
    fi
fi

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
            PASSWORD=$(grep -oP '(?<=Generated random password: )[^\s]+' /tmp/wg-panel-init.log 2>/dev/null || true)
            if [[ -n "$PASSWORD" ]]; then
                print_info "Generated admin password: $PASSWORD"
            else
                # Fallback: show the log content to help debug
                print_info "Configuration generated. Check the log for password:"
                cat /tmp/wg-panel-init.log
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

# Show IP forwarding warning
show_ip_forwarding_warning

print_info ""
print_info "You can:"
print_info "  - Check service status: systemctl status wireguard-panel"
print_info "  - View logs: journalctl -u wireguard-panel -f"
print_info "  - Stop service: systemctl stop wireguard-panel"
print_info "  - Restart service: systemctl restart wireguard-panel"

# Show final status
sleep 2
systemctl status wireguard-panel --no-pager --lines=10 || true
print_info "Access the panel via http://[your-server-ip]:5000"
print_info "Generated admin password: $PASSWORD"