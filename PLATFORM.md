# Platform Compatibility

## Supported Platforms

**✅ Linux Only**: WG-Panel is designed exclusively for Linux systems.

### Supported Architectures
- **Linux AMD64** (x86_64)
- **Linux ARM64** (aarch64)

## Why Linux Only?

WG-Panel relies on several Linux-specific technologies that are not available on other platforms:

### 1. Netlink Interface (`github.com/vishvananda/netlink`)
- **Purpose**: Network interface management, IP address monitoring, route manipulation
- **Linux Specific**: Uses Linux kernel's netlink socket interface
- **Used in**: SNAT roaming service, interface IP detection, network monitoring

### 2. Raw Packet Capture (`github.com/google/gopacket/pcap`)
- **Purpose**: ARP/NDP packet interception for pseudo-bridge functionality  
- **Dependencies**: Requires libpcap, raw socket access
- **Used in**: Pseudo-bridge service for network bridging

### 3. Linux Syscalls and Constants
- **Purpose**: Network interface flags, address states
- **Examples**: `syscall.IFA_F_PERMANENT`, `syscall.IFA_F_TENTATIVE`
- **Used in**: Interface address prioritization and filtering

## Alternative Approaches for Cross-Platform

If cross-platform support is needed in the future, these components would need platform-specific implementations:

### Network Interface Management
- **Linux**: netlink
- **Windows**: WMI, netsh, or Win32 APIs
- **macOS**: BSD socket APIs, system commands

### Packet Capture
- **Linux**: libpcap
- **Windows**: WinPcap/Npcap (requires installation)
- **macOS**: libpcap (may require privileges)

### System Integration
- **Linux**: systemd, iptables/netfilter
- **Windows**: Windows Services, Windows Firewall APIs
- **macOS**: launchd, pfctl

## Runtime Requirements

### Linux System Dependencies
```bash
# Ubuntu/Debian
sudo apt-get install libpcap-dev

# RHEL/CentOS/Fedora  
sudo yum install libpcap-devel
# or
sudo dnf install libpcap-devel

# Arch Linux
sudo pacman -S libpcap
```

### Kernel Features Required
- **Netlink sockets**: Built into modern Linux kernels
- **Packet sockets**: For raw packet capture (CONFIG_PACKET)
- **Network namespaces**: For advanced networking features

### Privileges
- **Raw sockets**: May require root privileges or CAP_NET_RAW capability
- **Network configuration**: Requires root or appropriate capabilities
- **Interface management**: Requires root or CAP_NET_ADMIN capability

## Testing Platform Compatibility

To verify your Linux system is compatible:

```bash
# Check netlink support
./wg-panel -v

# Check if interfaces can be accessed  
ip link show

# Check packet capture capabilities (as root)
tcpdump -i any -c 1 arp 2>/dev/null && echo "Packet capture works"
```

## Docker Support

WG-Panel can run in Docker containers with appropriate privileges:

```dockerfile
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y libpcap0.8
COPY wg-panel /usr/local/bin/
# Requires --cap-add=NET_ADMIN and potentially --privileged
```