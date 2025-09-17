#!/bin/bash

set -e

# Usage: ./install.sh [--install=y]
# --install=y : Automatically install dependencies without confirmation

# Parse command line arguments
AUTO_INSTALL=false
PASSWORD=""
for arg in "$@"; do
    case $arg in
        --install=y)
            AUTO_INSTALL=true
            shift
            ;;
        *)
            ;;
    esac
done

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

# Function to ask for confirmation
ask_confirmation() {
    local message="$1"
    if [[ "$AUTO_INSTALL" == true ]]; then
        print_info "Auto-install mode enabled, proceeding with: $message"
        return 0
    fi
    
    echo -e "${YELLOW}[CONFIRM]${NC} $message"
    read -p "Do you want to continue? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        return 0
    else
        return 1
    fi
}

# Function to check if all required tools exist
check_tools_exist() {
    local missing_tools=()
    
    # Check for ip command (from iproute2)
    if ! command -v ip &> /dev/null; then
        missing_tools+=("iproute2 (ip command)")
    fi
    
    # Check for wg and wg-quick (from wireguard-tools)
    if ! command -v wg &> /dev/null; then
        missing_tools+=("wireguard-tools (wg command)")
    fi
    if ! command -v wg-quick &> /dev/null; then
        missing_tools+=("wireguard-tools (wg-quick command)")
    fi
    
    # Check for iptables commands
    if ! command -v iptables &> /dev/null; then
        missing_tools+=("iptables (iptables command)")
    fi
    if ! command -v ip6tables &> /dev/null; then
        missing_tools+=("iptables (ip6tables command)")
    fi
    if ! command -v iptables-save &> /dev/null; then
        missing_tools+=("iptables (iptables-save command)")
    fi
    if ! command -v ip6tables-save &> /dev/null; then
        missing_tools+=("iptables (ip6tables-save command)")
    fi
    
    if [[ ${#missing_tools[@]} -eq 0 ]]; then
        print_info "All required tools are already installed"
        return 0
    else
        print_warn "Missing required tools:"
        for tool in "${missing_tools[@]}"; do
            print_warn "  - $tool"
        done
        return 1
    fi
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
    # First check if all tools already exist
    if check_tools_exist; then
        return 0
    fi
    
    detect_os
    
    if [[ "$OS_FAMILY" == *"debian"* ]] || [[ "$OS" == "ubuntu" ]] || [[ "$OS" == "debian" ]]; then
        print_info "Detected Debian/Ubuntu system"
        if ask_confirmation "Install required packages: iproute2 wireguard-tools iptables"; then
            print_info "Installing required system tools..."
            apt-get update
            if ! apt-get install -y iproute2 wireguard-tools iptables; then
                print_error "Failed to install required packages. Please install manually:"
                print_error "apt-get install iproute2 wireguard-tools iptables"
                exit 1
            fi
            print_info "System tools installation completed"
        else
            print_warn "Skipping dependency installation. Please ensure the following are installed:"
            print_warn "  - iproute2 (ip command)"
            print_warn "  - wireguard-tools (wg, wg-quick)"
            print_warn "  - iptables (iptables, ip6tables, iptables-save, ip6tables-save)"
        fi
    elif [[ "$OS_FAMILY" == *"rhel"* ]] || [[ "$OS" == "rhel" ]] || [[ "$OS" == "centos" ]] || [[ "$OS" == "rocky" ]] || [[ "$OS" == "almalinux" ]] || [[ "$OS" == "fedora" ]]; then
        print_info "Detected RHEL/CentOS/Rocky/Fedora system"
        if ask_confirmation "Install required packages: iproute wireguard-tools iptables"; then
            print_info "Installing required system tools..."
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
            print_info "System tools installation completed"
        else
            print_warn "Skipping dependency installation. Please ensure the following are installed:"
            print_warn "  - iproute2 (ip command)"
            print_warn "  - wireguard-tools (wg, wg-quick)"
            print_warn "  - iptables (iptables, ip6tables, iptables-save, ip6tables-save)"
        fi
    else
        print_warn "Unknown operating system: $OS"
        print_warn "Please manually install required tools:"
        print_warn "  - iproute2 (ip command)"
        print_warn "  - wireguard-tools (wg, wg-quick)"
        print_warn "  - iptables (iptables, ip6tables, iptables-save, ip6tables-save)"
    fi
}

# Function to show IP forwarding warning
show_ip_forwarding_warning() {
    local ipv4_status="unknown"
    local ipv6_status="unknown"
    local needs_warning=false

    if [[ -r /proc/sys/net/ipv4/ip_forward ]]; then
        if [[ "$(cat /proc/sys/net/ipv4/ip_forward)" == "1" ]]; then
            ipv4_status="enabled"
        else
            ipv4_status="disabled"
            needs_warning=true
        fi
    else
        needs_warning=true
    fi

    if [[ -r /proc/sys/net/ipv6/conf/all/forwarding ]]; then
        if [[ "$(cat /proc/sys/net/ipv6/conf/all/forwarding)" == "1" ]]; then
            ipv6_status="enabled"
        else
            ipv6_status="disabled"
            needs_warning=true
        fi
    else
        # Some systems may not have IPv6 enabled; warn only if IPv4 is also disabled.
        [[ "$ipv4_status" != "enabled" ]] && needs_warning=true
    fi

    if [[ "$needs_warning" == true ]]; then
        print_warn "IMPORTANT: IP forwarding must be enabled for WireGuard to function properly"
        print_warn "Detected state -> IPv4: $ipv4_status, IPv6: $ipv6_status"
        print_warn "Please configure IP forwarding by running:"
        print_warn "  echo 'net.ipv4.ip_forward = 1' > /etc/sysctl.d/99-wireguard.conf"
        print_warn "  echo 'net.ipv6.conf.all.forwarding = 1' >> /etc/sysctl.d/99-wireguard.conf"
        print_warn "  sysctl -p /etc/sysctl.d/99-wireguard.conf"
    fi
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
if [[ -n "$PASSWORD" ]]; then
    print_info "Generated admin password: $PASSWORD"
else
    print_info "Admin password was not changed or could not be detected automatically. Use existing credentials or inspect the service log for the current value."
fi
