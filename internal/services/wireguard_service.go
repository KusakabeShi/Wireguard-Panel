package services

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"wg-panel/internal/logging"
	"wg-panel/internal/models"
	"wg-panel/internal/utils"
)

type WireGuardService struct {
	configPath string
}

func NewWireGuardService(configPath string) *WireGuardService {
	return &WireGuardService{
		configPath: configPath,
	}
}

func (s *WireGuardService) SyncToConfAndInterface(iface *models.Interface) error {
	// Generate standalone configuration with firewall rules
	if err := s.SyncToConf(iface); err != nil {
		return err
	}
	// Apply configuration using wg-quick or wg syncconf
	return s.SyncToInterface(iface.Ifname, iface.Enabled, iface.PrivateKey)
}

func (s *WireGuardService) SyncToConf(iface *models.Interface) error {
	// Generate standalone configuration with firewall rules
	config := s.GenerateConf(iface)

	// Write configuration file to wireguardConfigPath
	configFile := filepath.Join(s.configPath, fmt.Sprintf("%s.conf", iface.Ifname))
	if err := utils.WriteFileAtomic(configFile, []byte(config), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	logging.LogInfo("Generated standalone WireGuard configuration at %s", configFile)
	return nil
}

func (s *WireGuardService) GenerateConf(iface *models.Interface) string {
	var config strings.Builder

	config.WriteString("[Interface]\n")
	config.WriteString(fmt.Sprintf("PrivateKey = %s\n", iface.PrivateKey))
	config.WriteString(fmt.Sprintf("ListenPort = %d\n", iface.Port))

	if iface.FwMark != nil && *iface.FwMark != "" {
		config.WriteString(fmt.Sprintf("FwMark = %s\n", *iface.FwMark))
	}

	// Add IP addresses from enabled servers
	addresses := make([]string, 0)
	for _, server := range iface.Servers {
		if !server.Enabled {
			continue
		}
		if server.IPv4 != nil && server.IPv4.Enabled && server.IPv4.Network != nil {
			addresses = append(addresses, server.IPv4.Network.String())
		}
		if server.IPv6 != nil && server.IPv6.Enabled && server.IPv6.Network != nil {
			addresses = append(addresses, server.IPv6.Network.String())
		}
	}

	if len(addresses) > 0 {
		config.WriteString(fmt.Sprintf("Address = %s\n", strings.Join(addresses, ", ")))
	}

	// Generate PostUp and PreDown commands
	postUpCommands := s.generatePostUpCommands(iface, true)
	preDownCommands := s.generatePreDownCommands(iface)

	// Add each PostUp command as a separate line
	for _, cmd := range postUpCommands {
		config.WriteString(fmt.Sprintf("PostUp = %s\n", cmd))
	}

	// Add each PreDown command as a separate line
	for _, cmd := range preDownCommands {
		config.WriteString(fmt.Sprintf("PreDown = %s\n", cmd))
	}

	config.WriteString("\n")

	// Add peers (enabled clients)
	for _, server := range iface.Servers {
		if !server.Enabled {
			continue
		}
		for _, client := range server.Clients {
			if !client.Enabled {
				continue
			}

			config.WriteString("[Peer]\n")
			config.WriteString(fmt.Sprintf("PublicKey = %s\n", client.PublicKey))

			if client.PresharedKey != nil && *client.PresharedKey != "" {
				config.WriteString(fmt.Sprintf("PresharedKey = %s\n", *client.PresharedKey))
			}

			// Calculate AllowedIPs
			allowedIPs := s.calculateAllowedIPs(client, server)
			if len(allowedIPs) > 0 {
				config.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(allowedIPs, ", ")))
			}

			if client.Keepalive != nil && *client.Keepalive > 0 {
				config.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", *client.Keepalive))
			}

			config.WriteString("\n")
		}
	}

	return config.String()
}

func (s *WireGuardService) calculateAllowedIPs(client *models.Client, server *models.Server) []string {
	allowedIPs := make([]string, 0)

	if server.IPv4 != nil && server.IPv4.Enabled && client.IPv4Offset != nil {
		if ipNet, err := client.GetIPv4(server.IPv4.Network); err == nil && ipNet != nil {
			allowedIPs = append(allowedIPs, fmt.Sprintf("%s/32", ipNet.IP.String()))
		}
	}

	if server.IPv6 != nil && server.IPv6.Enabled && client.IPv6Offset != nil {
		if ipNet, err := client.GetIPv6(server.IPv6.Network); err == nil && ipNet != nil {
			allowedIPs = append(allowedIPs, fmt.Sprintf("%s/128", ipNet.IP.String()))
		}
	}

	return allowedIPs
}

func (s *WireGuardService) SyncToInterface(ifname string, enabled bool, checkWgKey string) (err error) {
	interfaceName := fmt.Sprintf("%s", ifname)
	configFile := filepath.Join(s.configPath, fmt.Sprintf("%s.conf", ifname))
	wgPubkey := ""
	if checkWgKey != "" {
		wgPubkey, err = utils.PrivToPublic(checkWgKey)
		if err != nil {
			return fmt.Errorf("failed to derive public key from private key: %v", err)
		}
	}

	// Check if interface exists
	interfaceExists := utils.RunCommand("ip", "link", "show", interfaceName) == nil

	if enabled {
		if !interfaceExists {
			// Situation 1: Interface not found and enable=true, wg-quick up normally
			logging.LogInfo("Bringing up WireGuard interface %s", interfaceName)
			if err := utils.RunCommand("wg-quick", "up", configFile); err != nil {
				return fmt.Errorf("failed to bring up interface with wg-quick: %v", err)
			}
		} else {
			// Situation 2: Interface exists, check if it's a WireGuard interface and public key
			if !s.isTargetWgInterface(interfaceName, wgPubkey) {
				if wgPubkey != "" {
					return fmt.Errorf("interface %s is not the target WireGuard interface (public key mismatch)", interfaceName)
				} else {
					return fmt.Errorf("interface %s exists but is not a WireGuard interface", interfaceName)
				}
			}

			// Use Go native approach: first strip config, then sync
			strippedConfig, err := s.stripWgQuickConfig(configFile)
			if err != nil {
				return fmt.Errorf("failed to strip config: %v", err)
			}

			logging.LogInfo("Syncing configuration to WireGuard interface %s", interfaceName)
			if err := s.syncConfToInterface(interfaceName, strippedConfig); err != nil {
				return fmt.Errorf("failed to sync interface configuration: %v", err)
			}
		}
	} else {
		// enable=false (disable the target wg conf)
		if !interfaceExists {
			// Situation 3: Interface not exists, skip
			return nil
		} else {
			// Situation 4: Interface exists, check if it's WireGuard and public key
			if !s.isTargetWgInterface(interfaceName, wgPubkey) {
				if wgPubkey != "" {
					return fmt.Errorf("interface %s is not the target WireGuard interface (public key mismatch)", interfaceName)
				} else {
					return fmt.Errorf("interface %s exists but is not a WireGuard interface", interfaceName)
				}
			}

			// Use wg-quick down to properly disable the interface
			logging.LogInfo("Bringing down WireGuard interface %s", interfaceName)
			if err := utils.RunCommand("wg-quick", "down", configFile); err != nil {
				return fmt.Errorf("failed to disable interface: %v", err)
			}
		}
	}

	return nil
}

func (s *WireGuardService) RemoveConfig(ifname string) error {
	logging.LogInfo("Removing WireGuard configuration for interface %s", ifname)
	configFile := filepath.Join(s.configPath, fmt.Sprintf("%s.conf", ifname))

	// Remove config file
	logging.LogInfo("Removing WireGuard config file %s", configFile)
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %v", err)
	}

	return nil
}

func (s *WireGuardService) SetInterfaceMTU(ifname string, mtu int) error {

	// Check if interface exists before trying to set MTU
	if err := utils.RunCommand("ip", "link", "show", ifname); err != nil {
		return fmt.Errorf("interface %s does not exist: %v", ifname, err)
	}

	if err := utils.RunCommand("ip", "link", "set", "dev", ifname, "mtu", fmt.Sprintf("%d", mtu)); err != nil {
		return fmt.Errorf("failed to set MTU: %v", err)
	}
	return nil
}

func (s *WireGuardService) GetPeerStats(interfaceName string) (map[string]*models.WGState, error) {
	output, err := utils.RunCommandWithOutput("wg", "show", fmt.Sprintf("%s", interfaceName), "dump")
	if err != nil {
		return nil, fmt.Errorf("failed to get peer stats: %v", err)
	}

	stats := make(map[string]*models.WGState)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 8 {
			continue
		}

		publicKey := parts[0]
		if publicKey == "" {
			continue
		}

		state := &models.WGState{}

		// Parse endpoint
		if parts[2] != "(none)" && parts[2] != "" {
			endpoint := parts[2]
			state.Endpoint = &endpoint
		}

		// Parse latest handshake
		if parts[4] != "0" && parts[4] != "" {
			if timestamp := parseUnixTimestamp(parts[4]); timestamp != nil {
				state.LatestHandshake = timestamp
			}
		}

		// Parse transfer stats
		if parts[5] != "0" && parts[5] != "" {
			if rx := parseInt64(parts[5]); rx != nil {
				state.TransferRx = rx
			}
		}

		if parts[6] != "0" && parts[6] != "" {
			if tx := parseInt64(parts[6]); tx != nil {
				state.TransferTx = tx
			}
		}

		stats[publicKey] = state
	}

	return stats, nil
}

func parseUnixTimestamp(ts string) *time.Time {
	if timestamp := parseInt64(ts); timestamp != nil {
		t := time.Unix(*timestamp, 0)
		return &t
	}
	return nil
}

func parseInt64(s string) *int64 {
	if val := strings.TrimSpace(s); val != "" && val != "0" {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return &i
		}
	}
	return nil
}

// isTargetWgInterface checks if the given interface is a WireGuard interface
// If key is provided, also checks if the public key matches
func (s *WireGuardService) isTargetWgInterface(ifname string, key string) bool {
	// First check if it's a WireGuard interface
	err := utils.RunCommand("wg", "show", ifname)
	if err != nil {
		return false
	}

	// If no key check is needed, return true
	if key == "" {
		return true
	}

	// Get the interface's public key and compare
	output, err := utils.RunCommandWithOutput("wg", "show", ifname, "public-key")
	if err != nil {
		return false
	}

	currentPubKey := strings.TrimSpace(output)
	return currentPubKey == key
}

func (s *WireGuardService) generatePostUpCommands(iface *models.Interface, ifnameUsePI bool) (commands []string) {

	// Add VRF configuration if specified
	if iface.VRFName != nil && *iface.VRFName != "" {
		commands = append(commands, fmt.Sprintf("ip link set dev %s master %s", iface.Ifname, *iface.VRFName))
	}

	ifacename := iface.Ifname
	if ifnameUsePI {
		ifacename = "%i"
	}

	// Add firewall rules for each enabled server
	for _, server := range iface.Servers {
		if !server.Enabled {
			continue
		}

		// IPv4 firewall rules
		if server.IPv4 != nil && server.IPv4.Enabled {
			commands = append(commands, utils.GenerateServerFirewallRules(ifacename, server.IPv4, 4)...)
		}

		// IPv6 firewall rules
		if server.IPv6 != nil && server.IPv6.Enabled {
			commands = append(commands, utils.GenerateServerFirewallRules(ifacename, server.IPv6, 6)...)
		}
	}

	// Create a script that executes all commands
	return
}

func (s *WireGuardService) generatePreDownCommands(iface *models.Interface) (commands []string) {
	// Remove firewall rules for each enabled server
	for _, server := range iface.Servers {
		if !server.Enabled {
			continue
		}

		// Remove IPv4 firewall rules
		if server.IPv4 != nil && server.IPv4.Enabled && server.IPv4.CommentString != "" {
			commands = append(commands, utils.GenerateCleanupRules(server.IPv4.CommentString, 4)...)
		}

		// Remove IPv6 firewall rules
		if server.IPv6 != nil && server.IPv6.Enabled && server.IPv6.CommentString != "" {
			commands = append(commands, utils.GenerateCleanupRules(server.IPv6.CommentString, 6)...)
		}
	}

	// Remove VRF configuration if specified
	if iface.VRFName != nil && *iface.VRFName != "" {
		commands = append(commands, fmt.Sprintf("ip link set dev %s nomaster", iface.Ifname))
	}
	// Create a script that executes all commands (ignore errors on cleanup)
	return
}

// stripWgQuickConfig executes wg-quick strip and returns the stripped configuration
func (s *WireGuardService) stripWgQuickConfig(configFile string) (string, error) {
	output, err := utils.RunCommandWithOutput("wg-quick", "strip", configFile)
	if err != nil {
		return "", fmt.Errorf("failed to strip config file %s: %v", configFile, err)
	}
	return output, nil
}

// syncConfToInterface applies the stripped configuration to the interface using wg syncconf
func (s *WireGuardService) syncConfToInterface(interfaceName, strippedConfig string) error {
	cmd := exec.Command("wg", "syncconf", interfaceName, "/dev/stdin")
	cmd.Stdin = strings.NewReader(strippedConfig)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to sync config to interface %s: %v, stderr: %s", interfaceName, err, stderr.String())
	}

	return nil
}
