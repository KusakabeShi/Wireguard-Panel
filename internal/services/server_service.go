package services

import (
	"fmt"

	"wg-panel/internal/config"
	"wg-panel/internal/internalservice"
	"wg-panel/internal/logging"
	"wg-panel/internal/models"
	"wg-panel/internal/utils"

	"github.com/google/uuid"
)

type ServerService struct {
	cfg *config.Config
	wg  *WireGuardService
	fw  *internalservice.FirewallService
}

func NewServerService(cfg *config.Config, wgService *WireGuardService, firewallService *internalservice.FirewallService) *ServerService {
	return &ServerService{
		cfg: cfg,
		wg:  wgService,
		fw:  firewallService,
	}
}

func (s *ServerService) CreateServer(interfaceID string, req ServerCreateRequest) (*models.Server, error) {
	logging.LogInfo("Creating server %s for interface %s", req.Name, interfaceID)
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		logging.LogError("Interface %s not found during server creation", interfaceID)
		return nil, fmt.Errorf("interface not found")
	}

	// Validate server configuration
	logging.LogVerbose("Validating server configuration for %s", req.Name)
	server, err := s.validateAndGenerateServerConfig(iface, &req, nil)
	if err != nil {
		logging.LogError("Server validation failed for %s: %v", req.Name, err)
		return nil, err
	}

	// Add server to interface
	iface.Servers = append(iface.Servers, server)
	s.cfg.SetInterface(interfaceID, iface)

	logging.LogVerbose("Saving configuration after server creation")
	if err := s.cfg.Save(); err != nil {
		logging.LogError("Failed to save configuration after creating server %s: %v", req.Name, err)
		return nil, fmt.Errorf("failed to save configuration: %v", err)
	}

	logging.LogInfo("Successfully created server %s (ID: %s) for interface %s", server.Name, server.ID, interfaceID)
	return server, nil
}

func (s *ServerService) GetServer(interfaceID, serverID string) (*models.Server, error) {
	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return nil, err
	}
	return server, nil
}

func (s *ServerService) GetServers(interfaceID string) ([]*models.Server, error) {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return nil, fmt.Errorf("interface not found")
	}
	return iface.Servers, nil
}

func (s *ServerService) UpdateServer(interfaceID, serverID string, req ServerCreateRequest) (*models.Server, error) {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return nil, fmt.Errorf("interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return nil, err
	}

	enabledState := server.Enabled

	// Validate server configuration
	newserver, err := s.validateAndGenerateServerConfig(iface, &req, server)
	if err != nil {
		return nil, err
	}
	if newserver.IPv4 != nil {
		_ = utils.CleanupRules(newserver.IPv4.CommentString, 4, nil, false)

	}
	if newserver.IPv6 != nil {
		_ = utils.CleanupRules(newserver.IPv6.CommentString, 6, nil, false)
	}

	s.SetServerEnabled(interfaceID, serverID, false, !enabledState)
	*server = *newserver
	server.Enabled = false
	if enabledState {
		s.SetServerEnabled(interfaceID, serverID, true, true)
	}

	// Update network configurations and determine if firewall update is needed
	s.cfg.SetInterface(interfaceID, iface)
	if err := s.cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %v", err)
	}

	return server, nil
}

func (s *ServerService) SetServerEnabled(interfaceID, serverID string, enabled bool, syncServiceAndConfig bool) error {
	logging.LogInfo("Setting server %s enabled=%t for interface %s", serverID, enabled, interfaceID)
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		logging.LogError("Interface %s not found when setting server enabled state", interfaceID)
		return fmt.Errorf("interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		logging.LogError("Server %s not found when setting enabled state: %v", serverID, err)
		return err
	}

	if server.Enabled == enabled {
		logging.LogVerbose("Server %s already in desired enabled state (%t)", serverID, enabled)
		return nil // Already in desired state
	}

	if enabled {
		// Enable: add IP addresses, firewall rules, and sync config
		logging.LogInfo("Enabling server %s - adding firewall rules", serverID)
		if iface.Enabled && server.IPv4 != nil && server.IPv4.Enabled {
			if err := s.fw.AddIpAndFwRules(iface.Ifname, server.IPv4); err != nil {
				logging.LogError("Failed to add IPv4 firewall rules for server %s: %v", serverID, err)
				return fmt.Errorf("failed to add IPv4 firewall rules: %v", err)
			}
		}
		if iface.Enabled && server.IPv6 != nil && server.IPv6.Enabled {
			if err := s.fw.AddIpAndFwRules(iface.Ifname, server.IPv6); err != nil {
				logging.LogError("Failed to add IPv6 firewall rules for server %s: %v", serverID, err)
				return fmt.Errorf("failed to add IPv6 firewall rules: %v", err)
			}
		}
	} else {
		// Disable: remove IP addresses, firewall rules, and sync config
		logging.LogInfo("Disabling server %s - removing firewall rules", serverID)
		if iface.Enabled && server.IPv4 != nil && server.IPv4.Enabled {
			s.fw.RemoveIpAndFwRules(iface.Ifname, server.IPv4)
		}
		if iface.Enabled && server.IPv6 != nil && server.IPv6.Enabled {
			s.fw.RemoveIpAndFwRules(iface.Ifname, server.IPv6)
		}
	}
	server.Enabled = enabled
	if syncServiceAndConfig {
		// Regenerate WireGuard configuration
		logging.LogVerbose("Syncing WireGuard configuration after server enable/disable")
		if err := s.wg.SyncToConfAndInterface(iface); err != nil {
			logging.LogError("Failed to sync WireGuard configuration for server %s: %v", serverID, err)
			return fmt.Errorf("failed to sync WireGuard configuration: %v", err)
		}

		s.cfg.SyncToInternalService()
	}
	s.cfg.SetInterface(interfaceID, iface)
	if syncServiceAndConfig {
		if err := s.cfg.Save(); err != nil {
			logging.LogError("Failed to save configuration after setting server enabled: %v", err)
			return fmt.Errorf("failed to save configuration: %v", err)
		}
	}
	logging.LogInfo("Successfully set server %s enabled=%t for interface %s", serverID, enabled, interfaceID)
	return nil
}

func (s *ServerService) DeleteServer(interfaceID, serverID string) error {
	logging.LogInfo("Deleting server %s from interface %s", serverID, interfaceID)
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		logging.LogError("Interface %s not found during server deletion", interfaceID)
		return fmt.Errorf("interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		logging.LogError("Server %s not found during deletion: %v", serverID, err)
		return err
	}

	// Disable first (removes firewall rules and IPs)
	if server.Enabled {
		logging.LogVerbose("Disabling server %s before deletion", serverID)
		if err := s.SetServerEnabled(interfaceID, serverID, false, true); err != nil {
			logging.LogError("Failed to disable server %s before deletion: %v", serverID, err)
			return fmt.Errorf("failed to disable server before deletion: %v", err)
		}
	}

	// Remove server from interface
	for i, srv := range iface.Servers {
		if srv.ID == serverID {
			iface.Servers = append(iface.Servers[:i], iface.Servers[i+1:]...)
			break
		}
	}

	s.cfg.SetInterface(interfaceID, iface)
	return s.cfg.Save()
}

func (s *ServerService) MoveServer(interfaceID, serverID, newInterfaceID string) error {
	// Get source interface and server
	srcIface := s.cfg.GetInterface(interfaceID)
	if srcIface == nil {
		return fmt.Errorf("source interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return err
	}

	// Get destination interface
	destIface := s.cfg.GetInterface(newInterfaceID)
	if destIface == nil {
		return fmt.Errorf("destination interface not found")
	}

	// Disable server first
	wasEnabled := server.Enabled
	if wasEnabled {
		if err := s.SetServerEnabled(interfaceID, serverID, false, true); err != nil {
			return fmt.Errorf("failed to disable server for move: %v", err)
		}
	}

	// Remove from source interface
	for i, srv := range srcIface.Servers {
		if srv.ID == serverID {
			srcIface.Servers = append(srcIface.Servers[:i], srcIface.Servers[i+1:]...)
			break
		}
	}

	// Add to destination interface
	destIface.Servers = append(destIface.Servers, server)

	// Save both interfaces
	s.cfg.SetInterface(interfaceID, srcIface)
	s.cfg.SetInterface(newInterfaceID, destIface)
	if err := s.cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	// Re-enable on destination if it was enabled
	if wasEnabled {
		if err := s.SetServerEnabled(newInterfaceID, serverID, true, true); err != nil {
			return fmt.Errorf("failed to re-enable server after move: %v", err)
		}
	}

	// Sync both interfaces
	if err := s.wg.SyncToConfAndInterface(srcIface); err != nil {
		return fmt.Errorf("failed to sync source interface: %v", err)
	}
	if err := s.wg.SyncToConfAndInterface(destIface); err != nil {
		return fmt.Errorf("failed to sync destination interface: %v", err)
	}
	s.cfg.SyncToInternalService()

	return nil
}

func (s *ServerService) validateAndGenerateServerConfig(iface *models.Interface, req *ServerCreateRequest, oldServer *models.Server) (*models.Server, error) {
	// 1. Validate that at least one of IPv4 or IPv6 is enabled
	ipv4 := req.IPv4
	ipv6 := req.IPv6
	if !(ipv4 != nil && ipv4.Enabled) && !(ipv6 != nil && ipv6.Enabled) {
		return nil, fmt.Errorf("at least one of IPv4 or IPv6 must be enabled")
	}

	routedNetworkPool := []string{}
	if ipv4 != nil {
		routedNetworkPool = append(routedNetworkPool, ipv4.RoutedNetworks...)
		ipv4.RoutedNetworks = []string{}
	}
	if ipv6 != nil {
		routedNetworkPool = append(routedNetworkPool, ipv6.RoutedNetworks...)
		ipv6.RoutedNetworks = []string{}
	}
	for _, r := range routedNetworkPool {
		net, err := models.ParseCIDRFromIP(r)
		if err != nil {
			continue
		}
		if net.Version == 4 && ipv4 != nil {
			ipv4.RoutedNetworks = append(ipv4.RoutedNetworks, r)
		}
		if net.Version == 6 && ipv6 != nil {
			ipv6.RoutedNetworks = append(ipv6.RoutedNetworks, r)
		}
	}
	var exid *string
	if oldServer != nil {
		exid = &oldServer.ID
	}
	// 2. Validate IPv4 configuration if filled
	if ipv4.Network != "" {
		if err := s.validateSingleServerNetworkConfig(4, iface, ipv4, exid); err != nil {
			return nil, fmt.Errorf("IPv4 validation failed: %v", err)
		}
	} else if ipv4.Enabled {
		return nil, fmt.Errorf("ipv4 Enabled but network is nil")
	}

	// 3. Validate IPv6 configuration if filled
	if ipv6.Network != "" {
		if err := s.validateSingleServerNetworkConfig(6, iface, ipv6, exid); err != nil {
			return nil, fmt.Errorf("IPv6 validation failed: %v", err)
		}
	} else if ipv6.Enabled {
		return nil, fmt.Errorf("ipv6 Enabled but network is nil")
	}

	var server *models.Server
	prefix := s.cfg.ServerId + "-"
	CommentString, _ := utils.GenerateRandomString(prefix, 12)
	ipv4CommentString := prefix + "-v4-" + CommentString
	ipv6CommentString := prefix + "-v6-" + CommentString

	if oldServer == nil {
		// Generate comment strings for firewall rules with server ID prefix
		server = &models.Server{
			ID:      uuid.New().String(),
			Name:    req.Name,
			Enabled: false, // Always start disabled
			DNS:     req.DNS,

			Clients: []*models.Client{},
		}
		newIPv4, err := s.prepareNetworkConfig(4, req.IPv4, ipv4CommentString)
		if err != nil {
			return nil, err
		}
		newIPv6, err := s.prepareNetworkConfig(6, req.IPv6, ipv6CommentString)
		if err != nil {
			return nil, err
		}
		server.IPv4 = newIPv4
		server.IPv6 = newIPv6
	} else {
		server = &models.Server{}
		*server = *oldServer

		if req.IPv4 != nil && req.IPv4.Network != "" {
			newV4, err := models.ParseCIDRAf(4, req.IPv4.Network)
			if err != nil {
				return nil, err
			}
			err = s.validateClientIPsInNewNetwork(4, server.Clients, newV4)
			if err != nil {
				return nil, err
			}
		}
		if req.IPv6 != nil && req.IPv6.Network != "" {
			newV6, err := models.ParseCIDRAf(6, req.IPv6.Network)
			if err != nil {
				return nil, err
			}

			err = s.validateClientIPsInNewNetwork(6, server.Clients, newV6)
			if err != nil {
				return nil, err
			}
		}

		server.Name = req.Name
		server.DNS = req.DNS

		newIPv4, err := s.prepareNetworkConfig(4, req.IPv4, utils.If(server.IPv4 == nil, ipv4CommentString, server.IPv4.CommentString))
		if err != nil {
			return nil, err
		}
		newIPv6, err := s.prepareNetworkConfig(6, req.IPv6, utils.If(server.IPv6 == nil, ipv6CommentString, server.IPv6.CommentString))
		if err != nil {
			return nil, err
		}

		server.IPv4 = newIPv4
		if server.IPv4 != nil && server.IPv4.Network != nil && oldServer.IPv4 != nil && oldServer.IPv4.Network != nil {
			oldServerBaseNet := oldServer.IPv4.Network.Network()
			newServerBaseNet := server.IPv4.Network.Network()
			if !oldServerBaseNet.Equal(&newServerBaseNet) {
				// if oldServerBaseNet exists in server.IPv4.RoutedNetworks, and newserverbasenet not exisis, replace it with newServerBaseNet
				newnetexisisinrns := false
				for _, rn := range server.IPv4.RoutedNetworks {
					if rn.Contains(oldServerBaseNet.IP) || oldServerBaseNet.Contains(rn.IP) {
						newnetexisisinrns = true
						break
					}
				}
				if !newnetexisisinrns {
					// replace oldServerBaseNet with newServerBaseNet in server.IPv4.RoutedNetworks
					for i, rn := range server.IPv4.RoutedNetworks {
						if oldServerBaseNet.Equal(&rn) {
							server.IPv4.RoutedNetworks[i] = newServerBaseNet
						}
					}
				}
			}
			if server.IPv4.Snat != nil && server.IPv4.Snat.SnatExcludedNetwork != nil {
				if server.IPv4.Snat.SnatExcludedNetwork.Equal(&oldServerBaseNet) {
					server.IPv4.Snat.SnatExcludedNetwork = &newServerBaseNet
				}
			}
		}
		server.IPv6 = newIPv6
		if server.IPv6 != nil && server.IPv6.Network != nil && oldServer.IPv6 != nil && oldServer.IPv6.Network != nil {
			oldserverbasenet := oldServer.IPv6.Network.Network()
			newserverbasenet := server.IPv6.Network.Network()
			if !oldserverbasenet.Equal(&newserverbasenet) {
				// if oldserverbasenet exists in server.IPv6.RoutedNetworks, and newserverbasenet not exisis, replace it with newserverbasenet
				newnetexisisinrns := false
				for _, rn := range server.IPv6.RoutedNetworks {
					if rn.Contains(oldserverbasenet.IP) || oldserverbasenet.Contains(rn.IP) {
						newnetexisisinrns = true
						break
					}
				}
				if !newnetexisisinrns {
					for i, rn := range server.IPv6.RoutedNetworks {
						if oldserverbasenet.Equal(&rn) {
							server.IPv6.RoutedNetworks[i] = newserverbasenet
						}
					}
				}
			}
			if server.IPv6.Snat != nil && server.IPv6.Snat.SnatExcludedNetwork != nil {
				if server.IPv6.Snat.SnatExcludedNetwork.Equal(&oldserverbasenet) {
					server.IPv6.Snat.SnatExcludedNetwork = &newserverbasenet
				}
			}
		}
	}

	return server, nil
}

func (s *ServerService) validateSingleServerNetworkConfig(af int, iface *models.Interface, cfg *ServerNetworkConfigRequest, excludeServerID *string) error {
	if cfg.Network == "" {
		return fmt.Errorf("network must be specified")
	}

	// Parse and validate network CIDR
	var network *models.IPNetWrapper
	var err error
	if af == 4 {
		network, err = models.ParseCIDRAf(4, cfg.Network)
	} else {
		network, err = models.ParseCIDRAf(6, cfg.Network)
	}
	if err != nil {
		return fmt.Errorf("invalid network CIDR: %v", err)
	}

	// 2. Check for network overlaps with other servers in the same VRF

	if err := s.cfg.CheckNetworkOverlapsInVRF(iface.VRFName, nil, excludeServerID, network); err != nil {
		return err
	}

	// 4. Validate routed networks don't overlap with each other
	if err := s.validateRoutedNetworksOverlap(af, cfg.RoutedNetworks); err != nil {
		return err
	}

	if cfg.PseudoBridgeMasterInterface != nil && len(*cfg.PseudoBridgeMasterInterface) > 0 {
		err := utils.IsIfaceLayer2(*cfg.PseudoBridgeMasterInterface)
		if err != nil {
			return err
		}
	}

	// 5. Validate SNAT configuration
	if cfg.Snat != nil && cfg.Snat.Enabled {
		if err := s.validateSnatConfiguration(af, network, cfg.Snat); err != nil {
			return fmt.Errorf("SNAT validation failed: %v", err)
		}
	}

	return nil
}

func (s *ServerService) validateClientIPsInNewNetwork(af int, clients []*models.Client, newNetwork *models.IPNetWrapper) error {
	for _, client := range clients {
		var clientIP *models.IPNetWrapper
		var err error

		if af == 4 && client.IPv4Offset != nil {
			clientIP, err = client.GetIPv4(newNetwork)
		} else if af == 6 && client.IPv6Offset != nil {
			clientIP, err = client.GetIPv6(newNetwork)
		}

		if err != nil || (clientIP != nil && !newNetwork.Contains(clientIP.IP)) {
			return fmt.Errorf("client IP %s would be out of new network range", client.Name)
		}
	}
	return nil
}

func (s *ServerService) validateRoutedNetworksOverlap(af int, routedNetworks []string) error {
	if len(routedNetworks) <= 1 {
		return nil
	}

	networks := make([]*models.IPNetWrapper, 0, len(routedNetworks))
	for _, routedNet := range routedNetworks {
		network, err := models.ParseCIDRAf(af, routedNet)
		if err != nil {
			return fmt.Errorf("invalid routed network CIDR %s: %v", routedNet, err)
		}
		baseNetwork := network.Network()
		networks = append(networks, &baseNetwork)
	}

	for i := 0; i < len(networks); i++ {
		for j := i + 1; j < len(networks); j++ {
			if networks[i].IsOverlap(networks[j]) {
				return fmt.Errorf("routed networks %s and %s overlap", networks[i].String(), networks[j].String())
			}
		}
	}
	return nil
}

func (s *ServerService) validateSnatConfiguration(af int, serverNetwork *models.IPNetWrapper, snat *SnatConfigRequest) error {
	isRoaming := false
	snatmode := ""
	if snat.RoamingMasterInterface != nil && len(*snat.RoamingMasterInterface) > 0 {
		isRoaming = true
	}
	if snat.SnatIPNet == "" {
		// MASQUERADE mode
		snatmode = "MASQUERADE"
		if isRoaming {
			return fmt.Errorf("masquerade mode doesn't support roaming, SnatIPNet must be set, or unset RoamingMasterInterface")
		}
	} else {
		var snatNet *models.IPNetWrapper
		var err error
		switch af {
		case 4:
			snatNet, err = models.ParseCIDRFromIPAf(4, snat.SnatIPNet)
		case 6:
			snatNet, err = models.ParseCIDRFromIPAf(6, snat.SnatIPNet)
		default:
			return fmt.Errorf("invalid IP version for SNAT configuration")
		}
		if err != nil {
			return fmt.Errorf("invalid SNAT IP/Net: %v", err)
		}

		switch af {
		case 4:
			if snatNet.Masklen() == 32 {
				snatmode = "SNAT"
				if isRoaming && !snatNet.EqualZero(af) {
					return fmt.Errorf("in roaming mode, SNAT IP must be 0.0.0.0/32")
				}
			} else {
				return fmt.Errorf("IPv4 SNAT doesn't support NETMAP mode, it supports SNAT mode only. Thus IPNet must be /32")
			}
		case 6:
			if snatNet.Masklen() == 128 {
				snatmode = "SNAT"
				if isRoaming && !snatNet.EqualZero(af) {
					return fmt.Errorf("in roaming mode, SNAT IP must be ::/128")
				}
			} else {
				snatmode = "NETMAP"
				// NETMAP mode, must match server network masklen
				if snatNet.Masklen() != serverNetwork.Masklen() {
					return fmt.Errorf("IPv6 SNAT IP must be /128 (SNAT mode) or equal with ServerNet /%d (NETMAP mode)", serverNetwork.Masklen())
				}
			}
		}
	}
	if isRoaming {
		err := utils.IsIfaceLayer2(*snat.RoamingMasterInterface)
		if err != nil {
			return err
		}
	}
	if snat.RoamingPseudoBridge && !isRoaming {
		return fmt.Errorf("RoamingPseudoBridge can be true only if RoamingMasterInterface is set")
	}
	if snat.RoamingPseudoBridge && (snatmode != "NETMAP") {
		return fmt.Errorf("RoamingPseudoBridge can only works in NETMAP mode instead of %s mode", snatmode)
	}

	return nil
}

func (s *ServerService) prepareNetworkConfig(af int, req *ServerNetworkConfigRequest, commentString string) (*models.ServerNetworkConfig, error) {
	if req == nil {
		return nil, nil
	}

	config := &models.ServerNetworkConfig{
		Enabled:                     req.Enabled,
		PseudoBridgeMasterInterface: req.PseudoBridgeMasterInterface,
		RoutedNetworks:              make([]models.IPNetWrapper, 0),
		RoutedNetworksFirewall:      req.RoutedNetworksFirewall,
		CommentString:               commentString,
	}

	// Parse network
	if req.Network != "" {
		network, err := models.ParseCIDR(req.Network)
		if err != nil {
			return nil, fmt.Errorf("%v is not a ipv%v network", req.Network, af)
		}
		if network.Version != af {
			return nil, fmt.Errorf("%v is not a ipv%v address", req.Network, af)
		}
		config.Network = network

	}

	// Parse routed networks and normalize using .Network()
	if len(req.RoutedNetworks) > 0 {
		for _, routedNet := range req.RoutedNetworks {
			network, err := models.ParseCIDR(routedNet)
			if err != nil {
				return nil, fmt.Errorf("%v is not a ipv%v network in %v", routedNet, af, req.RoutedNetworks)
			}
			if network.Version != af {
				return nil, fmt.Errorf("%v is not a ipv%v address in %v", routedNet, af, req.RoutedNetworks)
			}
			normalizedNetwork := network.Network()
			config.RoutedNetworks = append(config.RoutedNetworks, normalizedNetwork)

		}
	} else if config.Network != nil {
		// Default to server's own network, normalized
		normalizedNetwork := config.Network.Network()
		config.RoutedNetworks = []models.IPNetWrapper{normalizedNetwork}
	}

	// Parse SNAT configuration
	if req.Snat != nil {
		config.Snat = &models.SnatConfig{
			Enabled:                req.Snat.Enabled,
			RoamingMasterInterface: req.Snat.RoamingMasterInterface,
			RoamingPseudoBridge:    req.Snat.RoamingPseudoBridge,
		}

		if req.Snat.SnatIPNet != "" {
			snatNet, err := models.ParseCIDRFromIPAf(af, req.Snat.SnatIPNet)
			if err != nil {
				return nil, fmt.Errorf("%v is not a ipv%v network", req.Snat.SnatIPNet, af)
			}
			if snatNet.Version != af {
				return nil, fmt.Errorf("%v is not a ipv%v network", req.Snat.SnatIPNet, af)
			}
			config.Snat.SnatIPNet = snatNet

		}

		if req.Snat.SnatExcludedNetwork != "" {
			excludedNet, err := models.ParseCIDRAf(af, req.Snat.SnatExcludedNetwork)
			if err != nil {
				return nil, fmt.Errorf("%v is not a ipv%v network", req.Snat.SnatExcludedNetwork, af)
			}
			if excludedNet.Version != af {
				return nil, fmt.Errorf("%v is not a ipv%v network", req.Snat.SnatExcludedNetwork, af)
			}
			normalizedNetwork := excludedNet.Network()
			config.Snat.SnatExcludedNetwork = &normalizedNetwork
		} else if config.Network != nil {
			// Default to server's own network, normalized
			normalizedNetwork := config.Network.Network()
			config.Snat.SnatExcludedNetwork = &normalizedNetwork
		}
	}

	return config, nil
}

// Request types
type ServerCreateRequest struct {
	Name string                      `json:"name" binding:"required"`
	DNS  []string                    `json:"dns"`
	IPv4 *ServerNetworkConfigRequest `json:"ipv4"`
	IPv6 *ServerNetworkConfigRequest `json:"ipv6"`
}

type ServerNetworkConfigRequest struct {
	Enabled                     bool               `json:"enabled"`
	Network                     string             `json:"network"`
	PseudoBridgeMasterInterface *string            `json:"pseudoBridgeMasterInterface"`
	Snat                        *SnatConfigRequest `json:"snat"`
	RoutedNetworks              []string           `json:"routedNetworks"`
	RoutedNetworksFirewall      bool               `json:"routedNetworksFirewall"`
}

type SnatConfigRequest struct {
	Enabled                bool    `json:"enabled"`
	SnatIPNet              string  `json:"snatIpNet"`
	SnatExcludedNetwork    string  `json:"snatExcludedNetwork"`
	RoamingMasterInterface *string `json:"roamingMasterInterface"`
	RoamingPseudoBridge    bool    `json:"roamingPseudoBridge"`
}
