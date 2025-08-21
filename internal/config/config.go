package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"wg-panel/internal/models"
	"wg-panel/internal/utils"
)

type Session struct {
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	LastSeen  time.Time `json:"lastSeen"`
}

type Config struct {
	ConfigPath          string                       `json:"-"`
	WireGuardConfigPath string                       `json:"wireguardConfigPath"`
	User                string                       `json:"user"`
	Password            string                       `json:"password"`
	ListenIP            string                       `json:"listenIP"`
	ListenPort          int                          `json:"listenPort"`
	SiteURLPrefix       string                       `json:"siteUrlPrefix"`
	SiteFrontendPath    string                       `json:"siteFrontendPath"`
	APIPrefix           string                       `json:"apiPrefix"`
	ServerId            string                       `json:"serverId"`
	Interfaces          map[string]*models.Interface `json:"interfaces"`
	Sessions            map[string]*Session          `json:"sessions"`

	// For thread safety
	mu sync.RWMutex `json:"-"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
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
	if cfg.ServerId == "" {
		serverId, err := utils.GenerateRandomString("", 6)
		if err != nil {
			return nil, fmt.Errorf("failed to generate server ID: %v", err)
		}
		cfg.ServerId = serverId
		// Save the config with the new ServerId
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("failed to save config with new server ID: %v", err)
		}
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
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
