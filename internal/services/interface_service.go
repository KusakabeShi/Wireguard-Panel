package services

import (
	"fmt"
	"regexp"

	"wg-panel/internal/config"
	"wg-panel/internal/models"
	"wg-panel/internal/utils"

	"github.com/google/uuid"
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
	if !isValidIfname(req.Ifname) {
		return nil, fmt.Errorf("invalid ifname: must match ^[A-Za-z0-9_-]{1,12}$")
	}

	// Check if ifname already exists
	for _, iface := range s.cfg.GetAllInterfaces() {
		if iface.Ifname == req.Ifname {
			return nil, fmt.Errorf("interface with ifname '%s' already exists", req.Ifname)
		}
	}

	// Generate private key if not provided
	privateKey := req.PrivateKey
	if privateKey == "" {
		var err error
		privateKey, err = utils.GenerateWGPrivateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %v", err)
		}
	}

	// Generate public key
	publicKey, err := utils.PrivToPublic(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %v", err)
	}

	// Set defaults
	if req.MTU <= 0 {
		req.MTU = 1420
	}

	iface := &models.Interface{
		ID:         uuid.New().String(),
		Ifname:     req.Ifname,
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
		return nil, fmt.Errorf("failed to save configuration: %v", err)
	}

	// Generate and apply WireGuard configuration
	if err := s.wg.GenerateAndSyncInterface(iface); err != nil {
		return nil, fmt.Errorf("failed to apply WireGuard configuration: %v", err)
	}

	return s.sanitizeInterface(iface), nil
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

	needsWGRegeneration := false
	needsMTUUpdate := false

	// Update fields
	if req.Ifname != "" && req.Ifname != iface.Ifname {
		if !isValidIfname(req.Ifname) {
			return nil, fmt.Errorf("invalid ifname: must match ^[A-Za-z0-9_-]{1,12}$")
		}
		// Check if new ifname already exists
		for _, otherIface := range s.cfg.GetAllInterfaces() {
			if otherIface.ID != id && otherIface.Ifname == req.Ifname {
				return nil, fmt.Errorf("interface with ifname '%s' already exists", req.Ifname)
			}
		}
		iface.Ifname = req.Ifname
	}

	if req.VRFName != nil && (iface.VRFName == nil || *req.VRFName != *iface.VRFName) {
		// Check for network overlaps when changing VRF
		if err := s.checkVRFNetworkOverlaps(iface, req.VRFName); err != nil {
			return nil, err
		}
		iface.VRFName = req.VRFName
		needsWGRegeneration = true
	}

	if req.FwMark != nil && (iface.FwMark == nil || *req.FwMark != *iface.FwMark) {
		iface.FwMark = req.FwMark
		needsWGRegeneration = true
	}

	if req.Endpoint != "" && req.Endpoint != iface.Endpoint {
		iface.Endpoint = req.Endpoint
	}

	if req.Port > 0 && req.Port != iface.Port {
		iface.Port = req.Port
		needsWGRegeneration = true
	}

	if req.MTU > 0 && req.MTU != iface.MTU {
		iface.MTU = req.MTU
		needsMTUUpdate = true
	}

	if req.PrivateKey != "" && req.PrivateKey != iface.PrivateKey {
		publicKey, err := utils.PrivToPublic(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to generate public key: %v", err)
		}
		iface.PrivateKey = req.PrivateKey
		iface.PublicKey = publicKey
		needsWGRegeneration = true
	}

	if err := s.cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %v", err)
	}

	// Apply system changes
	if needsWGRegeneration {
		if err := s.wg.GenerateAndSyncInterface(iface); err != nil {
			return nil, fmt.Errorf("failed to regenerate WireGuard configuration: %v", err)
		}
	}

	if needsMTUUpdate {
		if err := s.wg.SetInterfaceMTU(iface.Ifname, iface.MTU); err != nil {
			return nil, fmt.Errorf("failed to update MTU: %v", err)
		}
	}

	return s.sanitizeInterface(iface), nil
}

func (s *InterfaceService) DeleteInterface(id string) error {
	iface := s.cfg.GetInterface(id)
	if iface == nil {
		return fmt.Errorf("interface not found")
	}

	// Remove WireGuard interface
	if err := s.wg.RemoveInterface(iface.Ifname); err != nil {
		return fmt.Errorf("failed to remove WireGuard interface: %v", err)
	}

	s.cfg.DeleteInterface(id)
	return s.cfg.Save()
}

func (s *InterfaceService) checkVRFNetworkOverlaps(iface *models.Interface, newVRFName *string) error {
	// Get all interfaces in the target VRF
	for _, otherIface := range s.cfg.GetAllInterfaces() {
		if otherIface.ID == iface.ID {
			continue
		}

		// Check if other interface is in the same VRF
		if (newVRFName == nil && otherIface.VRFName == nil) ||
			(newVRFName != nil && otherIface.VRFName != nil && *newVRFName == *otherIface.VRFName) {
			
			// Check for network overlaps among child servers
			for _, server := range iface.Servers {
				for _, otherServer := range otherIface.Servers {
					if s.serverNetworksOverlap(server, otherServer) {
						return fmt.Errorf("server network overlap detected in VRF")
					}
				}
			}
		}
	}
	return nil
}

func (s *InterfaceService) serverNetworksOverlap(s1, s2 *models.Server) bool {
	if s1.IPv4 != nil && s2.IPv4 != nil && s1.IPv4.Enabled && s2.IPv4.Enabled {
		if s1.IPv4.Network != nil && s2.IPv4.Network != nil {
			if s1.IPv4.Network.IsOverlap(s2.IPv4.Network) {
				return true
			}
		}
	}
	if s1.IPv6 != nil && s2.IPv6 != nil && s1.IPv6.Enabled && s2.IPv6.Enabled {
		if s1.IPv6.Network != nil && s2.IPv6.Network != nil {
			if s1.IPv6.Network.IsOverlap(s2.IPv6.Network) {
				return true
			}
		}
	}
	return false
}

func (s *InterfaceService) sanitizeInterface(iface *models.Interface) *models.Interface {
	// Create a copy without the private key
	result := *iface
	result.PrivateKey = ""
	return &result
}

func isValidIfname(ifname string) bool {
	matched, _ := regexp.MatchString("^[A-Za-z0-9_-]{1,12}$", ifname)
	return matched
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