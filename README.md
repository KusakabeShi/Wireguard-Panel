[繁體中文](README_zh.md)

# WG-Panel

WG-Panel is a user-friendly, web-based management panel for WireGuard, designed to simplify the setup and administration of your VPN server. It features a clear hierarchical structure (Interfaces > Servers > Clients) and powerful networking capabilities, including dynamic IP support and advanced NAT configurations.

## Installation

### Quick Install (Recommended)

Run the automated installation script as root:

```bash
curl -fsSL https://raw.githubusercontent.com/KusakabeShi/Wireguard-Panel/refs/heads/main/install.sh | bash
```

This script will:
- Download the latest WG-Panel binary for your architecture
- Install it to `/usr/local/sbin/wg-panel`
- Create systemd service configuration
- Generate initial configuration and show the admin password
- Start and enable the WG-Panel service

After installation, access the web panel at `http://your-server:5000`

### Manual Installation

1. Download the appropriate binary for your platform from the [releases page](https://github.com/KusakabeShi/Wireguard-Panel/releases)
2. Make it executable: `chmod +x wg-panel-linux-*`
3. Run it once to generate initial configuration: `./wg-panel-linux-*`

## Getting Started

Follow these steps to get your WG-Panel server up and running (manual installation).

### 1. Initial Setup

First, run the WG-Panel binary to generate an initial configuration file.

```bash
./wg-panel
```

This command will create a `config.json` file in the same directory with a randomly generated password, which will be printed to the console.

### 2. Configuration

Next, open `config.json` and edit the following fields to match your environment:

*   `ListenIP`: The IP address for the web panel to listen on. Default is `::` (all interfaces).
*   `ListenPort`: The port for the web panel. Default is `5000`.

You can also change the `User` and `Password` here. If you change the password, make sure to use a bcrypt hash. You can set a new plaintext password by running:

```bash
./wg-panel -p "your_new_password"
```

### 3. Start the Server

Once configured, start the server again:

```bash
./wg-panel -c ./config.json
```

You can now access the web panel by navigating to `http://<ListenIP>:<ListenPort>` in your browser.

## Usage

WG-Panel organizes your WireGuard setup into three levels: **Interfaces**, **Servers**, and **Clients**.

### 1. Create an Interface

An interface represents a physical WireGuard network device (e.g., `wg0`).

1.  Navigate to the "Interfaces" section and click "Create".
2.  **Name**: A short name for the interface (e.g., `wg-home-vpn`).
3.  **Endpoint**: The public domain or IP address of your server. This will be used as the endpoint in the generated client configuration files.
4.  **Private Key**: Leave this empty to automatically generate a secure private key, or provide your own.
5.  Save the interface. It will be created but disabled by default. Enable it from the main interface list.

### 2. Create a Server

A server defines a logical group of clients and their associated network configuration within an interface.

1.  Select your newly created interface and go to the "Servers" tab.
2.  Click "Create" and fill in the server details:
    *   **Server Name**: A descriptive name (e.g., `Personal-Devices`).
    *   **DNS**: DNS servers to be pushed to clients (e.g., `1.1.1.1`).
    *   **Enable IPv4/v6 Subnet**: Check the box for each IP family you want to support.
    *   **IP Network**: The internal network for this server in CIDR format (e.g., `10.0.0.1/24`). The server will take the specified IP, and clients will be assigned addresses from this subnet.
    *   **Routed Networks**: A list of networks (CIDR format) that clients are allowed to access through the VPN. This corresponds to the `AllowedIPs` setting in the client configuration.
    *   **Block Non-Routed Network Packets**: If enabled, this option generates firewall rules to ensure clients can *only* access the networks specified in **Routed Networks**. All other traffic from the client will be dropped.

### 3. Advanced Server Features

#### Pseudo-Bridge

This feature makes VPN clients appear as if they are on the same Layer 2 network as the server. It works by responding to ARP (IPv4) and Neighbor Discovery (IPv6) requests on a specified master interface, effectively bridging the VPN and local networks.

#### SNAT (Source Network Address Translation)

SNAT allows clients to access the internet using the server's public IP address. The behavior is determined by the **SNAT IP/Net** field:

*   **MASQUERADE Mode**: Leave **SNAT IP/Net** empty. The firewall will use the primary IP of the outgoing interface for all traffic. This is the simplest mode.
*   **SNAT Mode**: Enter a single IP address. The firewall will use this specific IP as the source for all outgoing client traffic.
*   **NETMAP Mode (IPv6 only)**: Enter an IPv6 network in CIDR format. This maps the internal VPN subnet to a public IPv6 subnet. The mask length of the SNAT network must match the mask length of the server's internal IPv6 network.

#### SNAT Roaming (Dynamic IP Support)

SNAT Roaming is a powerful feature for servers with dynamic public IPs. It automatically updates firewall rules when the IP address of a specified **SNAT Roaming Master Interface** changes. This feature is only compatible with **SNAT** and **NETMAP** modes.

*   **In SNAT Mode**:
    *   Set **SNAT IP/Net** to `0.0.0.0` (for IPv4) or `::` (for IPv6).
    *   The service will automatically detect the current IP of the master interface and use it for the SNAT rule.

*   **In NETMAP Mode (IPv6)**:
    *   The **SNAT IP/Net** field is treated as a *network offset*. The service combines this offset with the network address of the master interface to create a publicly routable subnet for your VPN clients.
    *   **Example**:
        *   Your server's master interface has a dynamic IPv6 address: `2a0d:3a87::123/64`. The network is `2a0d:3a87::/64`.
        *   Your internal WireGuard server network is `fd28:f50:55c2::/112`.
        *   You set **SNAT IP/Net** to `::980d:0/112`.
        *   The service combines the master network and the offset, mapping your internal network `fd28:f50:55c2::/112` to the public network `2a0d:3a87::980d:0/112`.
    *   This allows your VPN clients to have publicly routable IPv6 addresses even when your server's public IPv6 prefix changes.

#### SNAT NETMAP Pseudo-Bridge

This option extends the **Pseudo-Bridge** functionality to the public subnet created by **SNAT NETMAP**. It will respond to ARP/ND requests for the mapped public IPs, making clients appear as if they are directly on the public network.

### 4. Create a Client

Finally, create clients for your server.

1.  Select a server and go to the "Clients" tab.
2.  Click "Create" and configure the client:
    *   **IP/IPv6**: You can manually assign specific IPs or leave them as `auto` to have WG-Panel assign the next available address from the server's network.
    *   **Private Key**: Leave this empty to generate a new keypair automatically, or provide a client's existing private key.

### 5. View Client Details and Config

After a client is created, you can manage it from the client list.

1.  Click the expand/details button next to the client entry.
2.  A detailed view will appear, showing the complete **WireGuard configuration**.
3.  Click the **QR Code** button to display a code that can be scanned for easy import onto mobile devices.
4.  This view also shows live connection status for the client, including:
    *   Data Transferred (Upload/Download)
    *   Last Handshake Time
    *   Endpoint (the client's public IP address)
