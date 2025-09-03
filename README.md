# WG-Panel

WG-Panel is a comprehensive web-based management panel for WireGuard servers, designed with a hierarchical three-level structure: **Interface > Server > Client**.

## Overview

This project provides a complete solution for managing WireGuard VPN infrastructure through an intuitive web interface. The backend is written in Go and provides a RESTful API, while the frontend is built with React and Material UI.

## Architecture

### Three-Level Hierarchy

1. **Interfaces** (`wg-[ifname]`) - Top-level WireGuard network interfaces
2. **Servers** - Logical VPN networks within an interface with specific IP ranges
3. **Clients** - Individual peers/users belonging to a server

### Key Components

- **Backend**: Go service providing RESTful API for WireGuard management
- **Frontend**: React SPA with Material UI for intuitive management
- **Authentication**: Username/password with session cookies
- **Dynamic Configuration**: Live config updates using `wg syncconf`
- **Advanced Networking**: SNAT, NETMAP, pseudo-bridging, and roaming support

## Features

### Interface Management
- Unique interface names matching `^[A-Za-z0-9_-]{1,12}$`
- VRF (Virtual Routing and Forwarding) support
- Configurable FwMark, MTU, and listen ports
- Automatic key generation or manual key assignment

### Server Management
- Dual-stack IPv4/IPv6 support with independent configuration
- DNS settings per server
- Advanced SNAT options:
  - MASQUERADE mode
  - Static IP SNAT
  - IPv6 NETMAP for network-to-network translation
- **SNAT Roaming**: Dynamic firewall rule updates based on master interface IP changes
- **Pseudo-bridging**: ARP/ND response service for local network integration
- Configurable routed networks and firewall rules
- Network overlap detection and validation

### Client Management
- Automatic or manual IP assignment within server networks
- Individual DNS overrides
- WireGuard key management (automatic generation or manual)
- Preshared key support
- Persistent keepalive configuration
- Real-time connection statistics (handshake, transfer data)

### Advanced Services

#### Pseudo-Bridge Service
- Uses pcap to monitor ARP requests and Neighbor Solicitation packets
- Responds with appropriate replies to make VPN clients appear on local network
- Per-interface and per-network configuration
- Automatic interface monitoring and recovery

#### SNAT Roaming Service
- Monitors master interface IP changes using netlink
- Dynamically updates firewall rules for SNAT/NETMAP
- Supports both IPv4 SNAT and IPv6 NETMAP roaming
- Handles interface failures and recovery

## SNAT and Roaming Configuration

WG-Panel supports three SNAT modes with different roaming capabilities:

### SNAT Modes Overview

| Mode | IPv4/IPv6 | Roaming Support | Pseudo-Bridge Support | Description |
|------|-----------|----------------|----------------------|-------------|
| MASQUERADE | Both | ❌ No | ❌ No | Uses interface's primary IP |
| SNAT | Both | ✅ Yes | ❌ No | Static IP or roaming to interface IP |
| NETMAP | IPv6 only | ✅ Yes | ✅ Yes | Network-to-network translation |

### 1. MASQUERADE Mode

**GUI Configuration:**
- ✅ Enable SNAT
- 🔘 SNAT Mode: **MASQUERADE**
- 📝 SNAT IP/Network: *(leave empty)*
- 📝 SNAT Roaming Interface: *(leave empty)*

**Behavior:**
- Uses `iptables -j MASQUERADE` 
- Automatically uses the interface's primary IP
- No roaming support - always uses current interface IP
- Cannot be combined with roaming or pseudo-bridging

### 2. SNAT Mode

#### Static SNAT
**GUI Configuration:**
- ✅ Enable SNAT
- 🔘 SNAT Mode: **SNAT**
- 📝 SNAT IP/Network: `203.0.113.10/32`
- 📝 SNAT Roaming Interface: *(leave empty)*

**Behavior:**
- Creates rule: `iptables -t nat -j SNAT --to-source 203.0.113.10`
- Uses fixed IP address
- No roaming capability

#### SNAT with Roaming
**GUI Configuration:**
- ✅ Enable SNAT
- 🔘 SNAT Mode: **SNAT**
- 📝 SNAT IP/Network: `0.0.0.0/32` *(IPv4)* or `::/128` *(IPv6)*
- 📝 SNAT Roaming Interface: `eth0`

**Parsing Logic:**
- `SNATIPNet` must be `0.0.0.0/32` (IPv4) or `::/128` (IPv6)
- The `0.0.0.0/32` acts as a placeholder indicating "use roaming IP"
- Real IP is dynamically retrieved from the master interface (`eth0`)

**Runtime Behavior:**
1. Monitor `eth0` for IP changes
2. When `eth0` has IP `203.0.113.50`, create rule:
   ```bash
   iptables -t nat -j SNAT --to-source 203.0.113.50
   ```
3. When `eth0` IP changes to `203.0.113.60`, update rule:
   ```bash
   iptables -t nat -D ... # Remove old rule
   iptables -t nat -j SNAT --to-source 203.0.113.60 # Add new rule
   ```

### 3. NETMAP Mode (IPv6 Only)

#### Static NETMAP
**GUI Configuration:**
- 📝 Server IPv6 Network: `fd00:db8:1::/64`
- ✅ Enable SNAT
- 🔘 SNAT Mode: **NETMAP**
- 📝 SNAT IP/Network: `2001:db8:100::/64`
- 📝 SNAT Roaming Interface: *(leave empty)*

**Requirements:**
- Server network and SNAT network must have same prefix length
- Creates bidirectional NETMAP rules:
```bash
ip6tables -t nat -A POSTROUTING -s fd00:db8:1::/64 -j NETMAP --to 2001:db8:100::/64
ip6tables -t nat -A PREROUTING -d 2001:db8:100::/64 -j NETMAP --to fd00:db8:1::/64
```

#### NETMAP with Roaming
**GUI Configuration:**
- 📝 Server IPv6 Network: `fd00:db8:1::/64`
- ✅ Enable SNAT
- 🔘 SNAT Mode: **NETMAP**
- 📝 SNAT IP/Network: `::100:0/112` *(offset from master interface)*
- 📝 SNAT Roaming Interface: `eth0`

**Parsing Logic:**
- `SNATIPNet` is parsed as an **offset**, not an absolute network
- The `::100:0/112` represents an offset from the master interface's network
- Must match the server network's prefix length for the host portion

**Example Calculation:**
1. **Master interface** `eth0` has IPv6: `2001:db8:abcd::1/64`
2. **Master network**: `2001:db8:abcd::/64`
3. **Offset**: `::100:0/112`
4. **Target network**: `2001:db8:abcd::/64` + `::100:0/112` = `2001:db8:abcd:100::/112`

**Generated Rules:**
```bash
ip6tables -t nat -A POSTROUTING -s fd00:db8:1::/64 -j NETMAP --to 2001:db8:abcd:100::/112
ip6tables -t nat -A PREROUTING -d 2001:db8:abcd:100::/112 -j NETMAP --to fd00:db8:1::/64
```

**When Master Interface Changes:**
1. `eth0` changes to `2001:db8:beef::1/64`
2. New master network: `2001:db8:beef::/64`
3. New target network: `2001:db8:beef::/64` + `::100:0/112` = `2001:db8:beef:100::/112`
4. Rules are updated automatically

### NETMAP Pseudo-Bridge Integration

Only NETMAP mode with roaming supports pseudo-bridging on the target network:

**GUI Configuration:**
- 📝 Server IPv6 Network: `fd00:db8:1::/64`
- ✅ Enable SNAT
- 🔘 SNAT Mode: **NETMAP**
- 📝 SNAT IP/Network: `::100:0/112`
- 📝 SNAT Roaming Interface: `eth0`
- ✅ **Enable NETMAP Pseudo-Bridge** *(checkbox)*

**Behavior:**
- Pseudo-bridge service monitors the master interface (`eth0`)
- Responds to Neighbor Solicitation for IPs in the target network (`2001:db8:beef:100::/112`)
- Makes VPN clients appear as if they're directly on the master interface's network
- Target network changes dynamically with the master interface

### Roaming Limitations and Requirements

#### Prefix Length Matching

**❌ Invalid Configuration:**
- 📝 Server IPv6 Network: `fd00:db8:1::/64` *(prefix length: /64)*
- 📝 SNAT IP/Network: `::100:0/112` *(prefix length: /112)*
- ⚠️ **Error**: Prefix lengths don't match (64 ≠ 112)

**✅ Valid Configuration:**
- 📝 Server IPv6 Network: `fd00:db8:1::/64` *(prefix length: /64)*
- 📝 SNAT IP/Network: `::100:0/64` *(prefix length: /64)*
- ✅ **Success**: Prefix lengths match (64 = 64)

#### Master Interface Requirements
- Must exist and be up during operation
- If interface disappears, roaming service waits and retries every 5 seconds
- Deprecated addresses are ignored
- Dynamic addresses preferred over static ones
- If no suitable address found, firewall rules are temporarily removed

#### Error Conditions
1. **Invalid offset format**: Offset must be valid IPv6 with correct prefix length
2. **Master interface not found**: Service waits and retries
3. **No suitable addresses**: Rules temporarily disabled until address available
4. **Firewall rule conflicts**: Old rules removed before adding new ones

### Monitoring and Troubleshooting

The roaming service provides logging for:
- Interface status changes
- IP address changes
- Firewall rule updates
- Error conditions and recovery attempts

This allows administrators to monitor the dynamic behavior and troubleshoot issues with roaming configurations.

## Installation & Usage

### Backend Configuration

The backend service uses a `config.json` file that is automatically managed. Key settings include:

- **WireGuard Config Path**: `/etc/wireguard` (where WireGuard configs are stored)
- **Web Interface**: Default at `http://localhost:5000`
- **Authentication**: Username/password (set via command line)
- **Frontend Files**: Served from `./frontend/build`
- **API Endpoints**: Available at `/api/*`

### Command Line Options

```bash
# Run with default config
./wg-panel

# Specify custom config path
./wg-panel -c /path/to/config.json

# Set new password
./wg-panel -p newpassword
```

### Configuration Parameters

- **WireGuardConfigPath**: Directory for WireGuard configs (default: `/etc/wireguard`)
- **User**: Username for web interface authentication
- **Password**: bcrypt-hashed password
- **ListenIP**: Service bind address (default: `::`)
- **ListenPort**: HTTP server port (default: `5000`)
- **BasePath**: Web interface URL prefix (default: `/`)

### First Run

On first run, if no config file exists:
1. A new `config.json` will be created
2. A random password will be generated and printed to console
3. Use this password with the default username to access the web interface

## API Documentation

The complete API specification is available in `API_SPEC.yaml`. The API provides endpoints for:

- Interface management (CRUD operations, enable/disable)
- Server management (CRUD operations, enable/disable, move between interfaces)
- Client management (CRUD operations, enable/disable, config generation)
- Real-time statistics and connection status

## Network Requirements

- Root privileges for WireGuard interface management
- iptables/ip6tables for firewall rule management
- Network interface access for pseudo-bridging and SNAT roaming
- WireGuard tools (`wg`, `wg-quick`) installed on the system

## Security Features

- Session-based authentication with server-side storage
- Network overlap validation to prevent conflicts
- Automatic firewall rule management with unique identifiers
- Secure key generation using crypto/rand
- Input validation and sanitization

## Development Status

This project implements a complete WireGuard management solution with production-ready features. All core functionality described in the specification has been implemented, including advanced features like SNAT roaming and pseudo-bridging.

## Building

```bash
# Backend
go mod tidy
go build -o wg-panel

# Frontend (if developing)
cd frontend
npm install
npm run build
```