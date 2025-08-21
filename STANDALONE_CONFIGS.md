# Standalone WireGuard Configurations

The WG-Panel now automatically generates **standalone WireGuard configurations** that include all firewall rules in PostUp/PreDown scripts. These configurations can be used independently with `wg-quick` without requiring the panel to be running.

## How It Works

### Automatic Generation
- When you create or modify interfaces/servers through the panel, it automatically generates standalone configurations
- All configurations are saved to the `wireguardConfigPath` (default: `/etc/wireguard/`)
- Each interface gets a file: `wg-{ifname}.conf`

### Firewall Rules Integration
The generated configurations include comprehensive firewall rules in PostUp/PreDown scripts:

#### PostUp Script Contains:
- **VRF Configuration**: Sets interface to correct VRF if specified
- **SNAT Rules**: MASQUERADE, SNAT, or IPv6 NETMAP rules for internet access
- **Routed Networks Firewall**: Allow/deny rules for client traffic routing
- **Server-specific Comment Strings**: All rules tagged with unique identifiers

#### PreDown Script Contains:
- **Firewall Cleanup**: Removes all rules using comment strings
- **VRF Removal**: Removes interface from VRF
- **Safe Cleanup**: Multiple cleanup methods to ensure all rules are removed

## Example Configuration

```ini
[Interface]
PrivateKey = <server-private-key>
ListenPort = 51820
Address = 10.0.1.1/24, fd00:10:1::1/64
PostUp = bash -c 'iptables -t nat -A POSTROUTING -s 10.0.1.0/24 ! -d 10.0.0.0/8 -j MASQUERADE -m comment --comment hvWVNb-ABC123; iptables -A FORWARD -s 10.0.1.0/24 -d 192.168.1.0/24 -j ACCEPT -m comment --comment hvWVNb-ABC123; iptables -A FORWARD -s 10.0.1.0/24 -j DROP -m comment --comment hvWVNb-ABC123'
PreDown = bash -c 'iptables-save | grep -v "comment hvWVNb-ABC123" | iptables-restore; while iptables -t nat -D POSTROUTING -m comment --comment hvWVNb-ABC123 2>/dev/null; do :; done; while iptables -D FORWARD -m comment --comment hvWVNb-ABC123 2>/dev/null; do :; done'

[Peer]
PublicKey = <client-public-key>
AllowedIPs = 10.0.1.10/32, fd00:10:1::a/128
```

## Independent Usage

### Using with wg-quick
```bash
# Start the VPN with all firewall rules
sudo wg-quick up /etc/wireguard/wg-vpn0.conf

# Stop the VPN and clean up all firewall rules
sudo wg-quick down /etc/wireguard/wg-vpn0.conf
```

### Using with systemd
```bash
# Enable auto-start on boot
sudo systemctl enable wg-quick@wg-vpn0

# Start manually
sudo systemctl start wg-quick@wg-vpn0

# Check status
sudo systemctl status wg-quick@wg-vpn0
```

## Benefits

### 🚀 **Independence**
- Configurations work without the panel running
- No dependency on the panel service for VPN operation
- Can be deployed to servers without the panel installed

### 🔒 **Security** 
- Firewall rules are properly applied and removed
- Uses unique comment strings to avoid conflicts with other applications
- Safe cleanup prevents orphaned rules

### 🛠️ **Operational**
- Standard WireGuard tools (`wg-quick`, `systemctl`)
- Easy backup and deployment
- Compatible with existing WireGuard workflows

### 📝 **Maintenance**
- All rules automatically tagged with server instance ID
- Clean separation from other firewall rules
- Easy troubleshooting with comment-based identification

## Configuration Features Supported

- ✅ **SNAT/MASQUERADE**: Internet access for clients
- ✅ **IPv6 NETMAP**: Advanced IPv6 NAT scenarios  
- ✅ **Routed Networks**: Traffic filtering and routing
- ✅ **VRF Support**: Virtual routing and forwarding
- ✅ **Dual Stack**: IPv4 and IPv6 simultaneously
- ✅ **Comment-based Cleanup**: Safe rule management

The configurations are production-ready and can be used in any environment that supports WireGuard and iptables/ip6tables.