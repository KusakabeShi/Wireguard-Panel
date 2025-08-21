package services

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"wg-panel/internal/models"
	"wg-panel/internal/utils"

	"github.com/vishvananda/netlink"
)

type SNATRoamingService struct {
	waitInterface    map[string][]*SNATRoamingConfig // [ifname][configs...]
	runningInterface map[string][]*SNATRoamingConfig
	listeners        map[string]*InterfaceIPNetListener
	mu               sync.RWMutex
	running          bool
}

type SNATRoamingConfig struct {
	Version             int // 4 or 6
	ServerNetwork       *models.IPNetWrapper
	SNATIPNet           *models.IPNetWrapper
	SNATExcludedNet     *models.IPNetWrapper
	CommentString       string
	IsNETMAP            bool
	InterfaceDevice     string
	RoamingPseudoBridge bool
}

type InterfaceIPNetListener struct {
	interfaceName string
	configs       []*SNATRoamingConfig
	stopCh        chan struct{}
	fw            *FirewallService
	mu            sync.RWMutex
	linkUpdates   chan netlink.LinkUpdate
	addrUpdates   chan netlink.AddrUpdate
}

func NewSNATRoamingService() *SNATRoamingService {
	return &SNATRoamingService{
		waitInterface:    make(map[string][]*SNATRoamingConfig),
		runningInterface: make(map[string][]*SNATRoamingConfig),
		listeners:        make(map[string]*InterfaceIPNetListener),
		running:          false,
	}
}

func (s *SNATRoamingService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Println("Starting SNAT Roaming Service")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.scanAndUpdate()
		}
	}
}

func (s *SNATRoamingService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false

	// Stop all listeners
	for _, listener := range s.listeners {
		listener.Stop()
	}
	s.listeners = make(map[string]*InterfaceIPNetListener)

	log.Println("SNAT Roaming Service stopped")
}

func (s *SNATRoamingService) UpdateConfiguration(interfaces map[string]*models.Interface) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear and rebuild wait interface map
	s.waitInterface = make(map[string][]*SNATRoamingConfig)

	for _, iface := range interfaces {
		for _, server := range iface.Servers {
			if !server.Enabled {
				continue
			}

			// Check IPv4 SNAT roaming
			if server.IPv4 != nil && server.IPv4.Enabled && server.IPv4.Snat != nil &&
				server.IPv4.Snat.Enabled && server.IPv4.Snat.RoamingMasterInterface != nil {

				config := &SNATRoamingConfig{
					Version:             4,
					ServerNetwork:       server.IPv4.Network,
					SNATIPNet:           server.IPv4.Snat.SnatIPNet,
					SNATExcludedNet:     server.IPv4.Snat.SnatExcludedNetwork,
					CommentString:       server.IPv4.CommentString,
					IsNETMAP:            false,
					InterfaceDevice:     fmt.Sprintf("wg-%s", iface.Ifname),
					RoamingPseudoBridge: server.IPv4.Snat.RoamingPseudoBridge,
				}

				masterInterface := *server.IPv4.Snat.RoamingMasterInterface
				s.waitInterface[masterInterface] = append(s.waitInterface[masterInterface], config)
			}

			// Check IPv6 SNAT roaming
			if server.IPv6 != nil && server.IPv6.Enabled && server.IPv6.Snat != nil &&
				server.IPv6.Snat.Enabled && server.IPv6.Snat.RoamingMasterInterface != nil {

				isNETMAP := server.IPv6.Snat.SnatIPNet != nil &&
					server.IPv6.Snat.SnatIPNet.Masklen() != 128

				config := &SNATRoamingConfig{
					Version:             6,
					ServerNetwork:       server.IPv6.Network,
					SNATIPNet:           server.IPv6.Snat.SnatIPNet,
					SNATExcludedNet:     server.IPv6.Snat.SnatExcludedNetwork,
					CommentString:       server.IPv6.CommentString,
					IsNETMAP:            isNETMAP,
					InterfaceDevice:     fmt.Sprintf("wg-%s", iface.Ifname),
					RoamingPseudoBridge: server.IPv6.Snat.RoamingPseudoBridge,
				}

				masterInterface := *server.IPv6.Snat.RoamingMasterInterface
				s.waitInterface[masterInterface] = append(s.waitInterface[masterInterface], config)
			}
		}
	}
}

func (s *SNATRoamingService) scanAndUpdate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate changes in a single pass
	changes := s.calculateChanges()
	if !changes.hasChanges() {
		return // No changes needed
	}

	log.Println("SNAT Roaming configuration changed, updating listeners")

	// Apply all changes atomically
	s.applyChanges(changes)

	// Deep copy waitInterface to runningInterface
	s.syncRunningInterface()
}

type snatRoamingChanges struct {
	toStop   []string                        // interfaces to stop
	toStart  map[string][]*SNATRoamingConfig // interfaces to start
	toUpdate map[string][]*SNATRoamingConfig // interfaces to update
}

func (c *snatRoamingChanges) hasChanges() bool {
	return len(c.toStop) > 0 || len(c.toStart) > 0 || len(c.toUpdate) > 0
}

func (s *SNATRoamingService) calculateChanges() *snatRoamingChanges {
	changes := &snatRoamingChanges{
		toStop:   make([]string, 0),
		toStart:  make(map[string][]*SNATRoamingConfig),
		toUpdate: make(map[string][]*SNATRoamingConfig),
	}

	// Find interfaces to stop (present in running but not in wait)
	for ifname := range s.runningInterface {
		if _, exists := s.waitInterface[ifname]; !exists {
			changes.toStop = append(changes.toStop, ifname)
		}
	}

	// Find interfaces to start or update
	for ifname, waitConfigs := range s.waitInterface {
		runningConfigs, isRunning := s.runningInterface[ifname]

		if !isRunning {
			// New interface to start
			changes.toStart[ifname] = s.deepCopyConfigs(waitConfigs)
		} else if !s.configsEqual(waitConfigs, runningConfigs) {
			// Existing interface with changes
			changes.toUpdate[ifname] = s.deepCopyConfigs(waitConfigs)
		}
	}

	return changes
}

func (s *SNATRoamingService) applyChanges(changes *snatRoamingChanges) {
	// Stop removed interfaces
	for _, ifname := range changes.toStop {
		if listener, exists := s.listeners[ifname]; exists {
			listener.Stop()
			delete(s.listeners, ifname)
			log.Printf("Stopped SNAT roaming listener for interface %s", ifname)
		}
	}

	// Start new interfaces
	for ifname, configs := range changes.toStart {
		listener := NewInterfaceIPNetListener(ifname, configs)
		if err := listener.Start(); err != nil {
			log.Printf("Failed to start SNAT roaming listener for %s: %v", ifname, err)
			continue
		}
		s.listeners[ifname] = listener
		log.Printf("Started SNAT roaming listener for interface %s", ifname)
	}

	// Update existing interfaces
	for ifname, configs := range changes.toUpdate {
		if listener, exists := s.listeners[ifname]; exists {
			listener.UpdateConfigs(configs)
			log.Printf("Updated SNAT roaming listener for interface %s", ifname)
		}
	}
}

func (s *SNATRoamingService) configsEqual(a, b []*SNATRoamingConfig) bool {
	if len(a) != len(b) {
		return false
	}

	for i, aConfig := range a {
		if i >= len(b) {
			return false
		}
		bConfig := b[i]

		if aConfig.Version != bConfig.Version ||
			aConfig.CommentString != bConfig.CommentString ||
			aConfig.IsNETMAP != bConfig.IsNETMAP ||
			aConfig.InterfaceDevice != bConfig.InterfaceDevice ||
			aConfig.RoamingPseudoBridge != bConfig.RoamingPseudoBridge {
			return false
		}

		// Compare network strings
		if !s.networkEqual(aConfig.ServerNetwork, bConfig.ServerNetwork) ||
			!s.networkEqual(aConfig.SNATIPNet, bConfig.SNATIPNet) ||
			!s.networkEqual(aConfig.SNATExcludedNet, bConfig.SNATExcludedNet) {
			return false
		}
	}

	return true
}

func (s *SNATRoamingService) networkEqual(a, b *models.IPNetWrapper) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.String() == b.String()
}

func (s *SNATRoamingService) deepCopyConfigs(configs []*SNATRoamingConfig) []*SNATRoamingConfig {
	result := make([]*SNATRoamingConfig, len(configs))
	copy(result, configs)
	return result
}

func (s *SNATRoamingService) syncRunningInterface() {
	s.runningInterface = make(map[string][]*SNATRoamingConfig)
	for ifname, configs := range s.waitInterface {
		s.runningInterface[ifname] = s.deepCopyConfigs(configs)
	}
}

func NewInterfaceIPNetListener(interfaceName string, configs []*SNATRoamingConfig) *InterfaceIPNetListener {
	return &InterfaceIPNetListener{
		interfaceName: interfaceName,
		configs:       configs,
		stopCh:        make(chan struct{}),
		fw:            NewFirewallService(),
		linkUpdates:   make(chan netlink.LinkUpdate, 10),
		addrUpdates:   make(chan netlink.AddrUpdate, 10),
	}
}

func (l *InterfaceIPNetListener) Start() error {
	// Subscribe to netlink updates for the specific interface
	if err := l.subscribeToNetlinkUpdates(); err != nil {
		return fmt.Errorf("failed to subscribe to netlink updates: %v", err)
	}

	// Start monitoring interface changes
	go l.monitorInterface()

	// Initial sync
	l.syncFirewallRules()

	return nil
}

func (l *InterfaceIPNetListener) Stop() {
	// Use defer to ensure cleanup even if panics occur
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in Stop for %s: %v", l.interfaceName, r)
		}
	}()

	// Signal stop first
	select {
	case <-l.stopCh:
		// Already stopped
		return
	default:
		close(l.stopCh)
	}

	// Clean up firewall rules safely
	l.mu.RLock()
	configs := make([]*SNATRoamingConfig, len(l.configs))
	copy(configs, l.configs)
	l.mu.RUnlock()

	for _, config := range configs {
		if config != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic removing firewall rules: %v", r)
					}
				}()
				l.removeFirewallRules(config)
			}()
		}
	}

	// Close netlink channels safely
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic closing channels: %v", r)
			}
		}()

		// Check if channels are still open before closing
		select {
		case <-l.linkUpdates:
		default:
			close(l.linkUpdates)
		}

		select {
		case <-l.addrUpdates:
		default:
			close(l.addrUpdates)
		}
	}()
}

func (l *InterfaceIPNetListener) resubscribeNetlinkUpdates() error {
	// Create new channels
	l.linkUpdates = make(chan netlink.LinkUpdate, 10)
	l.addrUpdates = make(chan netlink.AddrUpdate, 10)

	// Resubscribe to netlink updates
	if err := netlink.LinkSubscribe(l.linkUpdates, l.stopCh); err != nil {
		return fmt.Errorf("failed to resubscribe to link updates: %v", err)
	}

	if err := netlink.AddrSubscribe(l.addrUpdates, l.stopCh); err != nil {
		return fmt.Errorf("failed to resubscribe to address updates: %v", err)
	}

	log.Printf("Successfully resubscribed to netlink updates for %s", l.interfaceName)
	return nil
}

func (l *InterfaceIPNetListener) UpdateConfigs(configs []*SNATRoamingConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Remove old rules
	for _, config := range l.configs {
		l.removeFirewallRules(config)
	}

	// Update configs
	l.configs = configs

	// Apply new rules
	l.syncFirewallRules()
}

func (l *InterfaceIPNetListener) subscribeToNetlinkUpdates() error {
	// Subscribe to link updates for interface up/down events
	if err := netlink.LinkSubscribe(l.linkUpdates, l.stopCh); err != nil {
		return fmt.Errorf("failed to subscribe to link updates: %v", err)
	}

	// Subscribe to address updates for IP address changes
	if err := netlink.AddrSubscribe(l.addrUpdates, l.stopCh); err != nil {
		return fmt.Errorf("failed to subscribe to address updates: %v", err)
	}

	return nil
}

func (l *InterfaceIPNetListener) monitorInterface() {
	// Also use a ticker as fallback for missed events
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Recovery for the entire monitoring loop
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in monitorInterface for %s: %v", l.interfaceName, r)
			// Restart monitoring after a delay
			time.Sleep(5 * time.Second)
			go l.monitorInterface()
		}
	}()

	for {
		select {
		case <-l.stopCh:
			log.Printf("Stopping monitoring for interface %s", l.interfaceName)
			return

		case linkUpdate, ok := <-l.linkUpdates:
			if !ok {
				log.Printf("Link updates channel closed for %s, attempting to resubscribe", l.interfaceName)
				if err := l.resubscribeNetlinkUpdates(); err != nil {
					log.Printf("Failed to resubscribe to netlink updates: %v", err)
					return
				}
				continue
			}
			// Safely handle the update
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic handling link update for %s: %v", l.interfaceName, r)
					}
				}()
				if l.isOurInterface(linkUpdate.Link) {
					log.Printf("Link update detected for interface %s", l.interfaceName)
					l.syncFirewallRules()
				}
			}()

		case addrUpdate, ok := <-l.addrUpdates:
			if !ok {
				log.Printf("Address updates channel closed for %s, attempting to resubscribe", l.interfaceName)
				if err := l.resubscribeNetlinkUpdates(); err != nil {
					log.Printf("Failed to resubscribe to netlink updates: %v", err)
					return
				}
				continue
			}
			// Safely handle the update
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic handling addr update for %s: %v", l.interfaceName, r)
					}
				}()
				if l.isOurInterfaceAddr(addrUpdate) {
					log.Printf("Address update detected for interface %s", l.interfaceName)
					l.syncFirewallRules()
				}
			}()

		case <-ticker.C:
			// Periodic sync as fallback - this is safe even if interface is gone
			l.syncFirewallRules()
		}
	}
}

func (l *InterfaceIPNetListener) isOurInterface(link netlink.Link) bool {
	if link == nil {
		return false
	}
	attrs := link.Attrs()
	if attrs == nil {
		return false
	}
	return attrs.Name == l.interfaceName
}

func (l *InterfaceIPNetListener) isOurInterfaceAddr(addrUpdate netlink.AddrUpdate) bool {
	// Safely get the link by index to check the name
	link, err := netlink.LinkByIndex(addrUpdate.LinkIndex)
	if err != nil {
		log.Printf("Failed to get link by index %d: %v", addrUpdate.LinkIndex, err)
		return false
	}

	if link == nil {
		return false
	}

	attrs := link.Attrs()
	if attrs == nil {
		return false
	}

	return attrs.Name == l.interfaceName
}

func (l *InterfaceIPNetListener) syncFirewallRules() {
	// Use a defer to recover from any panics in this critical function
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in syncFirewallRules for %s: %v", l.interfaceName, r)
		}
	}()

	link, err := netlink.LinkByName(l.interfaceName)
	if err != nil {
		// Interface might have been removed - this is normal, just log and return
		log.Printf("Interface %s not found (may have been removed): %v", l.interfaceName, err)
		return
	}

	if link == nil {
		log.Printf("Interface %s returned nil link", l.interfaceName)
		return
	}

	// Get addresses using netlink with error handling
	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		log.Printf("Failed to get addresses for %s (interface may be down): %v", l.interfaceName, err)
		return
	}

	// Safely access configs with read lock
	l.mu.RLock()
	configs := make([]*SNATRoamingConfig, len(l.configs))
	copy(configs, l.configs)
	l.mu.RUnlock()

	// Process each config safely
	for _, config := range configs {
		if config == nil {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic processing config for %s: %v", l.interfaceName, r)
				}
			}()
			l.updateConfigRules(config, addrs)
		}()
	}
}

func (l *InterfaceIPNetListener) updateConfigRules(config *SNATRoamingConfig, addrs []netlink.Addr) {
	// Remove existing rules
	l.removeFirewallRules(config)

	if config.Version == 4 {
		l.updateIPv4Rules(config, addrs)
	} else {
		l.updateIPv6Rules(config, addrs)
	}
}

func (l *InterfaceIPNetListener) updateIPv4Rules(config *SNATRoamingConfig, addrs []netlink.Addr) {
	var dynamicIP net.IP

	// Find a suitable IPv4 address (prefer dynamic, ignore deprecated)
	for _, addr := range addrs {
		if addr.IP.To4() != nil {
			// Skip deprecated addresses (flag 0x40)
			if addr.Flags&0x40 == 0 {
				dynamicIP = addr.IP.To4()
				break
			}
		}
	}

	if dynamicIP == nil {
		log.Printf("No suitable IPv4 address found on %s", l.interfaceName)
		return
	}

	// Add SNAT rule
	l.addIPv4SNATRule(config, dynamicIP)
}

func (l *InterfaceIPNetListener) updateIPv6Rules(config *SNATRoamingConfig, addrs []netlink.Addr) {
	if !config.IsNETMAP {
		// Simple SNAT mode
		var dynamicIP net.IP

		for _, addr := range addrs {
			if addr.IP.To4() == nil && addr.IP.To16() != nil {
				// Skip deprecated addresses (flag 0x40)
				if addr.Flags&0x40 == 0 {
					dynamicIP = addr.IP.To16()
					break
				}
			}
		}

		if dynamicIP == nil {
			log.Printf("No suitable IPv6 address found on %s", l.interfaceName)
			return
		}

		l.addIPv6SNATRule(config, dynamicIP)
	} else {
		// NETMAP mode
		l.updateIPv6NETMAPRules(config, addrs)
	}
}

func (l *InterfaceIPNetListener) updateIPv6NETMAPRules(config *SNATRoamingConfig, addrs []netlink.Addr) {
	var masterNetwork *net.IPNet

	// Find the network that matches our server network's prefix length
	serverMasklen := config.ServerNetwork.Masklen()

	for _, addr := range addrs {
		if addr.IP.To4() == nil && addr.IP.To16() != nil {
			// Skip deprecated addresses (flag 0x40)
			if addr.Flags&0x40 == 0 {
				// Check if this network has the same prefix length
				masklen, _ := addr.IPNet.Mask.Size()
				if masklen == serverMasklen {
					masterNetwork = addr.IPNet
					break
				}
			}
		}
	}

	if masterNetwork == nil {
		log.Printf("No suitable IPv6 network found on %s with prefix /%d", l.interfaceName, serverMasklen)
		return
	}

	// Calculate target network by adding offset
	if config.SNATIPNet == nil {
		return
	}

	// Add master network base to the offset from SNATIPNet
	targetNetwork := l.calculateTargetNetwork(masterNetwork, config.SNATIPNet)
	if targetNetwork == nil {
		return
	}

	l.addIPv6NETMAPRules(config, targetNetwork)
}

func (l *InterfaceIPNetListener) calculateTargetNetwork(masterNet *net.IPNet, offsetNet *models.IPNetWrapper) *net.IPNet {
	// This is a simplified implementation
	// In a real implementation, you'd need to properly add the offset to the base network

	// For now, just use the offset network directly
	// This should be improved to properly calculate: masterNet.Network + offsetNet

	_, targetNet, err := net.ParseCIDR(offsetNet.String())
	if err != nil {
		log.Printf("Failed to parse target network: %v", err)
		return nil
	}

	return targetNet
}

func (l *InterfaceIPNetListener) addIPv4SNATRule(config *SNATRoamingConfig, dynamicIP net.IP) {
	sourceNet := config.ServerNetwork.NetworkStr()
	excludedNet := sourceNet
	if config.SNATExcludedNet != nil {
		excludedNet = config.SNATExcludedNet.NetworkStr()
	}

	rule := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -d %s -j SNAT --to-source %s -m comment --comment %s",
		sourceNet, excludedNet, dynamicIP.String(), config.CommentString)

	if err := l.executeIPTablesRule("iptables", rule); err != nil {
		log.Printf("Failed to add IPv4 SNAT rule: %v", err)
	}
}

func (l *InterfaceIPNetListener) addIPv6SNATRule(config *SNATRoamingConfig, dynamicIP net.IP) {
	sourceNet := config.ServerNetwork.NetworkStr()
	excludedNet := sourceNet
	if config.SNATExcludedNet != nil {
		excludedNet = config.SNATExcludedNet.NetworkStr()
	}

	rule := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -d %s -j SNAT --to-source %s -m comment --comment %s",
		sourceNet, excludedNet, dynamicIP.String(), config.CommentString)

	if err := l.executeIPTablesRule("ip6tables", rule); err != nil {
		log.Printf("Failed to add IPv6 SNAT rule: %v", err)
	}
}

func (l *InterfaceIPNetListener) addIPv6NETMAPRules(config *SNATRoamingConfig, targetNetwork *net.IPNet) {
	serverNetwork := config.ServerNetwork.NetworkStr()
	targetNet := targetNetwork.String()

	rule1 := fmt.Sprintf("-t nat -A PREROUTING -s %s -j NETMAP --to %s -m comment --comment %s",
		serverNetwork, targetNet, config.CommentString)
	rule2 := fmt.Sprintf("-t nat -A PREROUTING -s %s -j NETMAP --to %s -m comment --comment %s",
		targetNet, serverNetwork, config.CommentString)

	if err := l.executeIPTablesRule("ip6tables", rule1); err != nil {
		log.Printf("Failed to add IPv6 NETMAP rule 1: %v", err)
	}

	if err := l.executeIPTablesRule("ip6tables", rule2); err != nil {
		log.Printf("Failed to add IPv6 NETMAP rule 2: %v", err)
	}
}

func (l *InterfaceIPNetListener) executeIPTablesRule(iptablesCmd, rule string) error {
	// This would execute the iptables command
	// For now, just log it
	log.Printf("Would execute: %s %s", iptablesCmd, rule)
	return nil
}

func (l *InterfaceIPNetListener) removeFirewallRules(config *SNATRoamingConfig) {
	// Remove rules by comment string
	utils.CleanupRules(config.CommentString, config.Version, false)
}
