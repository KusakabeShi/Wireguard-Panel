package services

import (
	"fmt"

	"wg-panel/internal/config"
	"wg-panel/internal/internalservice"
	"wg-panel/internal/logging"
	"wg-panel/internal/models"
)

type StartupService struct {
	cfg *config.Config
	wg  *WireGuardService
	fw  *internalservice.FirewallService
}

func NewStartupService(cfg *config.Config, wgService *WireGuardService, firewallService *internalservice.FirewallService) *StartupService {
	return &StartupService{
		cfg: cfg,
		wg:  wgService,
		fw:  firewallService,
	}
}

// InitializeInterfaces brings up all enabled interfaces and applies firewall rules during startup
func (s *StartupService) InitializeInterfaces() error {
	logging.LogInfo("Initializing WireGuard interfaces and firewall rules...")

	interfaces := s.cfg.GetAllInterfaces()
	if len(interfaces) == 0 {
		logging.LogInfo("No interfaces found, skipping initialization")
		return nil
	}

	for _, iface := range interfaces {
		if err := s.initializeInterface(iface); err != nil {
			logging.LogError("Failed to initialize interface %s: %v", iface.Ifname, err)
			// Continue with other interfaces even if one fails
			iface.Enabled = false
			continue
		}
		logging.LogInfo("Successfully initialized interface %s", iface.Ifname)
	}
	s.cfg.SyncToInternalService()

	return nil
}

func (s *StartupService) initializeInterface(iface *models.Interface) error {
	// Check if interface has any enabled servers
	if !iface.Enabled {
		return nil
	}
	hasEnabledServers := false
	for _, server := range iface.Servers {
		if server.Enabled {
			hasEnabledServers = true
			break
		}
	}

	if !hasEnabledServers {
		logging.LogVerbose("Interface %s has no enabled servers, skipping", iface.Ifname)
		return nil
	}

	// Generate and apply WireGuard configuration
	if err := s.wg.SyncToConfAndInterface(iface); err != nil {
		return fmt.Errorf("failed to sync WireGuard configuration:-> %v", err)
	}

	// Apply firewall rules for all enabled servers
	for _, server := range iface.Servers {
		if !server.Enabled {
			continue
		}

		if err := s.initializeServerFirewallRules(iface.Ifname, iface.VRFName, server); err != nil {
			logging.LogError("Failed to initialize firewall rules for server %s: %v", server.Name, err)
			// Continue with other servers
			continue
		}
	}

	return nil
}

func (s *StartupService) initializeServerFirewallRules(ifname string, vrf *string, server *models.Server) error {
	// Apply IPv4 firewall rules
	if server.IPv4 != nil && server.IPv4.Enabled {
		if err := s.fw.AddIpAndFwRules(ifname, vrf, server.IPv4); err != nil {
			return fmt.Errorf("failed to add IPv4 firewall rules:-> %v", err)
		}
	}

	// Apply IPv6 firewall rules
	if server.IPv6 != nil && server.IPv6.Enabled {
		if err := s.fw.AddIpAndFwRules(ifname, vrf, server.IPv6); err != nil {
			return fmt.Errorf("failed to add IPv6 firewall rules:-> %v", err)
		}
	}

	return nil
}
