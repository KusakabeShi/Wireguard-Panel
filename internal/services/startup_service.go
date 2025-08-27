package services

import (
	"fmt"
	"log"

	"wg-panel/internal/config"
	"wg-panel/internal/internalservice"
	"wg-panel/internal/models"
	"wg-panel/internal/utils"
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
	log.Printf("Initializing WireGuard interfaces and firewall rules...")

	interfaces := s.cfg.GetAllInterfaces()
	if len(interfaces) == 0 {
		log.Printf("No interfaces found, skipping initialization")
		return nil
	}

	for _, iface := range interfaces {
		if err := s.initializeInterface(iface); err != nil {
			log.Printf("Failed to initialize interface %s: %v", iface.Ifname, err)
			// Continue with other interfaces even if one fails
			continue
		}
		log.Printf("Successfully initialized interface %s", iface.Ifname)
	}

	return nil
}

func (s *StartupService) initializeInterface(iface *models.Interface) error {
	// Check if interface has any enabled servers
	hasEnabledServers := false
	for _, server := range iface.Servers {
		if server.Enabled {
			hasEnabledServers = true
			break
		}
	}

	if !hasEnabledServers {
		log.Printf("Interface %s has no enabled servers, skipping", iface.Ifname)
		return nil
	}

	// Generate and apply WireGuard configuration
	if err := s.wg.GenerateAndSyncInterface(iface); err != nil {
		return fmt.Errorf("failed to sync WireGuard configuration: %v", err)
	}

	// Apply firewall rules for all enabled servers
	for _, server := range iface.Servers {
		if !server.Enabled {
			continue
		}

		if err := s.initializeServerFirewallRules(iface.Ifname, server); err != nil {
			log.Printf("Failed to initialize firewall rules for server %s: %v", server.Name, err)
			// Continue with other servers
			continue
		}
	}

	return nil
}

func (s *StartupService) initializeServerFirewallRules(ifname string, server *models.Server) error {
	// Apply IPv4 firewall rules
	if server.IPv4 != nil && server.IPv4.Enabled {
		if err := s.fw.AddServerRules(ifname, server.IPv4); err != nil {
			return fmt.Errorf("failed to add IPv4 firewall rules: %v", err)
		}
	}

	// Apply IPv6 firewall rules
	if server.IPv6 != nil && server.IPv6.Enabled {
		if err := s.fw.AddServerRules(ifname, server.IPv6); err != nil {
			return fmt.Errorf("failed to add IPv6 firewall rules: %v", err)
		}
	}

	return nil
}

// CleanupOrphanedRules removes any orphaned firewall rules from previous sessions
func (s *StartupService) CleanupOrphanedRules() error {
	log.Printf("Cleaning up orphaned firewall rules...")

	// Get all current comment strings from active servers
	_ = utils.CleanupRules(s.cfg.ServerId, 4, true)
	_ = utils.CleanupRules(s.cfg.ServerId, 6, true)
	return nil
}
