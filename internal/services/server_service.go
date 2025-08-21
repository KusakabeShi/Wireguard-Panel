package services

import (
	"fmt"

	"wg-panel/internal/config"
	"wg-panel/internal/models"
	"wg-panel/internal/utils"

	"github.com/google/uuid"
)

type ServerService struct {
	cfg *config.Config
	wg  *WireGuardService
	fw  *FirewallService
}

func NewServerService(cfg *config.Config, wgService *WireGuardService, firewallService *FirewallService) *ServerService {
	return &ServerService{
		cfg: cfg,
		wg:  wgService,
		fw:  firewallService,
	}
}

func (s *ServerService) CreateServer(interfaceID string, req ServerCreateRequest) (*models.Server, error) {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return nil, fmt.Errorf("interface not found")
	}

	// Validate server configuration
	server, err := s.validateAndGenerateServerConfig(iface, &req, nil)
	if err != nil {
		return nil, err
	}

	// Add server to interface
	iface.Servers = append(iface.Servers, server)
	s.cfg.SetInterface(interfaceID, iface)

	if err := s.cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %v", err)
	}

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
		_ = utils.CleanupRules(newserver.IPv4.CommentString, 4, false)

	}
	if newserver.IPv6 != nil {
		_ = utils.CleanupRules(newserver.IPv6.CommentString, 6, false)
	}

	s.SetServerEnabled(interfaceID, serverID, false)
	*server = *newserver
	server.Enabled = false
	if enabledState {
		s.SetServerEnabled(interfaceID, serverID, true)
	}

	// Update network configurations and determine if firewall update is needed
	s.cfg.SetInterface(interfaceID, iface)
	if err := s.cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %v", err)
	}

	return server, nil
}

func (s *ServerService) SetServerEnabled(interfaceID, serverID string, enabled bool) error {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return fmt.Errorf("interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return err
	}

	if server.Enabled == enabled {
		return nil // Already in desired state
	}

	if enabled {
		// Enable: add IP addresses, firewall rules, and sync config
		if server.IPv4 != nil && server.IPv4.Enabled {
			if err := s.fw.AddServerRules(iface.Ifname, server.IPv4); err != nil {
				return fmt.Errorf("failed to add IPv4 firewall rules: %v", err)
			}
		}
		if server.IPv6 != nil && server.IPv6.Enabled {
			if err := s.fw.AddServerRules(iface.Ifname, server.IPv6); err != nil {
				return fmt.Errorf("failed to add IPv6 firewall rules: %v", err)
			}
		}
	} else {
		// Disable: remove IP addresses, firewall rules, and sync config
		if server.IPv4 != nil && server.IPv4.Enabled {
			s.fw.RemoveServerRules(iface.Ifname, server.IPv4)
		}
		if server.IPv6 != nil && server.IPv6.Enabled {
			s.fw.RemoveServerRules(iface.Ifname, server.IPv6)
		}
	}

	server.Enabled = enabled
	s.cfg.SetInterface(interfaceID, iface)
	if err := s.cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	// Regenerate WireGuard configuration
	if err := s.wg.GenerateAndSyncInterface(iface); err != nil {
		return fmt.Errorf("failed to sync WireGuard configuration: %v", err)
	}

	return nil
}

func (s *ServerService) DeleteServer(interfaceID, serverID string) error {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return fmt.Errorf("interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return err
	}

	// Disable first (removes firewall rules and IPs)
	if server.Enabled {
		if err := s.SetServerEnabled(interfaceID, serverID, false); err != nil {
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
		if err := s.SetServerEnabled(interfaceID, serverID, false); err != nil {
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
		if err := s.SetServerEnabled(newInterfaceID, serverID, true); err != nil {
			return fmt.Errorf("failed to re-enable server after move: %v", err)
		}
	}

	// Sync both interfaces
	if err := s.wg.GenerateAndSyncInterface(srcIface); err != nil {
		return fmt.Errorf("failed to sync source interface: %v", err)
	}
	if err := s.wg.GenerateAndSyncInterface(destIface); err != nil {
		return fmt.Errorf("failed to sync destination interface: %v", err)
	}

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

	// 2. Validate IPv4 configuration if filled
	if ipv4.Network != "" {
		exids := []string{}
		if oldServer != nil {
			exids = []string{oldServer.ID}
		}
		if err := s.validateSingleServerNetworkConfig(iface, ipv4, 4, exids); err != nil {
			return nil, fmt.Errorf("IPv4 validation failed: %v", err)
		}
	} else if ipv4.Enabled {
		return nil, fmt.Errorf("ipv4 Enabled but network is nil")
	}

	// 3. Validate IPv6 configuration if filled
	if ipv6.Network != "" {
		exids := []string{}
		if oldServer != nil {
			exids = []string{oldServer.ID}
		}
		if err := s.validateSingleServerNetworkConfig(iface, ipv6, 6, exids); err != nil {
			return nil, fmt.Errorf("IPv6 validation failed: %v", err)
		}
	} else if ipv6.Enabled {
		return nil, fmt.Errorf("ipv6 Enabled but network is nil")
	}

	var server *models.Server
	prefix := s.cfg.ServerId + "-"
	ipv4CommentString, _ := utils.GenerateRandomString(prefix, 16)
	ipv6CommentString, _ := utils.GenerateRandomString(prefix, 16)

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
			newV4, err := models.ParseCIDRAf(req.IPv4.Network, 4)
			if err != nil {
				return nil, err
			}
			err = s.validateClientIPsInNewNetwork(server.Clients, newV4, 4)
			if err != nil {
				return nil, err
			}
		}
		if req.IPv6 != nil && req.IPv6.Network != "" {
			newV6, err := models.ParseCIDRAf(req.IPv6.Network, 6)
			if err != nil {
				return nil, err
			}

			err = s.validateClientIPsInNewNetwork(server.Clients, newV6, 6)
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
			oldserverbasenet := oldServer.IPv4.Network.Network()
			newserverbasenet := server.IPv4.Network.Network()
			if len(server.IPv4.RoutedNetworks) == 1 && len(oldServer.IPv4.RoutedNetworks) == 1 {
				if server.IPv4.RoutedNetworks[0].Equal(&oldserverbasenet) {
					server.IPv4.RoutedNetworks = []*models.IPNetWrapper{&newserverbasenet}
				}
			}
			if server.IPv4.Snat != nil && server.IPv4.Snat.SnatExcludedNetwork != nil {
				if server.IPv4.Snat.SnatExcludedNetwork.Equal(&oldserverbasenet) {
					server.IPv4.Snat.SnatExcludedNetwork = &newserverbasenet
				}
			}
		}
		server.IPv6 = newIPv6
		if server.IPv6 != nil && server.IPv6.Network != nil && oldServer.IPv6 != nil && oldServer.IPv6.Network != nil {
			oldserverbasenet := oldServer.IPv6.Network.Network()
			newserverbasenet := server.IPv6.Network.Network()
			if len(server.IPv6.RoutedNetworks) == 1 && len(oldServer.IPv6.RoutedNetworks) == 1 {

				if server.IPv6.RoutedNetworks[0].Equal(&oldserverbasenet) {
					server.IPv6.RoutedNetworks = []*models.IPNetWrapper{&newserverbasenet}
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

func (s *ServerService) validateSingleServerNetworkConfig(iface *models.Interface, cfg *ServerNetworkConfigRequest, version int, excludeServerID []string) error {
	if cfg.Network == "" {
		return fmt.Errorf("network must be specified")
	}

	// Parse and validate network CIDR
	var network *models.IPNetWrapper
	var err error
	if version == 4 {
		network, err = models.ParseCIDRAf(cfg.Network, 4)
	} else {
		network, err = models.ParseCIDRAf(cfg.Network, 6)
	}
	if err != nil {
		return fmt.Errorf("invalid network CIDR: %v", err)
	}

	// 2. Check for network overlaps with other servers in the same VRF
	if err := s.validateNetworkOverlaps(iface, network, version, excludeServerID...); err != nil {
		return err
	}

	// 3. Validate pseudo-bridge and SNAT mutual exclusion
	if cfg.PseudoBridgeMasterInterface != nil && *cfg.PseudoBridgeMasterInterface != "" &&
		cfg.Snat != nil && cfg.Snat.Enabled {
		return fmt.Errorf("SNAT and pseudo-bridge are mutually exclusive")
	}

	// 4. Validate routed networks don't overlap with each other
	if err := s.validateRoutedNetworksOverlap(cfg.RoutedNetworks); err != nil {
		return err
	}

	// 5. Validate SNAT configuration
	if cfg.Snat != nil && cfg.Snat.Enabled {
		if err := s.validateSnatConfiguration(network, cfg.Snat, version); err != nil {
			return fmt.Errorf("SNAT validation failed: %v", err)
		}
	}

	return nil
}

func (s *ServerService) validateClientIPsInNewNetwork(clients []*models.Client, newNetwork *models.IPNetWrapper, version int) error {
	for _, client := range clients {
		var clientIP *models.IPNetWrapper
		var err error

		if version == 4 && client.IPv4Offset != nil {
			clientIP, err = client.GetIPv4(newNetwork)
		} else if version == 6 && client.IPv6Offset != nil {
			clientIP, err = client.GetIPv6(newNetwork)
		}

		if err != nil || (clientIP != nil && !newNetwork.Contains(clientIP.IP)) {
			return fmt.Errorf("client IP %s would be out of new network range", client.Name)
		}
	}
	return nil
}

func (s *ServerService) validateNetworkOverlaps(iface *models.Interface, network *models.IPNetWrapper, version int, excludeServerID ...string) error {
	excludeMap := make(map[string]bool)
	for _, id := range excludeServerID {
		excludeMap[id] = true
	}

	for _, otherIface := range s.cfg.GetAllInterfaces() {
		if !s.interfacesInSameVRF(iface, otherIface) {
			continue
		}

		for _, server := range otherIface.Servers {
			if excludeMap[server.ID] {
				continue
			}

			var otherNetwork *models.IPNetWrapper
			if version == 4 && server.IPv4 != nil && server.IPv4.Network != nil {
				otherNetwork = server.IPv4.Network
			} else if version == 6 && server.IPv6 != nil && server.IPv6.Network != nil {
				otherNetwork = server.IPv6.Network
			}

			if otherNetwork != nil && network.IsOverlap(otherNetwork) {
				return fmt.Errorf("network overlaps with existing server network in same VRF")
			}
		}
	}
	return nil
}

func (s *ServerService) validateRoutedNetworksOverlap(routedNetworks []string) error {
	if len(routedNetworks) <= 1 {
		return nil
	}

	networks := make([]*models.IPNetWrapper, 0, len(routedNetworks))
	for _, routedNet := range routedNetworks {
		network, err := models.ParseCIDR(routedNet)
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

func (s *ServerService) validateSnatConfiguration(serverNetwork *models.IPNetWrapper, snat *SnatConfigRequest, version int) error {
	if snat.SnatIPNet != "" {
		var snatNet *models.IPNetWrapper
		var err error
		if version == 4 {
			snatNet, err = models.ParseCIDRFromIPAf(snat.SnatIPNet, 4)
		} else {
			snatNet, err = models.ParseCIDRFromIPAf(snat.SnatIPNet, 6)
		}
		if err != nil {
			return fmt.Errorf("invalid SNAT IP/Net: %v", err)
		}

		if version == 4 && snatNet.Masklen() != 32 {
			return fmt.Errorf("IPv4 SNAT IP must be /32")
		}
		if version == 6 && snatNet.Masklen() != 128 && snatNet.Masklen() != serverNetwork.Masklen() {
			return fmt.Errorf("IPv6 SNAT IP must be /128 (SNAT mode) or equal with ServerNet /%d (NETMAP mode)", serverNetwork.Masklen())
		}
	}
	return nil
}

func (s *ServerService) interfacesInSameVRF(iface1, iface2 *models.Interface) bool {
	if iface1.VRFName == nil && iface2.VRFName == nil {
		return true
	}
	if iface1.VRFName != nil && iface2.VRFName != nil {
		return *iface1.VRFName == *iface2.VRFName
	}
	return false
}

func (s *ServerService) prepareNetworkConfig(af int, req *ServerNetworkConfigRequest, commentString string) (*models.ServerNetworkConfig, error) {
	if req == nil {
		return nil, nil
	}

	config := &models.ServerNetworkConfig{
		Enabled:                     req.Enabled,
		PseudoBridgeMasterInterface: req.PseudoBridgeMasterInterface,
		RoutedNetworks:              make([]*models.IPNetWrapper, 0),
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
			config.RoutedNetworks = append(config.RoutedNetworks, &normalizedNetwork)

		}
	} else if config.Network != nil {
		// Default to server's own network, normalized
		normalizedNetwork := config.Network.Network()
		config.RoutedNetworks = []*models.IPNetWrapper{&normalizedNetwork}
	}

	// Parse SNAT configuration
	if req.Snat != nil {
		config.Snat = &models.SnatConfig{
			Enabled:                req.Snat.Enabled,
			RoamingMasterInterface: req.Snat.RoamingMasterInterface,
			RoamingPseudoBridge:    req.Snat.RoamingPseudoBridge,
		}

		if req.Snat.SnatIPNet != "" {
			snatNet, err := models.ParseCIDR(req.Snat.SnatIPNet)
			if err != nil {
				return nil, fmt.Errorf("%v is not a ipv%v network", req.Snat.SnatIPNet, af)
			}
			if snatNet.Version != af {
				return nil, fmt.Errorf("%v is not a ipv%v network", req.Snat.SnatIPNet, af)
			}
			config.Snat.SnatIPNet = snatNet

		}

		if req.Snat.SnatExcludedNetwork != "" {
			excludedNet, err := models.ParseCIDR(req.Snat.SnatExcludedNetwork)
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
