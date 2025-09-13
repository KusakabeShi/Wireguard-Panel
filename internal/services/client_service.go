package services

import (
	"fmt"
	"net"
	"strings"

	"wg-panel/internal/config"
	"wg-panel/internal/models"
	"wg-panel/internal/utils"
)

type ClientService struct {
	cfg *config.Config
	wg  *WireGuardService
}

func NewClientService(cfg *config.Config, wgService *WireGuardService) *ClientService {
	return &ClientService{
		cfg: cfg,
		wg:  wgService,
	}
}

func (s *ClientService) ToClientFrontend(ifid string, sid string, c *models.Client) (*models.ClientFrontend, error) {
	if c == nil {
		return nil, nil
	}

	server, err := s.cfg.GetServer(ifid, sid)
	if err != nil {
		return nil, err
	}

	return c.ToClientFrontend(server)
}

func (s *ClientService) CreateClient(interfaceID, serverID string, req ClientCreateRequest) (*models.Client, error) {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return nil, fmt.Errorf("interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return nil, err
	}

	if err := utils.IsSafeName(req.Name); err != nil {
		return nil, fmt.Errorf("request validation failed:-> %v", err)
	}

	// Validate that at least one IP is requested
	if (req.IP == nil || *req.IP == "") && (req.IPv6 == nil || *req.IPv6 == "") {
		return nil, fmt.Errorf("at least one of IPv4 or IPv6 must be specified")
	}

	// Generate keypair if needed
	var privateKey, publicKey string
	if req.PrivateKey != nil && *req.PrivateKey != "" {
		privateKey = *req.PrivateKey
		publicKey, err = utils.PrivToPublic(privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to derive public key:-> %v", err)
		}
	} else if req.PublicKey != nil && *req.PublicKey != "" {
		publicKey = *req.PublicKey
	} else {
		privateKey, publicKey, err = utils.GenerateWGKeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate keypair:-> %v", err)
		}
	}
	for _, dns := range req.DNS {
		if err := utils.ValidateIPorDomain(dns); err != nil {
			return nil, fmt.Errorf("request validation failed:-> %v", err)
		}
	}

	client := &models.Client{
		ID:           s.cfg.GetAvailableClientID(iface.ID, serverID),
		Name:         req.Name,
		Enabled:      false, // Always start disabled
		DNS:          req.DNS,
		PublicKey:    publicKey,
		PresharedKey: req.PresharedKey,
		Keepalive:    req.Keepalive,
	}

	if privateKey != "" {
		client.PrivateKey = &privateKey
	}

	// Allocate IP addresses
	if req.IP != nil && *req.IP != "" && server.IPv4 != nil && server.IPv4.Enabled {
		if _, err := s.allocateIPv4(client, server, *req.IP); err != nil {
			return nil, fmt.Errorf("IPv4 allocation failed:-> %v", err)
		}
	}

	if req.IPv6 != nil && *req.IPv6 != "" && server.IPv6 != nil && server.IPv6.Enabled {
		if _, err := s.allocateIPv6(client, server, *req.IPv6); err != nil {
			return nil, fmt.Errorf("IPv6 allocation failed:-> %v", err)
		}
	}

	// Add client to server
	server.Clients = append(server.Clients, client)
	s.cfg.SetInterface(interfaceID, iface)

	if err := s.cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save configuration:-> %v", err)
	}

	return client, nil
}

func (s *ClientService) GetClient(interfaceID, serverID, clientID string) (*models.Client, error) {
	client, err := s.cfg.GetClient(interfaceID, serverID, clientID)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (s *ClientService) GetClients(interfaceID, serverID string) ([]*models.Client, error) {
	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return nil, err
	}

	clients := make([]*models.Client, len(server.Clients))
	copy(clients, server.Clients)

	return clients, nil
}

func (s *ClientService) GetClientsFrontendWithState(interfaceID, serverID string) ([]*ClientWithState, error) {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return nil, fmt.Errorf("interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return nil, err
	}

	// Get WireGuard stats
	stats, err := s.wg.GetPeerStats(iface.Ifname)
	if err != nil {
		stats = make(map[string]*models.WGState) // Continue with empty stats
	}

	clients := make([]*ClientWithState, len(server.Clients))
	for i, client := range server.Clients {
		clientWithIPs, _ := s.ToClientFrontend(interfaceID, serverID, client)

		clientWithState := &ClientWithState{
			ClientFrontend: *clientWithIPs,
		}

		if state, exists := stats[client.PublicKey]; exists {
			clientWithState.WGState = *state
		}

		clients[i] = clientWithState
	}

	return clients, nil
}

func (s *ClientService) UpdateClient(interfaceID, serverID, clientID string, req ClientUpdateRequest) (*models.Client, error) {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return nil, fmt.Errorf("interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return nil, err
	}

	client, err := s.cfg.GetClient(interfaceID, serverID, clientID)
	if err != nil {
		return nil, err
	}

	if err := utils.IsSafeName(req.Name); err != nil {
		return nil, fmt.Errorf("request validation failed:-> %v", err)
	}
	for _, dns := range req.DNS {
		if err := utils.ValidateIPorDomain(dns); err != nil {
			return nil, fmt.Errorf("request validation failed:-> %v", err)
		}
	}

	needsWGSync := false

	// Update basic fields
	if req.Name != "" {
		client.Name = req.Name
	}
	if req.DNS != nil {
		client.DNS = req.DNS
	}
	if req.Keepalive != nil {
		client.Keepalive = req.Keepalive
		needsWGSync = true
	}

	for _, dns := range req.DNS {
		if err := utils.ValidateIPorDomain(dns); err != nil {
			return nil, fmt.Errorf("request validation failed:-> %v", err)
		}
	}

	// Update keys
	if req.PrivateKey != nil {
		if *req.PrivateKey != "" {
			publicKey, err := utils.PrivToPublic(*req.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to derive public key:-> %v", err)
			}
			client.PrivateKey = req.PrivateKey
			client.PublicKey = publicKey
			needsWGSync = true
		}
	} else if req.PublicKey != nil && *req.PublicKey != "" {
		client.PublicKey = *req.PublicKey
		client.PrivateKey = nil
		needsWGSync = true
	}

	if req.PresharedKey != nil {
		if *req.PresharedKey == "" {
			client.PresharedKey = nil
		} else {
			client.PresharedKey = req.PresharedKey
		}
		needsWGSync = true
	}

	// Update IP addresses
	if req.IP == nil || *req.IP == "" {
		needsWGSync = client.IPv4Offset != nil || needsWGSync
		client.IPv4Offset = nil
	} else {
		if changed, err := s.updateClientIPv4(client, server, *req.IP); err != nil {
			return nil, fmt.Errorf("IPv4 update failed:-> %v", err)
		} else if changed {
			needsWGSync = true
		}
	}

	if req.IPv6 == nil || *req.IPv6 == "" {
		needsWGSync = client.IPv6Offset != nil || needsWGSync
		client.IPv6Offset = nil
	} else {
		if changed, err := s.updateClientIPv6(client, server, *req.IPv6); err != nil {
			return nil, fmt.Errorf("IPv6 update failed:-> %v", err)
		} else if changed {
			needsWGSync = true
		}
	}

	s.cfg.SetInterface(interfaceID, iface)
	if err := s.cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save configuration:-> %v", err)
	}

	// Sync WireGuard if needed
	if needsWGSync && server.Enabled {
		if err := s.wg.SyncToConfAndInterface(iface); err != nil {
			return nil, fmt.Errorf("failed to sync WireGuard configuration:-> %v", err)
		}
	}

	return client, nil
}

func (s *ClientService) SetClientEnabled(interfaceID, serverID, clientID string, enabled bool) error {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return fmt.Errorf("interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return err
	}

	client, err := s.cfg.GetClient(interfaceID, serverID, clientID)
	if err != nil {
		return err
	}

	if client.Enabled == enabled {
		return nil // Already in desired state
	}

	client.Enabled = enabled
	s.cfg.SetInterface(interfaceID, iface)
	if err := s.cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration:-> %v", err)
	}

	// Sync WireGuard configuration
	if server.Enabled {
		if err := s.wg.SyncToConfAndInterface(iface); err != nil {
			return fmt.Errorf("failed to sync WireGuard configuration:-> %v", err)
		}
	}

	return nil
}

func (s *ClientService) DeleteClient(interfaceID, serverID, clientID string) error {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return fmt.Errorf("interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return err
	}

	// Disable first
	if client, err := s.cfg.GetClient(interfaceID, serverID, clientID); err == nil && client.Enabled {
		if err := s.SetClientEnabled(interfaceID, serverID, clientID, false); err != nil {
			return fmt.Errorf("failed to disable client before deletion:-> %v", err)
		}
	}

	// Remove client from server
	for i, client := range server.Clients {
		if client.ID == clientID {
			server.Clients = append(server.Clients[:i], server.Clients[i+1:]...)
			break
		}
	}

	s.cfg.SetInterface(interfaceID, iface)
	return s.cfg.Save()
}

func (s *ClientService) GetClientConfig(interfaceID, serverID, clientID string) (string, error) {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return "", fmt.Errorf("interface not found")
	}

	server, err := s.cfg.GetServer(interfaceID, serverID)
	if err != nil {
		return "", err
	}

	client, err := s.cfg.GetClient(interfaceID, serverID, clientID)
	if err != nil {
		return "", err
	}

	return s.generateClientConfig(iface, server, client), nil
}

func (s *ClientService) GetClientWGState(interfaceID, serverID, clientID string) (*models.WGState, error) {
	iface := s.cfg.GetInterface(interfaceID)
	if iface == nil {
		return nil, fmt.Errorf("interface not found")
	}

	client, err := s.cfg.GetClient(interfaceID, serverID, clientID)
	if err != nil {
		return nil, err
	}

	stats, err := s.wg.GetPeerStats(iface.Ifname)
	if err != nil {
		return &models.WGState{}, nil // Return empty state on error
	}

	if state, exists := stats[client.PublicKey]; exists {
		return state, nil
	}

	return &models.WGState{}, nil
}

func (s *ClientService) allocateIPv4(client *models.Client, server *models.Server, ipRequest string) (changed bool, err error) {
	if server.IPv4 == nil || !server.IPv4.Enabled || server.IPv4.Network == nil {
		return false, fmt.Errorf("server does not have IPv4 enabled")
	}

	if ipRequest == "auto" {
		return s.autoAllocateIPv4(client, server)
	}

	// Manual IP assignment
	ip := net.ParseIP(ipRequest)
	if ip == nil || ip.To4() == nil {
		return false, fmt.Errorf("invalid IPv4 address")
	}

	return client.SetIP(4, server.IPv4.Network, ip, server.Clients)
}

func (s *ClientService) allocateIPv6(client *models.Client, server *models.Server, ipRequest string) (changed bool, err error) {
	if server.IPv6 == nil || !server.IPv6.Enabled || server.IPv6.Network == nil {
		return false, fmt.Errorf("server does not have IPv6 enabled")
	}

	if ipRequest == "auto" {
		return s.autoAllocateIPv6(client, server)
	}

	// Manual IP assignment
	ip := net.ParseIP(ipRequest)
	if ip == nil || ip.To4() != nil {
		return false, fmt.Errorf("invalid IPv6 address")
	}

	return client.SetIP(6, server.IPv6.Network, ip, server.Clients)
}

func (s *ClientService) autoAllocateIPv4(client *models.Client, server *models.Server) (changed bool, err error) {
	network := server.IPv4.Network
	masklen := network.Masklen()
	hostBits := 32 - masklen

	if hostBits <= 2 {
		return false, fmt.Errorf("network too small for client allocation")
	}

	// Collect used IPs
	usedOffsets := make(map[string]bool)
	for _, otherClient := range server.Clients {
		if otherClient.ID != client.ID && otherClient.IPv4Offset != nil {
			usedOffsets[string(otherClient.IPv4Offset)] = true
		}
	}

	// Reserve network and broadcast addresses, and server IP
	networkOffset := make(net.IP, 4)
	usedOffsets[string(networkOffset)] = true

	broadcastOffset := models.IncrementIP2Power(networkOffset, hostBits)[:4]
	broadcastOffset = broadcastOffset[len(broadcastOffset)-4:]
	usedOffsets[string(broadcastOffset)] = true

	serverOffset, _ := network.GetOffset()
	usedOffsets[string(serverOffset)] = true

	// Find first available IP
	for i := 1; i < (1<<uint(hostBits))-1; i++ {
		offset := make(models.IPWrapper, 4)
		offset[3] = byte(i)

		if i >= 256 {
			offset[2] = byte(i >> 8)
			offset[3] = byte(i & 0xFF)
		}
		if i >= 65536 {
			offset[1] = byte(i >> 16)
			offset[2] = byte((i >> 8) & 0xFF)
			offset[3] = byte(i & 0xFF)
		}
		if i >= 16777216 {
			offset[0] = byte(i >> 24)
			offset[1] = byte((i >> 16) & 0xFF)
			offset[2] = byte((i >> 8) & 0xFF)
			offset[3] = byte(i & 0xFF)
		}

		if !usedOffsets[string(offset)] {
			client.IPv4Offset = offset
			return true, nil
		}
	}

	return false, fmt.Errorf("no available IPv4 addresses in network")
}

func (s *ClientService) autoAllocateIPv6(client *models.Client, server *models.Server) (changed bool, err error) {
	network := server.IPv6.Network
	masklen := network.Masklen()
	hostBits := 128 - masklen

	if hostBits <= 1 {
		return false, fmt.Errorf("network too small for client allocation")
	}

	// Collect used IPs
	usedOffsets := make(map[string]bool)
	for _, otherClient := range server.Clients {
		if otherClient.ID != client.ID && otherClient.IPv6Offset != nil {
			usedOffsets[string(otherClient.IPv6Offset)] = true
		}
	}

	// Reserve network address and server IP
	networkOffset := make(net.IP, 16)
	usedOffsets[string(networkOffset)] = true

	serverOffset, _ := network.GetOffset()
	usedOffsets[string(serverOffset)] = true

	// Find first available IP (simple sequential allocation)
	for i := 1; i < 65536; i++ { // Limit search for practicality
		offset := make(models.IPWrapper, 16)
		offset[15] = byte(i & 0xFF)
		offset[14] = byte((i >> 8) & 0xFF)

		if !usedOffsets[string(offset)] {
			client.IPv6Offset = offset
			return true, nil
		}
	}

	return false, fmt.Errorf("no available IPv6 addresses in network")
}

func (s *ClientService) updateClientIPv4(client *models.Client, server *models.Server, ipStr string) (changed bool, err error) {
	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() == nil {
		return false, fmt.Errorf("invalid IPv4 address")
	}
	ip = ip.To4()
	return client.SetIP(4, server.IPv4.Network, ip, server.Clients)
}

func (s *ClientService) updateClientIPv6(client *models.Client, server *models.Server, ipStr string) (changed bool, err error) {
	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() != nil {
		return false, fmt.Errorf("invalid IPv6 address")
	}

	return client.SetIP(6, server.IPv6.Network, ip, server.Clients)
}

func (s *ClientService) generateClientConfig(iface *models.Interface, server *models.Server, client *models.Client) string {
	var config strings.Builder

	config.WriteString("[Interface]\n")

	// Private key
	if client.PrivateKey != nil {
		config.WriteString(fmt.Sprintf("PrivateKey = %s\n", *client.PrivateKey))
	} else {
		config.WriteString("PrivateKey = [privkey is not available for this client]\n")
	}

	// Address
	addresses := make([]string, 0)
	if client.IPv4Offset != nil && server.IPv4 != nil && server.IPv4.Enabled {
		if ipNet, err := client.GetIPv4(server.IPv4.Network); err == nil && ipNet != nil {
			addresses = append(addresses, fmt.Sprintf("%s/32", ipNet.IP.String()))
		}
	}
	if client.IPv6Offset != nil && server.IPv6 != nil && server.IPv6.Enabled {
		if ipNet, err := client.GetIPv6(server.IPv6.Network); err == nil && ipNet != nil {
			addresses = append(addresses, fmt.Sprintf("%s/128", ipNet.IP.String()))
		}
	}
	if len(addresses) > 0 {
		config.WriteString(fmt.Sprintf("Address = %s\n", strings.Join(addresses, ", ")))
	}
	config.WriteString(fmt.Sprintf("MTU = %d\n", iface.MTU))

	// DNS
	dns := client.DNS
	if dns == nil {
		dns = server.DNS
	}
	if len(dns) > 0 {
		config.WriteString(fmt.Sprintf("DNS = %s\n", strings.Join(dns, ", ")))
	}

	config.WriteString("\n[Peer]\n")
	config.WriteString(fmt.Sprintf("PublicKey = %s\n", iface.PublicKey))

	if client.PresharedKey != nil {
		config.WriteString(fmt.Sprintf("PresharedKey = %s\n", *client.PresharedKey))
	}

	// AllowedIPs (routed networks)
	allowedIPs := make([]string, 0)
	if server.IPv4 != nil && server.IPv4.Enabled {
		for _, routedNet := range server.IPv4.RoutedNetworks {
			allowedIPs = append(allowedIPs, routedNet.BaseNet.String())
		}
	}
	if server.IPv6 != nil && server.IPv6.Enabled {
		for _, routedNet := range server.IPv6.RoutedNetworks {
			allowedIPs = append(allowedIPs, routedNet.BaseNet.String())
		}
	}
	if len(allowedIPs) > 0 {
		config.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(allowedIPs, ", ")))
	}

	config.WriteString(fmt.Sprintf("Endpoint = %s:%d\n", iface.Endpoint, iface.Port))

	if client.Keepalive != nil && *client.Keepalive > 0 {
		config.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", *client.Keepalive))
	}

	return config.String()
}

// Request types
type ClientCreateRequest struct {
	Name         string   `json:"name" binding:"required"`
	IP           *string  `json:"ip"`
	IPv6         *string  `json:"ipv6"`
	DNS          []string `json:"dns"`
	PrivateKey   *string  `json:"privateKey"`
	PublicKey    *string  `json:"publicKey"`
	PresharedKey *string  `json:"presharedKey"`
	Keepalive    *uint    `json:"keepalive"`
}

type ClientUpdateRequest struct {
	Name         string   `json:"name"`
	IP           *string  `json:"ip"`
	IPv6         *string  `json:"ipv6"`
	DNS          []string `json:"dns"`
	PrivateKey   *string  `json:"privateKey"`
	PublicKey    *string  `json:"publicKey"`
	PresharedKey *string  `json:"presharedKey"`
	Keepalive    *uint    `json:"keepalive"`
}

type ClientWithState struct {
	models.ClientFrontend
	models.WGState
}
