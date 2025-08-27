package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

func (s *WireGuardService) GenerateAndSyncInterface(iface *models.Interface) error {
	// Generate standalone configuration with firewall rules
	config := s.generateInterfaceConfig(iface)

	// Write configuration file to wireguardConfigPath
	configFile := filepath.Join(s.configPath, fmt.Sprintf("%s.conf", iface.Ifname))
	if err := utils.WriteFileAtomic(configFile, []byte(config), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	log.Printf("Generated standalone WireGuard configuration at %s", configFile)

	// Apply configuration using wg-quick or wg syncconf
	return s.syncInterface(iface.Ifname, configFile)
}

func (s *WireGuardService) generateInterfaceConfig(iface *models.Interface) string {
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
	postUpCommands := s.generatePostUpCommands(iface)
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

func (s *WireGuardService) syncInterface(ifname string, configFile string) error {
	interfaceName := fmt.Sprintf("%s", ifname)

	// Check if interface exists
	if err := utils.RunCommand("ip", "link", "show", interfaceName); err != nil {
		// Interface doesn't exist, use wg-quick up
		if err := utils.RunCommand("wg-quick", "up", configFile); err != nil {
			return fmt.Errorf("failed to bring up interface with wg-quick: %v", err)
		}
	} else {
		// Interface exists, use wg syncconf
		syncCmd := fmt.Sprintf("wg syncconf %s <(wg-quick strip %s)", interfaceName, configFile)
		if err := utils.RunCommand("bash", "-c", syncCmd); err != nil {
			return fmt.Errorf("failed to sync interface configuration: %v", err)
		}
	}

	return nil
}

func (s *WireGuardService) RemoveInterface(ifname string) error {
	configFile := filepath.Join(s.configPath, fmt.Sprintf("%s.conf", ifname))

	// Check if interface exists before trying to remove it
	if err := utils.RunCommand("ip", "link", "show", ifname); err != nil {
		// Interface doesn't exist, skip removal but continue with config file cleanup
		// This is not an error - interface might have been removed already
	} else {
		// Interface exists, try to remove it
		// Use wg-quick down to remove interface
		if err := utils.RunCommand("wg-quick", "down", configFile); err != nil {
			// Try to remove manually if wg-quick fails
			if err := utils.RunCommand("ip", "link", "delete", ifname); err != nil {
				return fmt.Errorf("failed to remove interface: %v", err)
			}
		}
	}

	// Remove config file
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
		if len(parts) < 9 {
			continue
		}

		publicKey := parts[1]
		if publicKey == "" {
			continue
		}

		state := &models.WGState{}

		// Parse endpoint
		if parts[3] != "(none)" && parts[3] != "" {
			endpoint := parts[3]
			state.Endpoint = &endpoint
		}

		// Parse latest handshake
		if parts[5] != "0" && parts[5] != "" {
			if timestamp := parseUnixTimestamp(parts[5]); timestamp != nil {
				state.LatestHandshake = timestamp
			}
		}

		// Parse transfer stats
		if parts[6] != "0" && parts[6] != "" {
			if rx := parseInt64(parts[6]); rx != nil {
				state.TransferRx = rx
			}
		}

		if parts[7] != "0" && parts[7] != "" {
			if tx := parseInt64(parts[7]); tx != nil {
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

func (s *WireGuardService) generatePostUpCommands(iface *models.Interface) (commands []string) {

	// Add VRF configuration if specified
	if iface.VRFName != nil && *iface.VRFName != "" {
		commands = append(commands, fmt.Sprintf("ip link set dev %s master %s", iface.Ifname, *iface.VRFName))
	}

	// Add firewall rules for each enabled server
	for _, server := range iface.Servers {
		if !server.Enabled {
			continue
		}

		// IPv4 firewall rules
		if server.IPv4 != nil && server.IPv4.Enabled {
			commands = append(commands, utils.GenerateServerFirewallRules(iface.Ifname, server.IPv4, 4)...)
		}

		// IPv6 firewall rules
		if server.IPv6 != nil && server.IPv6.Enabled {
			commands = append(commands, utils.GenerateServerFirewallRules(iface.Ifname, server.IPv6, 6)...)
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
