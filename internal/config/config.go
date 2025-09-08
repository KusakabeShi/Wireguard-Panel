package config

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"wg-panel/internal/internalservice"
	"wg-panel/internal/logging"
	"wg-panel/internal/models"
	"wg-panel/internal/utils"

	"github.com/google/uuid"
)

type Session struct {
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	LastSeen  time.Time `json:"lastSeen"`
}

type ToFrontendMessage struct {
	Firewalldefault bool
	InitWarningMsg  string
}

type Config struct {
	ConfigPath          string                       `json:"-"`
	WireGuardConfigPath string                       `json:"wireguardConfigPath"`
	WgIfPrefix          string                       `json:"wgIfPrefix"`
	LogLevel            logging.LogLevel             `json:"logLevel"`
	User                string                       `json:"user"`
	Password            string                       `json:"password"`
	ListenIP            string                       `json:"listenIP"`
	ListenPort          int                          `json:"listenPort"`
	BasePath            string                       `json:"basePath"`
	APIPrefix           string                       `json:"apiPrefix"`
	WGPanelId           string                       `json:"serverId"`
	Interfaces          map[string]*models.Interface `json:"interfaces"`
	Sessions            map[string]*Session          `json:"sessions"`

	// For thread safety
	mu      sync.RWMutex                         `json:"-"`
	FendMsg ToFrontendMessage                    `json:"-"`
	pbs     *internalservice.PseudoBridgeService `json:"-"`
	srs     *internalservice.SNATRoamingService  `json:"-"`
}

func LoadConfig(path string) (*Config, error) {
	logging.LogVerbose("Loading configuration from: %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		logging.LogError("Failed to read config file %s: %v", path, err)
		return nil, fmt.Errorf("failed to read config file:-> %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		logging.LogError("Failed to parse config file %s: %v", path, err)
		return nil, fmt.Errorf("failed to parse config file:-> %v", err)
	}

	cfg.ConfigPath = path

	// Initialize maps if nil
	if cfg.Interfaces == nil {
		cfg.Interfaces = make(map[string]*models.Interface)
	}
	if cfg.Sessions == nil {
		cfg.Sessions = make(map[string]*Session)
	}

	// Generate ServerId if not present
	if cfg.WGPanelId == "" {
		logging.LogInfo("Generating new server ID")
		serverId, err := utils.GenerateRandomString("", 6)
		if err != nil {
			logging.LogError("Failed to generate server ID: %v", err)
			return nil, fmt.Errorf("failed to generate server ID:-> %v", err)
		}
		cfg.WGPanelId = serverId
		logging.LogInfo("Generated server ID: %s", serverId)
		// Save the config with the new ServerId
		if err := cfg.Save(); err != nil {
			logging.LogError("Failed to save config with new server ID: %v", err)
			return nil, fmt.Errorf("failed to save config with new server ID:-> %v", err)
		}
		logging.LogInfo("Saved configuration with new server ID")
	}

	return &cfg, nil
}

func (c *Config) LoadInternalServices(pbs *internalservice.PseudoBridgeService, srs *internalservice.SNATRoamingService, fendMsg ToFrontendMessage) {
	c.pbs = pbs
	c.srs = srs
	c.FendMsg = fendMsg
}

func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config:-> %v", err)
	}

	return utils.WriteFileAtomic(c.ConfigPath, data, 0600)
}

func (c *Config) GetInterface(id string) *models.Interface {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Interfaces[id]
}

func (c *Config) GetAllInterfaces() map[string]*models.Interface {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*models.Interface)
	for k, v := range c.Interfaces {
		result[k] = v
	}
	return result
}

func (c *Config) SetInterface(id string, iface *models.Interface) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Interfaces[id] = iface
}

func (c *Config) DeleteInterface(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Interfaces, id)
}

func (c *Config) GetServer(ifaceID, serverID string) (*models.Server, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	iface, exists := c.Interfaces[ifaceID]
	if !exists {
		return nil, fmt.Errorf("interface not found")
	}

	for _, server := range iface.Servers {
		if server.ID == serverID {
			return server, nil
		}
	}
	return nil, fmt.Errorf("server not found")
}

func (c *Config) GetAllServers(ifaceID string) ([]*models.Server, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	iface, exists := c.Interfaces[ifaceID]
	if !exists {
		return nil, fmt.Errorf("interface not found")
	}

	return iface.Servers, nil
}

func (c *Config) GetAllClients(ifaceID, serverID string) ([]*models.Client, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	server, err := c.getServerUnsafe(ifaceID, serverID)
	if err != nil {
		return nil, err
	}
	return server.Clients, nil
}

func (c *Config) GetClient(ifaceID, serverID, clientID string) (*models.Client, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	server, err := c.getServerUnsafe(ifaceID, serverID)
	if err != nil {
		return nil, err
	}

	for _, client := range server.Clients {
		if client.ID == clientID {
			return client, nil
		}
	}
	return nil, fmt.Errorf("client not found")
}

func (c *Config) getServerUnsafe(ifaceID, serverID string) (*models.Server, error) {
	iface, exists := c.Interfaces[ifaceID]
	if !exists {
		return nil, fmt.Errorf("interface not found")
	}

	for _, server := range iface.Servers {
		if server.ID == serverID {
			return server, nil
		}
	}
	return nil, fmt.Errorf("server not found")
}

func (c *Config) AddSession(token string, session *Session) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Sessions[token] = session
}

func (c *Config) GetSession(token string) *Session {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Sessions[token]
}

func (c *Config) DeleteSession(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Sessions, token)
}

func (c *Config) CleanExpiredSessions() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for token, session := range c.Sessions {
		if now.Sub(session.LastSeen) > 24*time.Hour {
			delete(c.Sessions, token)
		}
	}
}

func (c *Config) CheckNetworkOverlapsInVRF(vrfName *string, skipedIfaceID *string, skipedServerID *string, network *models.IPNetWrapper) error {
	// Get all interfaces in the target VRF
	if network == nil {
		return nil
	}
	for _, iface := range c.GetAllInterfaces() {
		if iface.VRFName != vrfName {
			continue
		}

		if skipedIfaceID != nil && iface.ID == *skipedIfaceID {
			continue
		}

		// Check for network overlaps among child servers
		for _, server := range iface.Servers {
			if skipedServerID != nil && server.ID == *skipedServerID {
				continue
			}
			switch network.Version {
			case 4:
				if server.IPv4 != nil && server.IPv4.Network != nil && server.IPv4.Network.IsOverlap(network) {
					return fmt.Errorf("network %v is overlapped with %v at server %v in interface %v", network, server.IPv4.Network, server.Name, iface.Ifname)
				}
			case 6:
				if server.IPv6 != nil && server.IPv6.Network != nil && server.IPv6.Network.IsOverlap(network) {
					return fmt.Errorf("network %v is overlapped with %v at server %v in interface %v", network, server.IPv6.Network, server.Name, iface.Ifname)
				}
			}
		}

	}
	return nil
}

func (c *Config) CheckSnatOffsetOverlapsInRoamingMasterInterface(ifname string, offset *models.IPNetWrapper, excludeServerID *string) error {
	if offset == nil {
		return nil
	}
	var ifaceNetwork *models.IPNetWrapper
	af := offset.Version
	var aflen int
	if ipv4, ipv6, err := utils.GetInterfaceIP(ifname); err != nil {
		return fmt.Errorf("failed to get interface %v IP address for SNAT roaming:-> %v", ifname, err)
	} else {
		switch af {
		case 4:
			ifaceNetwork = ipv4
			aflen = 32
		case 6:
			ifaceNetwork = ipv6
			aflen = 128
		default:
			return fmt.Errorf("invalid IP version %v for SNAT roaming", af)
		}
	}
	if ifaceNetwork == nil {
		return fmt.Errorf("no IP address found on interface %v for SNAT roaming", ifname)
	}

	if ifaceNetwork.Masklen() > offset.Masklen() {
		return fmt.Errorf("invalid SNAT offset [%v]:-> exceeds the size of the network [%v] on interface %v", offset, ifaceNetwork.Network(), ifname)
	}
	//check offser ip part
	if offset.IpExceed2PowerN(ifaceNetwork.Masklen()) {
		return fmt.Errorf("offset: [%s] exceeds the range of the netowrk: [%s] detected from interface: %v", offset, ifaceNetwork.NetworkStr(), ifname)
	}

	// Verify that the offset network's base IP is properly aligned for its mask
	if !offset.IsHostbitAllZero() {
		return fmt.Errorf("non-zero host bits:-> offset [%s] is not properly aligned for its mask", offset.String())
	}

	for _, iface := range c.GetAllInterfaces() {
		// Check for network overlaps among child servers
		for _, server := range iface.Servers {
			if excludeServerID != nil && server.ID == *excludeServerID {
				continue
			}
			var snatconf *models.SnatConfig
			switch af {
			case 4:
				if server.IPv4 != nil {
					snatconf = server.IPv4.Snat
				}
			case 6:
				if server.IPv6 != nil {
					snatconf = server.IPv6.Snat
				}
			}
			if snatconf == nil || !snatconf.Enabled ||
				snatconf.SnatIPNet == nil || snatconf.SnatIPNet.Masklen() != aflen ||
				snatconf.RoamingMasterInterface == nil || *snatconf.RoamingMasterInterface != ifname {
				continue
			}
			// Check for overlap
			if offset.IsOverlap(snatconf.SnatIPNet) {
				return fmt.Errorf("SNAT offset %v is overlapped with SNAT roaming address %v at server %v in interface %v", offset, snatconf.SnatIPNet, server.Name, iface.Ifname)
			}
		}
	}
	return nil
}

func (c *Config) SyncToInternalService() {
	c.mu.RLock()
	srsConfig := make(map[string]map[string]*models.ServerNetworkConfig)
	pbsConfig := make(map[string]internalservice.ResponderNetworks)
	for _, iface := range c.GetAllInterfaces() {
		// Check for network overlaps among child servers
		if !iface.Enabled {
			continue
		}
		for _, server := range iface.Servers {
			if !server.Enabled {
				continue
			}
			if server.IPv4 != nil && server.IPv4.Enabled {
				if server.IPv4.Network != nil &&
					server.IPv4.PseudoBridgeMasterInterface != nil &&
					*server.IPv4.PseudoBridgeMasterInterface != "" {
					ifname := *server.IPv4.PseudoBridgeMasterInterface
					network := server.IPv4.Network
					var base_net models.IPNetWrapper
					if network != nil {
						base_net = network.Network()
					}
					addPbsConf(pbsConfig, "v4", ifname, &base_net)
					addPbsSkipIP(pbsConfig, "v4", ifname, &server.IPv4.Network.IP)
					addSrsConf(srsConfig, ifname, nil)
				}
				if server.IPv4.Snat != nil && server.IPv4.Snat.Enabled &&
					server.IPv4.Snat.SnatIPNet != nil &&
					server.IPv4.Snat.RoamingMasterInterface != nil &&
					*server.IPv4.Snat.RoamingMasterInterface != "" {
					ifname := *server.IPv4.Snat.RoamingMasterInterface
					network := server.IPv4.Snat.SnatIPNet
					if network.EqualZero(4) && !server.IPv4.Snat.RoamingPseudoBridge {
						//addPbsConf(pbsConfig, "v4o", ifname, network)
						addSrsConf(srsConfig, ifname, server.IPv4)
					} else {
						logging.LogError("Non 0.0.0.0/32 address for snat roaming for interface: %v server: %v at interface %v", iface.Ifname, server.Name, ifname)
					}

				}
			}
			if server.IPv6 != nil && server.IPv6.Enabled {
				if server.IPv6.Network != nil &&
					server.IPv6.PseudoBridgeMasterInterface != nil &&
					*server.IPv6.PseudoBridgeMasterInterface != "" {
					ifname := *server.IPv6.PseudoBridgeMasterInterface
					network := server.IPv6.Network
					var base_net models.IPNetWrapper
					if network != nil {
						base_net = network.Network()
					}
					addPbsConf(pbsConfig, "v6", ifname, &base_net)
					addPbsSkipIP(pbsConfig, "v6", ifname, &server.IPv6.Network.IP)
					addSrsConf(srsConfig, ifname, nil)
				}
				if server.IPv6.Snat != nil && server.IPv6.Snat.Enabled &&
					server.IPv6.Snat.SnatIPNet != nil &&
					server.IPv6.Snat.RoamingMasterInterface != nil &&
					*server.IPv6.Snat.RoamingMasterInterface != "" {
					// Roaming enabled
					ifname := *server.IPv6.Snat.RoamingMasterInterface
					network := server.IPv6.Snat.SnatIPNet

					if network.EqualZero(6) {
						// SNAT Roaming
						if !server.IPv6.Snat.RoamingPseudoBridge {
							// No Roaming pseudo bridge
							//addPbsConf(pbsConfig, "v6o", ifname, network)
							addSrsConf(srsConfig, ifname, server.IPv6)
						} else {
							// Roaming pseudo bridge
							logging.LogError("Roaming pseudo bridge for SNAT mode is not supported, use NETMAP mode instead for interface: %v server: %v at interface %v", iface.Ifname, server.Name, ifname)
						}
					} else {
						// NETMAP Roaming
						if server.IPv6.Network != nil && server.IPv6.Network.Masklen() == network.Masklen() {
							if server.IPv6.Snat.RoamingPseudoBridge {
								// Roaming pseudo bridge
								addPbsConf(pbsConfig, "v6o", ifname, network)
							}
							// Pure NETMAP Roaming
							addSrsConf(srsConfig, ifname, server.IPv6)
						} else {
							logging.LogError("Error to set snat roaming for interface: %v server: %v at interface %v, network.Masklen= %v which is not /128 for SNAT mode, nor same as server network: %v for NETMAP mode", iface.Ifname, server.Name, ifname, network.Masklen(), server.IPv6.Network.String())
						}
					}
				}
			}
		}
	}
	c.mu.RUnlock()
	c.pbs.UpdateConfiguration(pbsConfig)
	c.srs.UpdateConfiguration(srsConfig)
}

func addSrsConf(srsConfig map[string]map[string]*models.ServerNetworkConfig, ifname string, network *models.ServerNetworkConfig) {
	if network == nil {
		srsConfig[ifname] = nil
		return
	}
	key := network.CommentString
	if key == "" {
		logging.LogError("Empty network.CommentString!")
		return
	}
	oldrn, ok := srsConfig[ifname]
	if !ok || oldrn == nil {
		oldrn = make(map[string]*models.ServerNetworkConfig)
		srsConfig[ifname] = oldrn
	}

	oldrn[key] = network.Copy()
	srsConfig[ifname] = oldrn
}

func addPbsConf(pbsConfig map[string]internalservice.ResponderNetworks, target string, ifname string, network *models.IPNetWrapper) {
	if network == nil {
		return
	}
	oldrn, ok := pbsConfig[ifname]
	if !ok {
		oldrn = internalservice.ResponderNetworks{}
	}
	network_copy := &models.IPNetWrapper{}

	*network_copy = *network
	switch target {
	case "v4":
		oldrn.V4Networks = append(oldrn.V4Networks, network_copy)
	case "v6":
		oldrn.V6Networks = append(oldrn.V6Networks, network_copy)
	case "v4o":
		oldrn.V4Offsets = append(oldrn.V4Offsets, network_copy)
	case "v6o":
		oldrn.V6Offsets = append(oldrn.V6Offsets, network_copy)
	}

	pbsConfig[ifname] = oldrn
}

func addPbsSkipIP(pbsConfig map[string]internalservice.ResponderNetworks, target string, ifname string, skipIP *net.IP) {
	if skipIP == nil {
		return
	}
	oldrn, ok := pbsConfig[ifname]
	if !ok {
		oldrn = internalservice.ResponderNetworks{}
	}

	var waitAdd []net.IP

	switch target {
	case "v4":
		waitAdd = oldrn.V4Skipped
	case "v6":
		waitAdd = oldrn.V6Skipped
	}
	for _, ip := range waitAdd {
		if ip.Equal(*skipIP) {
			// already exists
			return
		}
	}
	waitAdd = append(waitAdd, *skipIP)
	switch target {
	case "v4":
		oldrn.V4Skipped = waitAdd
	case "v6":
		oldrn.V6Skipped = waitAdd
	}

	pbsConfig[ifname] = oldrn
}

func (c *Config) GetAvailableInterfaceID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	pfx := "i"
	ids := map[string]bool{}
	for _, insts := range c.GetAllInterfaces() {
		ids[insts.ID] = true
	}
	for i := 0; i <= 99999; i++ {
		id := fmt.Sprintf("%s%d", pfx, i)
		if _, ok := ids[id]; !ok {
			return id
		}
	}
	return uuid.New().String()
}
func (c *Config) GetAvailableServerID(ifaceID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	pfx := "s"
	ids := map[string]bool{}
	instances, err := c.GetAllServers(ifaceID)
	if err != nil {
		return uuid.New().String()
	}
	for _, instance := range instances {
		ids[instance.ID] = true
	}
	for i := 0; i <= 99999; i++ {
		id := fmt.Sprintf("%s%d", pfx, i)
		if _, ok := ids[id]; !ok {
			return id
		}
	}
	return uuid.New().String()
}
func (c *Config) GetAvailableClientID(ifaceID, serverID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	pfx := "c"
	ids := map[string]bool{}
	instances, err := c.GetAllClients(ifaceID, serverID)
	if err != nil {
		return uuid.New().String()
	}
	for _, instance := range instances {
		ids[instance.ID] = true
	}
	for i := 0; i <= 99999; i++ {
		id := fmt.Sprintf("%s%d", pfx, i)
		if _, ok := ids[id]; !ok {
			return id
		}
	}
	return uuid.New().String()
}
