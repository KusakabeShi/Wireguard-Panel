package utils

import (
	"fmt"
	"strings"

	"wg-panel/internal/logging"
	"wg-panel/internal/models"
)

func GenerateServerFirewallRules(interfaceName string, config *models.ServerNetworkConfig, version int) []string {
	if config == nil || !config.Enabled {
		return []string{}
	}

	var rules []string
	comment := config.CommentString
	iptablesCmd := "iptables"
	if version == 6 {
		iptablesCmd = "ip6tables"
	}

	// Add SNAT rules
	if config.Snat != nil && config.Snat.Enabled {
		if config.Snat.RoamingMasterInterface != nil && *config.Snat.RoamingMasterInterface != "" {
			// If roaming is enabled, SNAT rules must managed by the roaming service
		} else {
			rules = append(rules, GenerateSNATRules(iptablesCmd, config, comment)...)
		}
	}

	// Add routed networks firewall rules
	if config.RoutedNetworksFirewall && len(config.RoutedNetworks) > 0 {
		rules = append(rules, GenerateRoutedNetworksRules(iptablesCmd, interfaceName, config, comment)...)
	}

	return rules
}

func GenerateSNATRules(iptablesCmd string, config *models.ServerNetworkConfig, comment string) []string {
	if config.Network == nil || config.Snat == nil {
		return []string{}
	}

	var rules []string
	sourceNet := config.Network.NetworkStr()
	excludedNet := sourceNet
	if config.Snat.SnatExcludedNetwork != nil {
		if config.Snat.SnatExcludedNetwork.EqualZero(4) || config.Snat.SnatExcludedNetwork.EqualZero(6) {
			excludedNet = ""
		} else {
			excludedNet = config.Snat.SnatExcludedNetwork.NetworkStr()
		}
	}

	destExclusion := ""
	if excludedNet != "" {
		destExclusion = fmt.Sprintf("! -d %s ", excludedNet)
	}

	if config.Snat.RoamingMasterInterface != nil && *config.Snat.RoamingMasterInterface != "" {
		// If roaming is enabled, SNAT rules must managed by the roaming service
		return []string{}
	}

	if config.Snat.SnatIPNet == nil {
		// MASQUERADE mode
		rules = append(rules, fmt.Sprintf("%s -t nat -A POSTROUTING -s %s %s -j MASQUERADE -m comment --comment %s",
			iptablesCmd, sourceNet, destExclusion, comment))
	} else if config.Network.Version == 4 {
		if config.Snat.SnatIPNet.Masklen() != 32 {
			// Error, invalid IPv4 SNAT configuration, masklen must be /32
			return []string{}
		} else if config.Snat.SnatIPNet.EqualZero(4) {
			// Error, invalid IPv4 SNAT configuration, can't be 0.0.0.0/32
			return []string{}
		}
		rules = append(rules, fmt.Sprintf("%s -t nat -A POSTROUTING -s %s %s -j SNAT --to-source %s -m comment --comment %s",
			iptablesCmd, sourceNet, destExclusion, config.Snat.SnatIPNet.IP.String(), comment))
	} else if config.Network.Version == 6 && config.Snat.SnatIPNet.Masklen() == 128 {
		if config.Snat.SnatIPNet.EqualZero(6) {
			// Error, invalid IPv6 SNAT configuration, can't be ::/128
			return []string{}
		}
		// SNAT mode (IPv4 /32 or IPv6 /128)
		rules = append(rules, fmt.Sprintf("%s -t nat -A POSTROUTING -s %s %s -j SNAT --to-source %s -m comment --comment %s",
			iptablesCmd, sourceNet, destExclusion, config.Snat.SnatIPNet.IP.String(), comment))
	} else {
		// IPv6 NETMAP mode
		serverNetwork := config.Network.NetworkStr()
		targetNetwork := config.Snat.SnatIPNet.NetworkStr()

		rules = append(rules, fmt.Sprintf("%s -t nat -A POSTROUTING -s %s %s -j NETMAP --to %s -m comment --comment %s",
			iptablesCmd, serverNetwork, destExclusion, targetNetwork, comment))
		rules = append(rules, fmt.Sprintf("%s -t nat -A PREROUTING -d %s -j NETMAP --to %s -m comment --comment %s",
			iptablesCmd, targetNetwork, serverNetwork, comment))
	}

	return rules
}

func GenerateRoutedNetworksRules(iptablesCmd string, ifname string, config *models.ServerNetworkConfig, comment string) []string {
	if config.Network == nil || len(config.RoutedNetworks) == 0 {
		return []string{}
	}

	var rules []string
	sourceNet := config.Network.NetworkStr()

	// Check if we have a "allow all" network
	hasAllowAll := false
	for _, routedNet := range config.RoutedNetworks {
		if (config.Network.Version == 4 && routedNet.String() == "0.0.0.0/0") ||
			(config.Network.Version == 6 && routedNet.String() == "::/0") {
			hasAllowAll = true
			break
		}
	}

	if !hasAllowAll {
		// Add specific allow rules for each routed network
		for _, routedNet := range config.RoutedNetworks {
			rules = append(rules, fmt.Sprintf("%s -A FORWARD -i %s -s %s -d %s -j ACCEPT -m comment --comment %s",
				iptablesCmd, ifname, sourceNet, routedNet.NetworkStr(), comment))
		}

		// Add deny rule for other destinations
		rules = append(rules, fmt.Sprintf("%s -A FORWARD -i %s -s %s -j REJECT -m comment --comment %s",
			iptablesCmd, ifname, sourceNet, comment))
	}

	return rules
}

func GenerateCleanupRules(comment string, version int) []string {
	iptablesCmd := "iptables"
	if version == 6 {
		iptablesCmd = "ip6tables"
	}

	return []string{
		fmt.Sprintf(
			`%s-save | awk -v c="-m comment --comment %s" '/^\*/{t=substr($1,2);next} c && index($0,c){sub(/^-A /,"",$0);system("%s -t " t " -D " $0)}'`,
			iptablesCmd, comment, iptablesCmd,
		),
	}
}

func CleanupRules(comment string, version int, targetTable *[]string, matchPrefix bool) error {
	if comment == "" {
		return fmt.Errorf("cleanFirewallRuleByComment: comment can't be empty")
	}

	logging.LogInfo("Cleaning up firewall rules with comment: %s (version: %d)", comment, version)
	if version == 46 {
		err4 := CleanupRules(comment, 4, targetTable, matchPrefix)
		err6 := CleanupRules(comment, 6, targetTable, matchPrefix)
		if err4 != nil && err6 != nil {
			return fmt.Errorf("err4:-> %v, err6:-> %v", err4, err6)
		} else if err4 != nil {
			return err4
		} else if err6 != nil {
			return err6
		}
	}
	iptablesCmd := "iptables"
	if version == 6 {
		iptablesCmd = "ip6tables"
	}
	currentRules, err := RunCommandWithOutput(fmt.Sprintf("%s-save", iptablesCmd))
	if err != nil {
		return err
	}

	var commands [][]string
	currentTable := ""
	for _, rule := range strings.Split(currentRules, "\n") {
		if len(rule) > 1 && rule[0] == '*' {
			currentTable = rule[1:]
			continue
		}
		if targetTable != nil && len(*targetTable) > 0 {
			if !stringInSlice(currentTable, *targetTable) {
				continue
			}
		}
		match := false
		if matchPrefix {
			if strings.Contains(rule, fmt.Sprintf("-m comment --comment %s", comment)) {
				match = true
			}
		} else {
			if strings.Contains(rule, fmt.Sprintf("-m comment --comment %s ", comment)) {
				match = true
			}
			if strings.HasSuffix(rule, fmt.Sprintf("-m comment --comment %s", comment)) {
				match = true
			}
		}
		if match {
			args := []string{"-t", currentTable, "-D"}
			args = append(args, strings.Fields(rule[3:])...)
			commands = append(commands, args)
		}
	}

	for _, arg := range commands {
		logging.LogInfo("Removing firewall rule: %s %s", iptablesCmd, strings.Join(arg, " "))
		_, err = RunCommandWithOutput(iptablesCmd, arg...)
	}

	if len(commands) > 0 {
		logging.LogInfo("Cleaned up %d firewall rules with comment: %s", len(commands), comment)
	} else {
		logging.LogInfo("No firewall rules found to clean up with comment: %s", comment)
	}

	return err
}

func stringInSlice(target string, slice []string) bool {
	for _, element := range slice {
		if element == target {
			return true // Found the string in the slice
		}
	}
	return false // String not found in the slice
}
