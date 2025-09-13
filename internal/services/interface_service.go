package services

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"wg-panel/internal/config"
	"wg-panel/internal/logging"
	"wg-panel/internal/models"
	"wg-panel/internal/utils"
)

type InterfaceService struct {
	cfg *config.Config
	wg  *WireGuardService
}

func NewInterfaceService(cfg *config.Config, wgService *WireGuardService) *InterfaceService {
	return &InterfaceService{
		cfg: cfg,
		wg:  wgService,
	}
}

func (s *InterfaceService) CreateInterface(req InterfaceCreateRequest) (*models.Interface, error) {
	// Validate ifname
	if err := utils.IsValidIfname(s.cfg.WgIfPrefix, req.Ifname); err != nil {
		return nil, err
	}

	// Check if ifname already exists in configuration
	for _, iface := range s.cfg.GetAllInterfaces() {
		if iface.Ifname == req.Ifname {
			return nil, fmt.Errorf("interface with ifname '%s' already exists", req.Ifname)
		}
	}

	// Check if ifname is available in OS and filesystem
	if err := s.CheckIfNameAvailable(req.Ifname); err != nil {
		return nil, err
	}

	// Generate private key if not provided
	privateKey := req.PrivateKey
	if privateKey == "" {
		var err error
		privateKey, err = utils.GenerateWGPrivateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key:-> %v", err)
		}
	}

	// Generate public key
	publicKey, err := utils.PrivToPublic(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key:-> %v", err)
	}

	// Validate VRF if specified
	if req.VRFName != nil && *req.VRFName != "" {
		if err := utils.CheckVRFExists(*req.VRFName); err != nil {
			return nil, err
		}
	}

	// Validate endpoint
	if newendpoint, err := s.ValidateEndpoint(req.Endpoint); err != nil {
		return nil, err
	} else {
		req.Endpoint = newendpoint
	}
	// Check if UDP port is available
	if err := s.CheckUDPPortAvailable(req.Port); err != nil {
		return nil, err
	}

	// Set defaults
	if req.MTU <= 0 {
		req.MTU = 1420
	}

	iface := &models.Interface{
		ID:         s.cfg.GetAvailableInterfaceID(),
		Ifname:     req.Ifname,
		Enabled:    false,
		VRFName:    req.VRFName,
		FwMark:     req.FwMark,
		Endpoint:   req.Endpoint,
		Port:       req.Port,
		MTU:        req.MTU,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Servers:    []*models.Server{},
	}

	s.cfg.SetInterface(iface.ID, iface)
	if err := s.cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save configuration:-> %v", err)
	}

	// Generate and apply WireGuard configuration
	if err := s.wg.SyncToConf(iface); err != nil {
		return nil, fmt.Errorf("failed to save WireGuard configuration:-> %v", err)
	}

	return s.sanitizeInterface(iface), nil
}

func (s *InterfaceService) SetInterfaceEnabled(id string, enabled bool) error {
	logging.LogInfo("Setting interface %s enabled=%t", id, enabled)
	iface := s.cfg.GetInterface(id)
	if iface == nil {
		return fmt.Errorf("interface not found")
	}

	// Regenerate WireGuard configuration
	if err := s.wg.SyncToConf(iface); err != nil {
		return fmt.Errorf("failed to sync WireGuard configuration:-> %v", err)
	}
	if err := s.wg.SyncToInterface(iface.Ifname, enabled, iface.PrivateKey); err != nil {
		return fmt.Errorf("failed to apply WireGuard configuration:-> %v", err)
	}

	iface.Enabled = enabled

	s.cfg.SetInterface(iface.ID, iface)
	if err := s.cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration:-> %v", err)
	}

	return nil
}

func (s *InterfaceService) GetInterface(id string) (*models.Interface, error) {
	iface := s.cfg.GetInterface(id)
	if iface == nil {
		return nil, fmt.Errorf("interface not found")
	}
	return s.sanitizeInterface(iface), nil
}

func (s *InterfaceService) GetAllInterfaces() []*models.Interface {
	interfaces := make([]*models.Interface, 0)
	for _, iface := range s.cfg.GetAllInterfaces() {
		interfaces = append(interfaces, s.sanitizeInterface(iface))
	}
	return interfaces
}

func (s *InterfaceService) UpdateInterface(id string, req InterfaceUpdateRequest) (*models.Interface, error) {
	iface := s.cfg.GetInterface(id)
	if iface == nil {
		return nil, fmt.Errorf("interface not found")
	}
	needsWGReCreateOldName := ""
	needsWGRegeneration := false
	needsMTUUpdate := false
	needsVRFUpdate := false

	// Update fields
	if req.Ifname != "" && req.Ifname != iface.Ifname {
		if err := utils.IsValidIfname(s.cfg.WgIfPrefix, req.Ifname); err != nil {
			return nil, err
		}
		// Check if new ifname already exists in configuration
		for _, otherIface := range s.cfg.GetAllInterfaces() {
			if otherIface.ID != id && otherIface.Ifname == req.Ifname {
				return nil, fmt.Errorf("interface with ifname '%s' already exists", req.Ifname)
			}
		}
		// Check if ifname is available in OS and filesystem
		if err := s.CheckIfNameAvailable(req.Ifname); err != nil {
			return nil, err
		}
		needsWGReCreateOldName = iface.Ifname
		iface.Ifname = req.Ifname
		needsWGRegeneration = true
	}

	if req.VRFName != nil && *req.VRFName != *iface.VRFName {
		// Validate VRF if specified
		if *req.VRFName != "" {
			if err := utils.CheckVRFExists(*req.VRFName); err != nil {
				return nil, err
			}
		}

		// Check for network overlaps when changing VRF
		for _, server := range iface.Servers {
			if err := s.cfg.CheckNetworkOverlapsInVRF(req.VRFName, nil, nil, server.GetNetwork(4)); err != nil {
				return nil, err
			}
			if err := s.cfg.CheckNetworkOverlapsInVRF(req.VRFName, nil, nil, server.GetNetwork(6)); err != nil {
				return nil, err
			}
		}

		iface.VRFName = req.VRFName
		needsWGRegeneration = true
		needsVRFUpdate = true
	}

	if req.FwMark != nil && (iface.FwMark == nil || *req.FwMark != *iface.FwMark) {
		iface.FwMark = req.FwMark
		needsWGRegeneration = true
	}

	if req.Endpoint != "" && req.Endpoint != iface.Endpoint {
		// Validate endpoint
		if newendpoint, err := s.ValidateEndpoint(req.Endpoint); err != nil {
			return nil, err
		} else {
			req.Endpoint = newendpoint
		}
		iface.Endpoint = req.Endpoint
	}

	if req.Port > 0 && req.Port != iface.Port {
		// Check if UDP port is available
		if err := s.CheckUDPPortAvailable(req.Port); err != nil {
			return nil, err
		}
		iface.Port = req.Port
		needsWGRegeneration = true
	}

	if req.MTU > 0 && req.MTU != iface.MTU {
		iface.MTU = req.MTU
		needsWGRegeneration = true
		needsMTUUpdate = true
	}

	if req.PrivateKey != "" && req.PrivateKey != iface.PrivateKey {
		publicKey, err := utils.PrivToPublic(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to generate public key:-> %v", err)
		}
		iface.PrivateKey = req.PrivateKey
		iface.PublicKey = publicKey
		needsWGRegeneration = true
	}

	if needsWGReCreateOldName == "" {
		// Apply system changes
		if needsWGRegeneration {
			if err := s.wg.SyncToConf(iface); err != nil {
				return nil, fmt.Errorf("failed to regenerate WireGuard configuration:-> %v", err)
			}
		}

		if needsMTUUpdate {
			if err := s.wg.SetInterfaceMTU(iface.Ifname, iface.MTU); err != nil {
				return nil, fmt.Errorf("failed to update MTU:-> %v", err)
			}
		}
	} else {
		// Remove old interface
		if err := s.wg.SyncToInterface(needsWGReCreateOldName, false, iface.PrivateKey); err != nil {
			return nil, fmt.Errorf("failed to bring down old WireGuard interface:-> %v", err)
		}
		if err := s.wg.RemoveConfig(needsWGReCreateOldName); err != nil {
			return nil, fmt.Errorf("failed to remove old WireGuard interface:-> %v", err)
		}
		// Create new interface
		if err := s.wg.SyncToConf(iface); err != nil {
			return nil, fmt.Errorf("failed to create new WireGuard interface:-> %v", err)
		}
	}
	if iface.Enabled {
		if err := s.wg.SyncToInterface(iface.Ifname, true, iface.PrivateKey); err != nil {
			return nil, fmt.Errorf("failed to bring up new WireGuard interface:-> %v", err)
		}

		// Update VRF binding if needed and interface is up
		if needsVRFUpdate {
			vrfName := ""
			if iface.VRFName != nil {
				vrfName = *iface.VRFName
			}
			if err := utils.SetInterfaceVRF(iface.Ifname, vrfName); err != nil {
				return nil, fmt.Errorf("failed to update VRF binding:-> %v", err)
			}
		}
	}

	if err := s.cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save configuration:-> %v", err)
	}

	return s.sanitizeInterface(iface), nil
}

func (s *InterfaceService) DeleteInterface(id string) error {
	logging.LogInfo("Deleting interface %s", id)
	iface := s.cfg.GetInterface(id)
	if iface == nil {
		return fmt.Errorf("interface not found")
	}

	if err := s.wg.SyncToInterface(iface.Ifname, false, iface.PrivateKey); err != nil {
		return fmt.Errorf("failed to delete WireGuard interface:-> %v", err)
	}
	// Remove WireGuard interface
	if err := s.wg.RemoveConfig(iface.Ifname); err != nil {
		return fmt.Errorf("failed to remove WireGuard config:-> %v", err)
	}

	s.cfg.DeleteInterface(id)
	return s.cfg.Save()
}

func (s *InterfaceService) sanitizeInterface(iface *models.Interface) *models.Interface {
	// Create a copy without the private key
	result := *iface
	result.PrivateKey = ""
	return &result
}

func (s *InterfaceService) CheckIfNameAvailable(ifname string) error {
	// Check if interface exists in OS
	if _, err := net.InterfaceByName(ifname); err == nil {
		return fmt.Errorf("interface '%s' already exists in OS", ifname)
	}

	// Check if WireGuard config file exists
	configPath := filepath.Join(s.cfg.WireGuardConfigPath, ifname+".conf")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("WireGuard config file '%s' already exists", configPath)
	}

	return nil
}

func (s *InterfaceService) CheckUDPPortAvailable(port int) error {
	// Try to bind to the UDP port
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address:-> %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("UDP port %d is not available:-> %v", port, err)
	}
	defer conn.Close()

	return nil
}

func (s *InterfaceService) ValidateEndpoint(endpoint string) (string, error) {
	if endpoint == "" {
		return "", fmt.Errorf("endpoint cannot be empty")
	}

	if len(endpoint) > 2 && endpoint[0] == '[' && endpoint[len(endpoint)-1] == ']' {
		endpoint = endpoint[1 : len(endpoint)-1]
	}

	// Check if it's a valid IPv4 address
	if ip := net.ParseIP(endpoint); ip != nil {
		if ip.To4() != nil {
			return endpoint, nil // Valid IPv4
		}
		if ip.To16() != nil {
			return "[" + endpoint + "]", nil // Valid IPv6
		}
	}

	// Check if it's a valid domain name
	if utils.IsValidDomain(endpoint) {
		return endpoint, nil
	}

	return "", fmt.Errorf("endpoint must be a valid IPv4 address, IPv6 address, or domain name")
}

func (s *InterfaceService) GetInterfaceClientsState(id string) (map[string]map[string]*ClientWithState, int64, error) {
	iface := s.cfg.GetInterface(id)
	milliseconds := time.Now().UnixMilli()
	if iface == nil {
		return nil, milliseconds, fmt.Errorf("interface not found")
	}

	// Get WireGuard stats for the entire interface

	stats, err := s.wg.GetPeerStats(iface.Ifname)
	if err != nil {
		stats = make(map[string]*models.WGState) // Continue with empty stats
	}

	result := make(map[string]map[string]*ClientWithState)

	// Iterate through all servers in this interface
	for _, server := range iface.Servers {
		if !server.Enabled {
			continue // Skip disabled servers
		}

		serverClients := make(map[string]*ClientWithState)

		// Iterate through all clients in this server
		for _, client := range server.Clients {
			if !client.Enabled {
				continue // Skip disabled clients
			}

			// Convert client to frontend format
			clientWithIPs, _ := client.ToClientFrontend(server)

			clientWithState := &ClientWithState{
				ClientFrontend: *clientWithIPs,
			}

			// Add WG state if available
			if state, exists := stats[client.PublicKey]; exists {
				clientWithState.WGState = *state
			}

			serverClients[client.ID] = clientWithState
		}

		if len(serverClients) > 0 {
			result[server.ID] = serverClients
		}
	}

	return result, milliseconds, nil
}

type InterfaceCreateRequest struct {
	Ifname     string  `json:"ifname" binding:"required"`
	VRFName    *string `json:"vrfName"`
	FwMark     *string `json:"fwMark"`
	Endpoint   string  `json:"endpoint" binding:"required"`
	Port       int     `json:"port" binding:"required"`
	MTU        int     `json:"mtu"`
	PrivateKey string  `json:"privateKey"`
}

type InterfaceUpdateRequest struct {
	Ifname     string  `json:"ifname"`
	VRFName    *string `json:"vrfName"`
	FwMark     *string `json:"fwMark"`
	Endpoint   string  `json:"endpoint"`
	Port       int     `json:"port"`
	MTU        int     `json:"mtu"`
	PrivateKey string  `json:"privateKey"`
}
