package internalservice

import (
	"fmt"
	"log"
	"sync"
	"time"

	"wg-panel/internal/models"
	"wg-panel/internal/utils"

	"github.com/vishvananda/netlink"
)

type SNATRoamingService struct {
	runningInterface    map[string]*SNATRoamingSynced
	listeners           map[string]*InterfaceIPNetListener
	mu                  sync.RWMutex
	pseudoBridgeService *PseudoBridgeService
	fw                  *FirewallService
	stopCh              chan struct{}
}

type SNATRoamingSynced struct {
	ExistsInNewConfig bool
}

type InterfaceIPNetListener struct {
	interfaceName       string
	configs             map[string]*models.ServerNetworkConfig
	stopCh              chan struct{}
	ifIPs               map[int]*models.IPNetWrapper
	fw                  *FirewallService
	mu                  sync.RWMutex
	configUpdate        chan []models.IPNetWrapper
	pseudoBridgeService *PseudoBridgeService
}

func NewSNATRoamingService(pseudoBridgeService *PseudoBridgeService, firewallService *FirewallService) *SNATRoamingService {
	s := &SNATRoamingService{
		runningInterface:    make(map[string]*SNATRoamingSynced),
		listeners:           make(map[string]*InterfaceIPNetListener),
		pseudoBridgeService: pseudoBridgeService,
		fw:                  firewallService,
	}
	log.Println("Starting SNAT Roaming Service")
	go s.mainLoop()
	return s
}

func (s *SNATRoamingService) UpdateConfiguration(waitnigInterface map[string]map[string]*models.ServerNetworkConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	addIF := make(map[string]map[string]*models.ServerNetworkConfig)
	updateIF := make(map[string]map[string]*models.ServerNetworkConfig)
	delIF := make(map[string]*SNATRoamingSynced)

	for _, runningNets := range s.runningInterface {
		runningNets.ExistsInNewConfig = false
	}

	for newIF, newNets := range waitnigInterface {
		runningIF, ok := s.runningInterface[newIF]
		if ok {
			runningIF.ExistsInNewConfig = true
			updateIF[newIF] = newNets

		} else {
			addIF[newIF] = newNets
		}
	}
	for runningIF, runningNets := range s.runningInterface {
		if !runningNets.ExistsInNewConfig {
			delIF[runningIF] = runningNets
		}
	}

	for ifname := range addIF {
		s.runningInterface[ifname] = &SNATRoamingSynced{
			ExistsInNewConfig: false,
		}
		s.listeners[ifname] = NewInterfaceIPNetListener(ifname, s.pseudoBridgeService, s.fw)
	}
	for ifname, configs := range updateIF {
		s.listeners[ifname].UpdateConfigs(configs)
	}
	for ifname := range delIF {
		s.listeners[ifname].Stop()
		delete(s.runningInterface, ifname)
		delete(s.listeners, ifname)
	}
}

func (s *SNATRoamingService) mainLoop() {
	linkUpdatesSubscribed := false
	addrUpdatesSubscribed := false
	linkUpdates := make(chan netlink.LinkUpdate, 10)
	addrUpdates := make(chan netlink.AddrUpdate, 10)
	defer func() {
		fmt.Printf("SNATRoamingService stopped")
		close(s.stopCh)
	}()
	log.Println("Starting SNATRoamingService")
	for {
		// Try to open pcap handle for the interface
		if !linkUpdatesSubscribed {
			// Subscribe to link updates for interface up/down events
			if err := netlink.LinkSubscribe(linkUpdates, s.stopCh); err != nil {
				log.Printf("failed to subscribe to link updates: %v, retrying in 5 seconds:", err)
				time.Sleep(5 * time.Second)
				continue
			}
			linkUpdatesSubscribed = true
		}
		if !addrUpdatesSubscribed {
			// Subscribe to address updates for IP address changes
			if err := netlink.AddrSubscribe(addrUpdates, s.stopCh); err != nil {
				log.Printf("failed to subscribe to address updates: %v, retrying in 5 seconds:", err)
				time.Sleep(5 * time.Second)
				continue
			}
			addrUpdatesSubscribed = true
		}
		if linkUpdatesSubscribed && addrUpdatesSubscribed {
			select {
			case <-s.stopCh:
				return
			case linkUpdate, ok := <-linkUpdates:
				if !ok {
					log.Printf("Link updates channel closed, attempting to resubscribe in 5 seconds")
					linkUpdatesSubscribed = false
					continue
				}
				ifname, err := getNetlinkIfName(linkUpdate)
				if err != nil {
					log.Printf("Failed to parse ifname from linkUpdate: %v", err)
				}
				s.mu.RLock()
				listener, ok := s.listeners[ifname]
				s.mu.RUnlock()
				if ok {
					listener.IPaddrUpdate()
				}

			case addrUpdate, ok := <-addrUpdates:
				if !ok {
					log.Printf("Address updates channel closed, attempting to resubscribe in 5 seconds")
					addrUpdatesSubscribed = false
					continue
				}
				ifname, err := getAddrUpdateIfName(addrUpdate)
				if err != nil {
					log.Printf("Failed to parse ifname from linkUpdate: %v", err)
				}
				s.mu.RLock()
				listener, ok := s.listeners[ifname]
				s.mu.RUnlock()
				if ok {
					listener.IPaddrUpdate()
				}
			}
		}
	}
}

func (l *InterfaceIPNetListener) mainLoop() {
	log.Printf("Starting InterfaceIPNetListener for %v", l.interfaceName)
	defer func() {
		close(l.stopCh)
	}()
	for {
		select {
		case <-l.stopCh:
			return
		case <-l.configUpdate:

		}
	}
}

func (s *SNATRoamingService) Stop() {
	// Stop all listeners
	for _, listener := range s.listeners {
		listener.Stop()
	}
	s.listeners = make(map[string]*InterfaceIPNetListener)

	log.Println("SNAT Roaming Service stopped")
}

func NewInterfaceIPNetListener(interfaceName string, pseudoBridgeService *PseudoBridgeService, fw *FirewallService) *InterfaceIPNetListener {
	r := &InterfaceIPNetListener{
		interfaceName:       interfaceName,
		configs:             make(map[string]*models.ServerNetworkConfig),
		stopCh:              make(chan struct{}),
		ifIPs:               make(map[int]*models.IPNetWrapper),
		fw:                  fw,
		configUpdate:        make(chan []models.IPNetWrapper),
		pseudoBridgeService: pseudoBridgeService,
	}
	r.ifIPs[4] = nil
	r.ifIPs[6] = nil

	r.mainLoop()
	return r
}

func (l *InterfaceIPNetListener) UpdateConfigs(configs map[string]*models.ServerNetworkConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()
	oldsynced := make(map[string]bool)
	for k := range l.configs {
		oldsynced[k] = false
	}

	toAdd := make(map[string]*models.ServerNetworkConfig)
	toUpdate := make(map[string]*models.ServerNetworkConfig)
	toDel := make(map[string]*models.ServerNetworkConfig)

	for key, newconf := range configs {
		if newconf == nil || !newconf.Enabled || newconf.Snat == nil ||
			newconf.Snat.RoamingMasterInterface == nil || len(*newconf.Snat.RoamingMasterInterface) == 0 {
			continue
		}

		oldconf, ok := l.configs[key]
		if ok {
			oldsynced[key] = true
			if !oldconf.Network.Equal(newconf.Network) || !oldconf.Snat.SnatExcludedNetwork.Equal(newconf.Snat.SnatExcludedNetwork) || !models.NetworksEqual(oldconf.RoutedNetworks, newconf.RoutedNetworks) {
				toUpdate[key] = newconf
			}
		} else {
			toAdd[key] = newconf
		}
	}
	for key, found := range oldsynced {
		if !found {
			toDel[key] = l.configs[key]
		}
	}
	for _, config := range toAdd {
		simulatedConfig, err := l.getSimulatedConfig(config)
		if err != nil {
			log.Printf("failed to calculate target_network: %v", err)
		}
		if err := l.fw.AddIpAndFwRules(l.interfaceName, simulatedConfig); err != nil {
			log.Printf("failed to add firewall rules: %v", err)
		}
	}
	for _, config := range toUpdate {
		simulatedConfig, err := l.getSimulatedConfig(config)
		if err != nil {
			log.Printf("failed to calculate target_network: %v", err)
		}
		l.fw.RemoveSnatRules(l.interfaceName, simulatedConfig, simulatedConfig.CommentString)
		if err := l.fw.AddSnatRules(l.interfaceName, simulatedConfig, simulatedConfig.CommentString); err != nil {
			log.Printf("failed to add firewall rules: %v", err)
		}
	}
	for _, config := range toDel {
		simulatedConfig, err := l.getSimulatedConfig(config)
		if err != nil {
			log.Printf("failed to calculate target_network: %v", err)
		}
		l.fw.RemoveSnatRules(l.interfaceName, simulatedConfig, simulatedConfig.CommentString)
	}
}

func (l *InterfaceIPNetListener) getSimulatedConfig(config *models.ServerNetworkConfig) (*models.ServerNetworkConfig, error) {

	target_network, err := config.Network.GetNetByOffset(config.Snat.SnatIPNet)
	if err != nil {
		return nil, fmt.Errorf("error get target_network, err: %v", err)
	}
	simulatedIfConfig := &models.ServerNetworkConfig{
		Enabled:                     true,
		Network:                     config.Network,
		PseudoBridgeMasterInterface: nil,
		Snat: &models.SnatConfig{
			Enabled:                true,
			SnatIPNet:              target_network,
			SnatExcludedNetwork:    config.Snat.SnatExcludedNetwork,
			RoamingMasterInterface: nil,
			RoamingPseudoBridge:    false,
		},
		RoutedNetworks:         config.RoutedNetworks,
		RoutedNetworksFirewall: config.RoutedNetworksFirewall,
		CommentString:          config.CommentString,
	}

	return simulatedIfConfig, nil
}

func (l *InterfaceIPNetListener) IPaddrUpdate() {
	ipv4, ipv6, err := utils.GetInterfaceIP(l.interfaceName)
	if err != nil {
		log.Printf("Read IP from %v failed, err: %v", l.interfaceName, err)
	}
	l.ifIPs[4] = ipv4
	l.ifIPs[6] = ipv6
	l.pseudoBridgeService.UpdateBaseNets(l.interfaceName, ipv4, ipv6)
}

func (l *InterfaceIPNetListener) Stop() {
	l.stopCh <- struct{}{}
}

func getNetlinkIfName(link netlink.Link) (string, error) {
	if link == nil {
		return "", fmt.Errorf("link is nil")
	}
	attrs := link.Attrs()
	if attrs == nil {
		return "", fmt.Errorf("aattrs is nil")
	}
	return attrs.Name, nil
}

func getAddrUpdateIfName(addrUpdate netlink.AddrUpdate) (string, error) {
	// Safely get the link by index to check the name
	link, err := netlink.LinkByIndex(addrUpdate.LinkIndex)
	if err != nil {
		log.Printf("Failed to get link by index %d: %v", addrUpdate.LinkIndex, err)
		return "", err
	}
	if link == nil {
		return "", fmt.Errorf("link is nil")
	}
	attrs := link.Attrs()
	if attrs == nil {
		return "", fmt.Errorf("attrs is nil")
	}
	return attrs.Name, nil
}
