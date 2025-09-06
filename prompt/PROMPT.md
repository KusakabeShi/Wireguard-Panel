I'm going to implement a WireGuard server panel:

### Backend:
*   **Programming language:** Golang
*   **Description:** This is a WireGuard server management service, which has a three-level hierarchy: Interface, Server, and Client.
    *   Clients belong to a server, and servers belong to an interface.
*   **API:** Provide a RESTful API, with separate modules in different files.
*   **Frontend:** Serve a static sites for frontend files.
*   **Authentication:** Username/password with session cookies stored in server memory. When the server closes, sessions are persisted to a `session.json` file.
*   **Parameters:**
    *   `-c [configpath]`: Optional. If not provided, defaults to `./config.json`. If the file does not exist, it will be created with a random password, which is then printed to the console.
    *   `-p [new_password]`: Sets a new password in the configuration file.

### And we have the following attributes:

### Service:
*   **WireGuardConfigPath:** The path to store the configuration of WireGuard interfaces. Default value: `/etc/wireguard`.
*   **User:** The username for the service.
*   **Password:** The bcrypt-hashed password.
*   **ListenIP:** The listen IP for the service, default "::"
*   **ListenPort:** The listen port for the service. default 5000
*   **BasePath:** The prefix to serve frontend static files. Default: `/`.

### Interface:
*   **ifname:** The interface's short name, matching `^[A-Za-z0-9_-]{1,15}$`. Must be unique and starts with WgIfPrefix.
*   **ID:** The unique ID of the interface, allocated when added. It is not changeable.
*   **Enabled:** A boolean indicating if this interface is enabled. This value cannot be edited via the standard 'edit' API; it can only be modified using the dedicated 'SetEnable' API. When created, it will always be disabled.
*   **VRF name:** The VRF (Virtual Routing and Forwarding) of the device. If null, no VRF is applied. Otherwise, `ip link set dev $IFNAME master $VRF_NAME` is added to the WireGuard configuration. Default value: null.
*   **FwMark:** The `FwMark` attribute for the wg interface. Default value: null. Can be decimal or hexadecimal. If not null, `FwMark` is added to the server configuration.
*   **Endpoint:** The public endpoint (domain or IP) of the WireGuard interface.
*   **Port:** The listen port of the WireGuard interface.
*   **MTU:** The MTU of the WireGuard interface.
*   **PrivateKey:** The private key of the interface. If null, it is randomly generated.
*   **Servers:** A list of servers belonging to this interface.

#### Actions:
*   **New:** Create a new WireGuard interface.
*   **Get:** Get the properties of this interface. Adds a `PublicKey` column, which is calculated from the private key.
*   **Edit:** Edit WireGuard interface properties.
    *   When **VRF** is edited, check for network overlaps for all child servers before applying the change. If a collision is detected (a child server's network overlaps with another server's network in the target VRF), the operation fails.
    *   When **VRF, FwMark, Port, or PrivateKey** are changed, regenerate the WireGuard config and sync it to the `wg` interface (`wg syncconf [ifname] <(wg-quick strip [confpath])`).
    *   When **MTU** is changed, use `ip link set dev $IFNAME mtu $NEW_MTU` to apply the change to the interface.
*   **SetEnable:**
    *   **true:** Set `Enabled` to `true`, enable the WireGuard interface.
    *   **false:** Set `Enabled` to `false`, disable the WireGuard interface.
*   **Delete:** Delete the WireGuard interface and all of its associated servers.

### IPNetWrapper
It's a net.IPNet Wrapper, same functionality as net.IPNet.
Because net.IPNet doesn't support marshal/unmarshal to humen readable format, so we use a IPNetWrapper to do it.
While parsing, use ParseCIDRv4 for IPv4 section and ParseCIDRv6 for IPv6 section to avoid incorrect Address Family type.
```
# This is a pert of prompt, for reference only, don't move this file. If nessesary, use copy instead of mv
@prompt/IPNetWarpper.go
```

### Server:
*   **Name:** The server's name.
*   **DNS:** DNS settings to be displayed in the client config. If null, the `DNS` row is not added to the client's configuration.
*   **ID:** The unique ID of the server, allocated when added. It is not changeable.
*   **Enabled:** A boolean indicating if this server is enabled. This value cannot be edited via the standard 'edit' API; it can only be modified using the dedicated 'SetEnable' API. When created, it will always be disabled.
*   **IPv4:**
    1.  **Enabled:** Whether IPv4/IPv6 is enabled.
    2.  **Network:** IP and mask length, like `192.168.1.1/24`. Use `IPNetWrapper` to store this to prevent potential exploits.
        *   The IP/prefix_len is a CIDR. For example, `IPNetWrapper("192.168.1.254/24")` results in `IP=192.168.1.254` and `Network=192.168.1.0/24`.
        *   When a new client is added, search for the first available IP within the network. The network and broadcast addresses (e.g., .0 and .255) are reserved, and the server's own IP is occupied. The operation fails if no IP is available.
        *   **Check:** Cannot overlap with other servers in the same VRF.
    3.  **Pseudo-bridge:** `null` to disable. A string (max length 15) to enable, and the content of the string is the master interface for the Pseudo-bridge service. This section is under IPvX, IPv4 and IPv6 has their own settings.
    4.  **SNAT:**
        1.  **Enabled:** Enable SNAT for IPv4/IPv6 or not.
        2.  **SNAT IPNet:** The IP to use for SNAT. Datatype: IPNetWrapper or null
            In SNAT section, if null, use ifconfig MASQUERADE.
            In SNAT section, If an single IP (/32 for ipv4 or /128 for ipv6) is provided, use ifconfig SNAT mode.
            In IPv4 SNAT section, If an IPv4Net provided and len != 32, raise error for not supported.
            In IPv6 SNAT section, If an IPv6Net provided and len != 128, use ifconfig NETMAP mode.
                In NETMAP mode, the mask length must be equal to the server's IPv6 Network mask length. otherwise raise error.
                And it will Generate two NETMAP rules to map <server_network>(Server IPv6 Network) and <target_network>(Server IPv6 SNAT IPNet):
                1. ip6tables -t nat -A PREROUTING -s <server_network> -j NETMAP --to <target_network>
                2. ip6tables -t nat -A PREROUTING -s <target_network> -j NETMAP --to <server_network>
            In conclusion, SNAT option exists on both IPv4 and IPv6 mode, but in IPv6, it supports additional NATMAP mode.
            The `SNAT IPNet` can be static or dynamic. If dynamic(SNAT Roaming enabled), don't add the firewall rules by the main thread. Let the SNAT Roaming Service to hendle the related firewall rule.
        3.  **SNAT Roaming:** `null` to disable. A string (max length 15) to enable, and the content of the string is the master interface for the SNAT Roaming.
            This feature is a part of SNAT. The difference betweeh SNAT mode and SNAT Roaming Mode is SNAT mode spacify SNAT IP(Net) Manually, but SNAT Roaming Mode choose IP(Net) from master interface by SNAT Roaming Service.
            So that it requires SNAT works on ifconfig SNAT mode or NATMAP mode. 
            When this option enabled, the `SNAT IPNet` will no longer be parsed to a IP address.
            In SNAT Mode, The `SNAT IPNet` must be `0.0.0.0/32` or `::/128`. Otherwise raise an error.
                In this case, real IPv4/IPv6 for SNAT is retrived from the master interface, and use it as `SNAT IPNet` in firewall rule generation.
            In NETMAP Mode, It will be parsed to a IPv6Net offset instead.
                In this case, The real IPv6Net is calculated from the master interface. It retrive the ipv6 network from the master interface, add to the offset and use it as real IPv6Net ( a.k.a target_network ) fot NETMAP firewall rule generation.
                For example, the IPv6 of the interface is 2a0d:3a87::123/64, then the netowrk is 2a0d:3a87::/64. So that if the IPv6Net is ::980d:0/112, the real `SNAT IPNet` ( a.k.a target_network ) for NATMAP is 2a0d:3a87::980d:0/112
            When we retrive IPv4/IPv6/IPv6Net from master interface, the deprecated address will be ignored. When multiple address scaned, use the dynamic address first. If we can't retrive any address, ignore this feature (don't generate firewall rule) and wait next scan.
            If the network on the master interface changes, it will update the firewall rules accordingly.
        4.  **SNAT NETMAP pseudo-bridge:** boolean. This mode works on IPv6 SNAT NETMAP mode only. it exist on ipv4 but not valid for configuration reuse.
            If false, do nothing. If true, Perform pseudo-bridge on SNAT Roaming master interface, but use target_network as the network, which will be hendled by pseudo-bridge module.
        5.  **SNAT Excluded Network:** A network range to exclude from SNAT. If null, defaults to the server's own network range.
        *   If enabled, add SNAT firewall rules for this server, allowing clients to use the server's IP to access the external internet.
        *   Generate a rule like `-s SELF_NET/prefix ! -d "SNAT Excluded NAT"`.
    5.  **Routed Networks:** A list of IPv4/IPv6 networks. If null, defaults to the server's own network range. Must contain the server's own network range and cannot overlap with each other. This is equivalent to `AllowedIPs` for the client.
    6.  **Routed Networks Firewall:** A boolean. If true, add firewall rules to allow `-s [Server Network] -d [Routed Network]` and block other destination IPs (unless `Routed Networks` is `0.0.0.0/0` for IPv4 or `::/0` for IPv6).
    7.  **CommentString:** A randomly generated string. When the server is started, use `iptables`/`ip6tables` to add firewall rules with this special comment. When stopped, remove firewall rules based on the comment. The comment should be static (saved in the config file) to ensure it can be removed if the server stops unexpectedly. This is for internal use only and transparent to the API/frontend.
*   **IPv6:**
    *   Same as IPv4, but the IPv6 version. All IP/Net related setting are IPv6
*   **Note:** At least one of IPv4 or IPv6 must be enabled.
*   **Clients:** A list of clients belonging to this server.

#### Actions:
*   **New:** Create a new server.
*   **Get:** Get the properties of this server.
*   **GetClients:** Get all client properties, including last handshake time and transfer/received data.
*   **Edit:** Edit server properties.
    *   When **Network** is edited, check: 1) The offset of existing client IPs is still valid. 2) `max(client_ip_offset) < 2^(32 - new_mask_len)` (for IPv4) or `2^(128 - new_mask_len)` (for IPv6).
    *   When **Network, Routed Networks, or SNAT** are edited successfully, remove old IP addresses and firewall rules from the `wg` interface, then apply the new IP and firewall rules.
*   **SetEnable:**
    *   **true:** Set `Enabled` to `true`, add the IP address to the `wg` interface, add related firewall rules, regenerate the WireGuard config (including server clients as peers), and sync to the `wg` interface.
    *   **false:** Set `Enabled` to `false`, remove the IP address from the `wg` interface, remove related firewall rules, regenerate the WireGuard config (excluding server clients), and sync to the `wg` interface.
*   **Delete:** Perform the same actions as disabling the server (remove IP/firewall/clients from `wg` config and sync), then delete the server.
*   **Move:** Move the server to another `wg` interface. This is the same as deleting it from the old interface and adding it to the new one. Since the network does not change, firewall rules do not need modification; we only need to sync the WireGuard interfaces.

### Client:
*   **Name:** A name tag for this client, which will be shown in the frontend.
*   **DNS:** DNS settings for this client. If null, it follows the server's settings.
*   **ID:** The unique ID of the client, allocated when added. It is not changeable.
*   **Enabled:** A boolean indicating if this client is enabled. This value cannot be edited via the standard 'edit' API; it can only be modified using the dedicated 'SetEnable' API. When created, it will always be disabled.
*   **IP:** (`null`: disable IPv4. `auto`: auto-allocate using the algorithm described in the Server section. `valid_ipv4_address`: manually assign).
    *   It stores `IPv4_offset` internally. Use IPNetWrapper to store the value. The real IP is calculated at runtime (`IPv4.Network + IPv4_offset`. Write a addiction function to add two ipv4 (convert to uint32 then add then convert back)) and is not stored directly. The offset cannot exceed `2^(32 - mask_len)`.
    *   Can't be accessed by external directly. Can only use SetIP or GetIP which convert to offset internally and convert it back when GetIP
*   **IPv6:** (`null`: disable IPv6. `auto`: auto-allocate. `valid_ipv6_address`: manually assign). At least one of IPv4 or IPv6 must be enabled.
    *   It stores `IPv6_offset` internally. Use IPNetWrapper to store the value. The real IPv6 is calculated at runtime (`IPv6.Network + IPv6_offset`. Write a byte[16] array addition function to add two ipv6) and is not stored directly. The offset cannot exceed `2^(128 - mask_len)`.
    *   Can't be accessed by external directly. Can only use SetIPv6 or GetIPv6 which convert to offset internally and convert it back when GetIPv6.
*   **PrivateKey:** If both `PrivateKey` and `PublicKey` are null, a new keypair is randomly generated.
*   **PublicKey:** If `PrivateKey` is not null, `PublicKey` is ignored and stored as null.
*   **PresharedKey:** Optional. If null, a preshared key is not used.
*   **Keepalive:** An optional `uint` for the persistent keepalive interval.

#### Actions:
*   **New:** Create a new client.
*   **Get:** Get the properties of this client. If `PrivateKey` is not null, `PublicKey` will be calculated from it.
*   **GetWGState:** Get `last_handshake`, transfer/received data, and the endpoint from the `wg` interface for this client. All three values may be empty (e.g., if `wg` is initialized but the client has not connected yet).
*   **Edit:** Edit client properties.
    *   When **IP, IPv6, PrivateKey, PublicKey, or PresharedKey** are edited, the `AllowedIPs` will be updated, so the WireGuard configuration should be regenerated and synced to the `wg` interface.
*   **SetEnable:**
    *   **true:** Set `Enabled` to `true`, include this client as a peer in the config, and sync to the `wg` interface.
    *   **false:** Set `Enabled` to `false`, exclude this client as a peer, and sync to the `wg` interface.
*   **Delete:** Perform the same actions as disabling the client (exclude from `wg` config and sync), then delete the client.
*   **GetConfig:** Get the WireGuard configuration for this client. If `PrivateKey` is null, the config will show `[privkey is not available for this client]`.

### Pseudo-bridge Service
This is a goroutine that starts with the server and handles the pseudo-bridge feature.
This service has two parts: Pseudo-bridge Service Master and Interface Responder, and use two variable PseudoBridgeWaitInterface/PseudoBridgeRunningInterface to sync data between them.
PseudoBridgeWaitInterface/PseudoBridgeRunningInterface is a list in a map in a map, [ifname][v4/v6][network1, network2, network3 ...]
Pseudo-bridge Service Master:
    Spawn a Interface Responder which is a new new goroutine and stop unnecessary responders when there is no need.
    Event driven, Start/Update and Stop.
    Update: get a PseudoBridgeWaitInterface then sync to PseudoBridgeRunningInterface. 
    During the comparing, get the interface and networks changes. if PseudoBridgeWaitInterface and PseudoBridgeRunningInterface are same, it is read locked, and do nothing.
    If they are different, lock and sync PseudoBridgeWaitInterface to PseudoBridgeRunningInterface. Spawn a Interface Responder if new interface added, clean unnecessary Interface Responder if the interface no longer nessesary, and update the networks to existing responders.
The Interface Responder:
    It reads from networks for ARP-Reply or Neighbor Advertisement packet generation.
    Monitoring ARP-request and Neighbor Solicitation packets on assigned interfaces using pcap.
    If it asks MAC address for the ip which is in the server networks which needs to be pseudo-bridged(in the PseudoBridgeRunningInterface of the interface), it generates an ARP Reply or Neighbor Advertisement packet to reply.
    Event-driven, With Start, Update, Stop events.
    Start: Start monitoring on the interface. If interface not exist, wait 5 second and try again. If the device vanished during running, wait 5 second and try again.
    Update: Update the networks
    Stop: Stop the Responder.

### SNAT Roaming Service
This is a goroutine that starts with the server and handles the SNAT Roaming feature.
This service has two parts: SNAT Roaming Service Master and Interface IPNet Listener. And use two variable SNATRoamingWaitInterface/SNATRoamingRunningInterface to sync data between them.
SNATRoamingWaitInterface/SNATRoamingRunningInterface is a list in a map, [ifname] [{IPv6Net offset 1, CommentString1}, {IPv6Net offset 2,CommentString2}, {IPv6Net offset 3,CommentString3} ...]
SNAT Roaming Service Master:
    Spawns a Interface IPNet Listener which is a new new goroutine, and stop unnecessary listener when there is no need.
    Event driven, Start/Update and Stop.
    Update: get a SNATRoamingWaitInterface then sync to SNATRoamingRunningInterface. 
    During the comparing, get the interface and IPv6Net offset changes. if SNATRoamingWaitInterface and SNATRoamingRunningInterface are same, it is read locked, and do nothing.
    If they are different, lock sync SNATRoamingWaitInterface to SNATRoamingRunningInterface. Spawn a Interface IPNet Listener if new interface added, clean unnecessary Interface IPNet Listener if the interface no longer nessesary, and update IPv6Net offset to existing listeners.
Interface IPNet Listener:
    It detects IP/IPv6/IPv6Net changed with netlink on assigned interface. If change, trigger the sync.
    If the IP/IPv6/IPv6Net changed, calculate the real IPv6Net ( a.k.a target_network ) and add/update/delete the firewall rules for IPv4 SNAT/IPv6 SNAT/IPv6 SNAT NETMAP.
    Event-driven, With Start, Update, Stop events.
    Start: Start monitoring on the interface. If interface not exist, wait 5 second and try again. If the device vanished during running, wait 5 second and try again.
    Update: Update the IPv6Net offsets
    Stop: Stop the Responder.
    

### Some Golang code to calculate the public key:

```go
import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"

	"golang.org/x/crypto/curve25519"
)

func GenerateWGPrivateKey() (string, error) {
	var privateKey [32]byte
	_, err := rand.Read(privateKey[:])
	if err != nil {
		return "", fmt.Errorf("failed to generate random data for private key: %v", err)
	}
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64
	return base64.StdEncoding.EncodeToString(privateKey[:]), nil
}

func PrivToPublic(privateKeyB64 string) (string, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 private key: %w", err)
	}
	if len(privateKeyBytes) != 32 {
		return "", fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(privateKeyBytes))
	}
	var privateKey [32]byte
	copy(privateKey[:], privateKeyBytes)
	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)
	return base64.StdEncoding.EncodeToString(publicKey[:]), nil
}
```

This is an example OpenAPI v3 for this backend
```
# This is a pert of prompt, for reference only, don't move this file. If nessesary, use copy instead of mv
@prompt/API_SPEC.yaml
```

Finish the backend. Complete all features above with production ready, don't generate any placeholders.
Then build the project, fix errors, implement all features.
