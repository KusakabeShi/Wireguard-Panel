package internalservice

import (
	"fmt"
	"strings"

	"wg-panel/internal/logging"
	"wg-panel/internal/models"
	"wg-panel/internal/utils"
)

type FirewallService struct{}

func NewFirewallService() *FirewallService {
	return &FirewallService{}
}

func (f *FirewallService) AddIpAndFwRules(interfaceName string, config *models.ServerNetworkConfig) error {
	if config == nil || !config.Enabled {
		return nil
	}

	logging.LogInfo("Adding firewall rules and IP configuration for interface %s", interfaceName)

	comment := config.CommentString

	// Add IP address to interface (only if not already present)
	if config.Network != nil {
		if err := f.addIPAddressIfNotExists(interfaceName, config.Network.String()); err != nil {
			return fmt.Errorf("failed to add IP address: %v", err)
		}
	}

	// Add SNAT rules
	if config.Snat != nil && config.Snat.Enabled {
		if config.Snat.RoamingMasterInterface != nil && *config.Snat.RoamingMasterInterface != "" {
			// If roaming is enabled, SNAT rules must managed by the roaming service
		} else {
			if err := f.AddSnatRules(config, comment); err != nil {
				return fmt.Errorf("failed to add SNAT rules: %v", err)
			}
		}
	}

	// Add routed networks firewall rules
	if config.RoutedNetworksFirewall && len(config.RoutedNetworks) > 0 {
		if err := f.addRoutedNetworksRules(interfaceName, config, comment); err != nil {
			return fmt.Errorf("failed to add routed networks rules: %v", err)
		}
	}

	return nil
}

func (f *FirewallService) RemoveIpAndFwRules(interfaceName string, config *models.ServerNetworkConfig) {
	if config == nil || !config.Enabled {
		return
	}

	logging.LogInfo("Removing firewall rules and IP configuration for interface %s", interfaceName)

	interfaceDevice := fmt.Sprintf("%s", interfaceName)
	comment := config.CommentString

	// Remove IP address from interface (only if it exists)
	if config.Network != nil {
		f.removeIPAddressIfExists(interfaceDevice, config.Network.String())
	}

	// Remove firewall rules by comment
	err := utils.CleanupRules(comment, config.Network.Version, nil, false)
	if err != nil {
		logging.LogError("Failed to remove firewall rules: %v", err)
	}
}

func (f *FirewallService) AddSnatRules(config *models.ServerNetworkConfig, comment string) error {
	if config == nil || !config.Enabled || config.Network == nil || config.Snat == nil {
		return nil
	}

	isIPv4 := config.Network.Version == 4
	iptablesCmd := "iptables"
	if !isIPv4 {
		iptablesCmd = "ip6tables"
	}
	if config.Snat.RoamingMasterInterface != nil && *config.Snat.RoamingMasterInterface != "" {
		// If roaming is enabled, SNAT rules must managed by the roaming service
		return fmt.Errorf("cannot add SNAT rules: roaming is enabled, rules must be managed by the roaming service")
	}
	// Generate SNAT rules using shared function
	rules := utils.GenerateSNATRules(iptablesCmd, config, comment)

	// Apply each rule
	for _, rule := range rules {
		ruleArgs := strings.Fields(rule)
		// Remove the iptables command from the beginning since we pass it separately
		if len(ruleArgs) > 0 && ruleArgs[0] == iptablesCmd {
			ruleArgs = ruleArgs[1:]
		}
		if err := f.addIptablesRuleIfNotExists(iptablesCmd, ruleArgs); err != nil {
			return fmt.Errorf("failed to add SNAT rule: %v", err)
		}
	}

	return nil
}

func (f *FirewallService) RemoveSnatRules(af int, comment string) error {
	if err := utils.CleanupRules(comment, af, &[]string{"nat"}, false); err != nil {
		return fmt.Errorf("failed to remove SNAT rules: %v", err)
	}
	return nil
}

func (f *FirewallService) addRoutedNetworksRules(interfaceDevice string, config *models.ServerNetworkConfig, comment string) error {
	if config.Network == nil || len(config.RoutedNetworks) == 0 {
		return nil
	}

	isIPv4 := config.Network.Version == 4
	iptablesCmd := "iptables"
	if !isIPv4 {
		iptablesCmd = "ip6tables"
	}

	// Generate routed networks rules using shared function
	rules := utils.GenerateRoutedNetworksRules(iptablesCmd, interfaceDevice, config, comment)

	// Apply each rule
	for _, rule := range rules {
		ruleArgs := strings.Fields(rule)
		// Remove the iptables command from the beginning since we pass it separately
		if len(ruleArgs) > 0 && ruleArgs[0] == iptablesCmd {
			ruleArgs = ruleArgs[1:]
		}
		if err := f.addIptablesRuleIfNotExists(iptablesCmd, ruleArgs); err != nil {
			return fmt.Errorf("failed to add routed network rule: %v", err)
		}
	}

	return nil
}

// addIPAddressIfNotExists adds an IP address to an interface only if it doesn't already exist
func (f *FirewallService) addIPAddressIfNotExists(interfaceDevice, ipAddr string) error {
	// Check if the IP address already exists on the interface
	if exists, err := f.ipAddressExists(interfaceDevice, ipAddr); err != nil {
		return fmt.Errorf("failed to check IP address existence: %v", err)
	} else if exists {
		// IP address already exists, skip addition
		return nil
	}

	// Add the IP address
	logging.LogInfo("Adding IP address %s to interface %s", ipAddr, interfaceDevice)
	if err := utils.RunCommand("ip", "addr", "add", ipAddr, "dev", interfaceDevice); err != nil {
		return err
	}

	return nil
}

// removeIPAddressIfExists removes an IP address from an interface only if it exists
func (f *FirewallService) removeIPAddressIfExists(interfaceDevice, ipAddr string) {
	// Check if the IP address exists on the interface
	if exists, err := f.ipAddressExists(interfaceDevice, ipAddr); err != nil || !exists {
		// Either error checking or doesn't exist, skip removal
		return
	}

	// Remove the IP address
	logging.LogInfo("Removing IP address %s from interface %s", ipAddr, interfaceDevice)
	utils.RunCommandIgnoreError("ip", "addr", "del", ipAddr, "dev", interfaceDevice)
}

// ipAddressExists checks if an IP address is assigned to a specific interface
func (f *FirewallService) ipAddressExists(interfaceDevice, ipAddr string) (bool, error) {
	output, err := utils.RunCommandWithOutput("ip", "addr", "show", "dev", interfaceDevice)
	if err != nil {
		// If interface doesn't exist, return false with no error
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "Device not found") {
			return false, nil
		}
		return false, err
	}

	// Parse the IP address to get just the network part for comparison
	// Format: "192.168.1.1/24" -> we need to match this exactly
	return strings.Contains(output, ipAddr), nil
}

// addIptablesRuleIfNotExists adds an iptables rule only if it doesn't already exist
func (f *FirewallService) addIptablesRuleIfNotExists(iptablesCmd string, ruleArgs []string) error {
	// Check if the rule already exists
	if exists, err := f.iptablesRuleExists(iptablesCmd, ruleArgs); err != nil {
		return fmt.Errorf("failed to check iptables rule existence: %v", err)
	} else if exists {
		// Rule already exists, skip addition
		return nil
	}

	// Add the rule
	logging.LogInfo("Adding firewall rule: %s %s", iptablesCmd, strings.Join(ruleArgs, " "))
	if err := utils.RunCommand(iptablesCmd, ruleArgs...); err != nil {
		return err
	}

	return nil
}

// iptablesRuleExists checks if an iptables rule already exists
func (f *FirewallService) iptablesRuleExists(iptablesCmd string, ruleArgs []string) (bool, error) {
	// Convert -A (append) to -C (check) to test if rule exists
	checkArgs := make([]string, len(ruleArgs))
	copy(checkArgs, ruleArgs)

	// Find and replace -A with -C
	for i, arg := range checkArgs {
		if arg == "-A" && i+1 < len(checkArgs) {
			checkArgs[i] = "-C"
			break
		}
	}

	// Handle -t (table) argument position for check command
	// iptables -t nat -C POSTROUTING ... (correct)
	// vs iptables -C -t nat POSTROUTING ... (incorrect)
	var finalArgs []string
	tableIdx := -1
	for i, arg := range checkArgs {
		if arg == "-t" && i+1 < len(checkArgs) {
			tableIdx = i
			break
		}
	}

	if tableIdx >= 0 {
		// Put table arguments first: iptables -t nat -C ...
		finalArgs = append(finalArgs, checkArgs[tableIdx:tableIdx+2]...) // -t nat
		// Add -C and the rest
		for i, arg := range checkArgs {
			if i != tableIdx && i != tableIdx+1 && arg != "-t" {
				finalArgs = append(finalArgs, arg)
			}
		}
	} else {
		finalArgs = checkArgs
	}

	// Run the check command
	err := utils.RunCommand(iptablesCmd, finalArgs...)
	if err == nil {
		// Rule exists (command succeeded)
		return true, nil
	}

	// Check if the error is "rule does not exist" vs actual error
	if cmdErr, ok := err.(*utils.CommandError); ok {
		// Exit code 1 typically means "rule not found" for iptables -C
		// Exit codes > 1 usually indicate actual errors
		if cmdErr.ExitCode == 1 {
			return false, nil
		}
	}

	// Some other error occurred
	return false, err
}
