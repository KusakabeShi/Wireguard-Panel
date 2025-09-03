package internalservice

import (
	"fmt"
	"sync"
	"time"

	"wg-panel/internal/logging"
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
	for ifname := range delIF {
		s.listeners[ifname].Stop()
		delete(s.runningInterface, ifname)
		delete(s.listeners, ifname)
	}
	for ifname, configs := range addIF {
		s.runningInterface[ifname] = &SNATRoamingSynced{
			ExistsInNewConfig: false,
		}
		s.listeners[ifname] = NewInterfaceIPNetListener(ifname, s.pseudoBridgeService, s.fw)
		s.listeners[ifname].SyncIpFromIface()
		s.listeners[ifname].UpdateConfigsAndSyncFw(configs, true)
	}
	for ifname, configs := range updateIF {
		s.listeners[ifname].SyncIpFromIface()
		s.listeners[ifname].UpdateConfigsAndSyncFw(configs, true)
	}

}

func (s *SNATRoamingService) mainLoop() {
	linkUpdatesSubscribed := false
	addrUpdatesSubscribed := false
	linkUpdates := make(chan netlink.LinkUpdate, 10)
	addrUpdates := make(chan netlink.AddrUpdate, 10)
	defer func() {
		logging.LogInfo("SNAT Roaming Service stopped")
		close(s.stopCh)
	}()
	logging.LogInfo("SNAT Roaming Service starting")
	for {
		// Try to open pcap handle for the interface
		if !linkUpdatesSubscribed {
			// Subscribe to link updates for interface up/down events
			if err := netlink.LinkSubscribe(linkUpdates, s.stopCh); err != nil {
				logging.LogError("Failed to subscribe to link updates: %v, retrying in 5 seconds", err)
				time.Sleep(5 * time.Second)
				continue
			}
			linkUpdatesSubscribed = true
		}
		if !addrUpdatesSubscribed {
			// Subscribe to address updates for IP address changes
			if err := netlink.AddrSubscribe(addrUpdates, s.stopCh); err != nil {
				logging.LogError("Failed to subscribe to address updates: %v, retrying in 5 seconds", err)
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
					logging.LogError("Link updates channel closed, attempting to resubscribe in 5 seconds")
					linkUpdatesSubscribed = false
					continue
				}
				ifname, err := getNetlinkIfName(linkUpdate)
				if err != nil {
					logging.LogError("Failed to parse ifname from linkUpdate: %v", err)
				}
				logging.LogVerbose("Link update detected for interface: %s", ifname)
				s.mu.RLock()
				listener, ok := s.listeners[ifname]
				s.mu.RUnlock()
				if ok {
					logging.LogVerbose("Syncing IP addresses for interface: %s", ifname)
					ipchanged := listener.SyncIpFromIface()
					if ipchanged {
						logging.LogInfo("IP change detected for interface: %s, updating firewall rules", ifname)
						listener.UpdateConfigsAndSyncFw(listener.configs, true)
					}
				} else {
					logging.LogVerbose("No SNAT roaming listener found for interface: %s, ignoring", ifname)
				}

			case addrUpdate, ok := <-addrUpdates:
				if !ok {
					logging.LogError("Address updates channel closed, attempting to resubscribe in 5 seconds")
					addrUpdatesSubscribed = false
					continue
				}
				ifname, err := getAddrUpdateIfName(addrUpdate)
				if err != nil {
					logging.LogError("Failed to parse ifname from addrUpdate: %v", err)
				}
				logging.LogVerbose("Address change detected for interface: %s", ifname)
				s.mu.RLock()
				listener, ok := s.listeners[ifname]
				s.mu.RUnlock()
				if ok {
					logging.LogVerbose("Syncing IP addresses for interface: %s after address change", ifname)
					ipchanged := listener.SyncIpFromIface()
					if ipchanged {
						logging.LogInfo("IP change detected for interface: %s, updating firewall rules", ifname)
						listener.UpdateConfigsAndSyncFw(listener.configs, true)
					}
				} else {
					logging.LogVerbose("No SNAT roaming listener found for interface: %s", ifname)
				}
			}
		}
	}
}

func (l *InterfaceIPNetListener) mainLoop() {
	logging.LogInfo("Interface IPNet Listener for %v started", l.interfaceName)
	defer func() {
		close(l.stopCh)
		logging.LogInfo("Interface IPNet Listener for %v stopped", l.interfaceName)
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

	logging.LogInfo("SNAT Roaming Service stopped")
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

	go r.mainLoop()
	return r
}

func (l *InterfaceIPNetListener) UpdateConfigsAndSyncFw(configs map[string]*models.ServerNetworkConfig, forceUpdateAll bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if configs == nil {
		return
	}
	oldsynced := make(map[string]bool)
	for k := range l.configs {
		oldsynced[k] = false
	}

	toAdd := make(map[string]*models.ServerNetworkConfig)
	toUpdate := make(map[string]*models.ServerNetworkConfig)
	toDel := make(map[string]*models.ServerNetworkConfig)

	for key, newconf := range configs {
		if newconf == nil || newconf.Network == nil || newconf.CommentString == "" || !newconf.Enabled || newconf.Snat == nil ||
			newconf.Snat.RoamingMasterInterface == nil || len(*newconf.Snat.RoamingMasterInterface) == 0 {
			continue
		}

		oldconf, ok := l.configs[key]
		if ok {
			oldsynced[key] = true
			if forceUpdateAll ||
				!oldconf.Network.Equal(newconf.Network) || !oldconf.Snat.SnatIPNet.Equal(newconf.Snat.SnatIPNet) ||
				!oldconf.Snat.SnatExcludedNetwork.Equal(newconf.Snat.SnatExcludedNetwork) ||
				!models.NetworksEqualNP(oldconf.RoutedNetworks, newconf.RoutedNetworks) {
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
	for key, config := range toDel {
		logging.LogVerbose("Removing SNAT roaming rules for %s on interface %s", key, l.interfaceName)
		l.fw.RemoveSnatRules(config.Network.Version, config.CommentString)
		logging.LogVerbose("Successfully removed SNAT roaming rules for %s on interface %s", key, l.interfaceName)
	}
	for key, config := range toAdd {
		logging.LogVerbose("Adding SNAT roaming rules for %s on interface %s", key, l.interfaceName)
		simulatedConfig, err := l.getSimulatedConfig(config)
		if err != nil {
			logging.LogError("Failed to calculate target_network: %v", err)
		}
		if err := l.fw.AddSnatRules(simulatedConfig, config.CommentString); err != nil {
			logging.LogError("Failed to add firewall rules: %v", err)
		} else {
			logging.LogVerbose("Successfully added SNAT roaming rules for %s on interface %s", key, l.interfaceName)
		}
	}
	for key, config := range toUpdate {
		logging.LogVerbose("Updating SNAT roaming rules for %s on interface %s", key, l.interfaceName)
		simulatedConfig, err := l.getSimulatedConfig(config)
		if err != nil {
			logging.LogError("Failed to calculate target_network: %v", err)
		}
		l.fw.RemoveSnatRules(config.Network.Version, config.CommentString)
		if err := l.fw.AddSnatRules(simulatedConfig, config.CommentString); err != nil {
			logging.LogError("Failed to add firewall rules: %v", err)
		} else {
			logging.LogVerbose("Successfully updated SNAT roaming rules for %s on interface %s", key, l.interfaceName)
		}
	}

	l.configs = configs
}

func (l *InterfaceIPNetListener) getSimulatedConfig(config *models.ServerNetworkConfig) (*models.ServerNetworkConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	if config.Network == nil || config.Snat == nil || config.Snat.SnatIPNet == nil {
		return nil, fmt.Errorf("config.Network or config.Snat or config.Snat.SnatIPNet is nil")
	}

	_, ok := l.ifIPs[config.Network.Version]
	if !ok || l.ifIPs[config.Network.Version] == nil {
		return nil, fmt.Errorf("interface %v has no IP for version %v", l.interfaceName, config.Network.Version)
	}

	target_network, err := l.ifIPs[config.Network.Version].GetNetByOffset(config.Snat.SnatIPNet)

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

func (l *InterfaceIPNetListener) SyncIpFromIface() (changed bool) {
	ipv4, ipv6, err := utils.GetInterfaceIP(l.interfaceName)
	if err != nil {
		logging.LogError("Read IP from %v failed, err: %v", l.interfaceName, err)
	} else {
		logging.LogVerbose("Synced primary IPs for %s: IPv4=%v, IPv6=%v", l.interfaceName, ipv4, ipv6)
	}
	if !l.ifIPs[4].Equal(ipv4) || !l.ifIPs[6].Equal(ipv6) {
		l.ifIPs[4] = ipv4
		l.ifIPs[6] = ipv6
		changed = true
	}
	ipv4s, ipv6s, err := utils.GetInterfaceIPs(l.interfaceName)
	if err != nil {
		logging.LogError("Failed to get all bound IPs for %v: %v", l.interfaceName, err)
	} else {
		logging.LogVerbose("Synced all IPs for %s: %d IPv4 addresses, %d IPv6 addresses", l.interfaceName, len(ipv4s), len(ipv6s))
	}
	l.pseudoBridgeService.UpdateIfaceBindInfo(l.interfaceName, ipv4, ipv6, ipv4s, ipv6s)
	return changed
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
		logging.LogError("Failed to get link by index %d: %v", addrUpdate.LinkIndex, err)
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
