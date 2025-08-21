package services

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"wg-panel/internal/models"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type PseudoBridgeService struct {
	waitInterface    map[string]map[string][]*models.IPNetWrapper // [ifname][v4/v6][networks...]
	runningInterface map[string]map[string][]*models.IPNetWrapper
	responders       map[string]*InterfaceResponder
	mu               sync.RWMutex
	running          bool
}

type InterfaceResponder struct {
	interfaceName string
	networks      map[string][]*models.IPNetWrapper // [v4/v6][networks...]
	handle        *pcap.Handle
	stopCh        chan struct{}
	mu            sync.RWMutex
}

func NewPseudoBridgeService() *PseudoBridgeService {
	return &PseudoBridgeService{
		waitInterface:    make(map[string]map[string][]*models.IPNetWrapper),
		runningInterface: make(map[string]map[string][]*models.IPNetWrapper),
		responders:       make(map[string]*InterfaceResponder),
		running:          false,
	}
}

func (s *PseudoBridgeService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Println("Starting Pseudo-bridge Service")
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.scanAndUpdate()
		}
	}
}

func (s *PseudoBridgeService) Stop() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in PseudoBridgeService.Stop: %v", r)
		}
	}()
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.running {
		return
	}
	
	s.running = false
	
	// Stop all responders safely
	for name, responder := range s.responders {
		if responder != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic stopping responder %s: %v", name, r)
					}
				}()
				responder.Stop()
			}()
		}
	}
	s.responders = make(map[string]*InterfaceResponder)
	
	log.Println("Pseudo-bridge Service stopped")
}

func (s *PseudoBridgeService) UpdateConfiguration(interfaces map[string]*models.Interface) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear and rebuild wait interface map
	s.waitInterface = make(map[string]map[string][]*models.IPNetWrapper)

	for _, iface := range interfaces {
		for _, server := range iface.Servers {
			if !server.Enabled {
				continue
			}

			// Check IPv4 pseudo-bridge
			if server.IPv4 != nil && server.IPv4.Enabled && server.IPv4.PseudoBridgeMasterInterface != nil {
				s.addNetworkToWait(*server.IPv4.PseudoBridgeMasterInterface, "v4", server.IPv4.Network)
			}

			// Check IPv4 SNAT pseudo-bridge (NETMAP mode)
			if server.IPv4 != nil && server.IPv4.Enabled && server.IPv4.Snat != nil && 
				server.IPv4.Snat.Enabled && server.IPv4.Snat.RoamingMasterInterface != nil &&
				server.IPv4.Snat.RoamingPseudoBridge && server.IPv4.Snat.SnatIPNet != nil {
				s.addNetworkToWait(*server.IPv4.Snat.RoamingMasterInterface, "v4", server.IPv4.Snat.SnatIPNet)
			}

			// Check IPv6 pseudo-bridge
			if server.IPv6 != nil && server.IPv6.Enabled && server.IPv6.PseudoBridgeMasterInterface != nil {
				s.addNetworkToWait(*server.IPv6.PseudoBridgeMasterInterface, "v6", server.IPv6.Network)
			}

			// Check IPv6 SNAT pseudo-bridge (NETMAP mode)
			if server.IPv6 != nil && server.IPv6.Enabled && server.IPv6.Snat != nil &&
				server.IPv6.Snat.Enabled && server.IPv6.Snat.RoamingMasterInterface != nil &&
				server.IPv6.Snat.RoamingPseudoBridge && server.IPv6.Snat.SnatIPNet != nil {
				s.addNetworkToWait(*server.IPv6.Snat.RoamingMasterInterface, "v6", server.IPv6.Snat.SnatIPNet)
			}
		}
	}
}

func (s *PseudoBridgeService) addNetworkToWait(interfaceName, version string, network *models.IPNetWrapper) {
	if network == nil {
		return
	}

	if s.waitInterface[interfaceName] == nil {
		s.waitInterface[interfaceName] = make(map[string][]*models.IPNetWrapper)
	}
	
	s.waitInterface[interfaceName][version] = append(s.waitInterface[interfaceName][version], network)
}

func (s *PseudoBridgeService) scanAndUpdate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate changes in a single pass
	changes := s.calculateChanges()
	if !changes.hasChanges() {
		return // No changes needed
	}

	log.Println("Pseudo-bridge configuration changed, updating responders")

	// Apply all changes atomically
	s.applyChanges(changes)

	// Deep copy waitInterface to runningInterface
	s.syncRunningInterface()
}

type interfaceChanges struct {
	toStop   []string                                           // interfaces to stop
	toStart  map[string]map[string][]*models.IPNetWrapper     // interfaces to start
	toUpdate map[string]map[string][]*models.IPNetWrapper     // interfaces to update
}

func (c *interfaceChanges) hasChanges() bool {
	return len(c.toStop) > 0 || len(c.toStart) > 0 || len(c.toUpdate) > 0
}

func (s *PseudoBridgeService) calculateChanges() *interfaceChanges {
	changes := &interfaceChanges{
		toStop:   make([]string, 0),
		toStart:  make(map[string]map[string][]*models.IPNetWrapper),
		toUpdate: make(map[string]map[string][]*models.IPNetWrapper),
	}

	// Find interfaces to stop (present in running but not in wait)
	for ifname := range s.runningInterface {
		if _, exists := s.waitInterface[ifname]; !exists {
			changes.toStop = append(changes.toStop, ifname)
		}
	}

	// Find interfaces to start or update
	for ifname, waitNetworks := range s.waitInterface {
		runningNetworks, isRunning := s.runningInterface[ifname]
		
		if !isRunning {
			// New interface to start
			changes.toStart[ifname] = s.deepCopyNetworks(waitNetworks)
		} else if !s.networksEqual(waitNetworks, runningNetworks) {
			// Existing interface with changes
			changes.toUpdate[ifname] = s.deepCopyNetworks(waitNetworks)
		}
	}

	return changes
}

func (s *PseudoBridgeService) applyChanges(changes *interfaceChanges) {
	// Stop removed interfaces
	for _, ifname := range changes.toStop {
		if responder, exists := s.responders[ifname]; exists {
			responder.Stop()
			delete(s.responders, ifname)
			log.Printf("Stopped pseudo-bridge responder for interface %s", ifname)
		}
	}

	// Start new interfaces
	for ifname, networks := range changes.toStart {
		responder := NewInterfaceResponder(ifname, networks)
		if err := responder.Start(); err != nil {
			log.Printf("Failed to start pseudo-bridge responder for %s: %v", ifname, err)
			continue
		}
		s.responders[ifname] = responder
		log.Printf("Started pseudo-bridge responder for interface %s", ifname)
	}

	// Update existing interfaces
	for ifname, networks := range changes.toUpdate {
		if responder, exists := s.responders[ifname]; exists {
			responder.UpdateNetworks(networks)
			log.Printf("Updated pseudo-bridge responder for interface %s", ifname)
		}
	}
}

func (s *PseudoBridgeService) networksEqual(a, b map[string][]*models.IPNetWrapper) bool {
	if len(a) != len(b) {
		return false
	}

	for version, aNets := range a {
		bNets, exists := b[version]
		if !exists || len(aNets) != len(bNets) {
			return false
		}

		// Compare network strings (order-sensitive)
		for i, aNet := range aNets {
			if i >= len(bNets) || aNet.String() != bNets[i].String() {
				return false
			}
		}
	}

	return true
}

func (s *PseudoBridgeService) deepCopyNetworks(networks map[string][]*models.IPNetWrapper) map[string][]*models.IPNetWrapper {
	result := make(map[string][]*models.IPNetWrapper)
	for version, nets := range networks {
		result[version] = make([]*models.IPNetWrapper, len(nets))
		copy(result[version], nets)
	}
	return result
}

func (s *PseudoBridgeService) syncRunningInterface() {
	s.runningInterface = make(map[string]map[string][]*models.IPNetWrapper)
	for ifname, networks := range s.waitInterface {
		s.runningInterface[ifname] = s.deepCopyNetworks(networks)
	}
}

func NewInterfaceResponder(interfaceName string, networks map[string][]*models.IPNetWrapper) *InterfaceResponder {
	return &InterfaceResponder{
		interfaceName: interfaceName,
		networks:      networks,
		stopCh:        make(chan struct{}),
	}
}

func (r *InterfaceResponder) Start() error {
	// Open pcap handle for the interface
	handle, err := pcap.OpenLive(r.interfaceName, 1600, true, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("failed to open pcap handle: %v", err)
	}

	// Set BPF filter for ARP and ICMPv6
	filter := "arp or (icmp6 and ip6[40] == 135)" // ARP or Neighbor Solicitation
	if err := handle.SetBPFFilter(filter); err != nil {
		handle.Close()
		return fmt.Errorf("failed to set BPF filter: %v", err)
	}

	r.handle = handle
	go r.packetLoop()
	
	return nil
}

func (r *InterfaceResponder) Stop() {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("Recovered from panic in InterfaceResponder.Stop: %v", rec)
		}
	}()
	
	// Safely close stop channel
	select {
	case <-r.stopCh:
		// Already closed
	default:
		close(r.stopCh)
	}
	
	// Safely close pcap handle
	if r.handle != nil {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("Recovered from panic closing pcap handle: %v", rec)
				}
			}()
			r.handle.Close()
		}()
	}
}

func (r *InterfaceResponder) UpdateNetworks(networks map[string][]*models.IPNetWrapper) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.networks = networks
}

func (r *InterfaceResponder) packetLoop() {
	packetSource := gopacket.NewPacketSource(r.handle, r.handle.LinkType())
	
	for {
		select {
		case <-r.stopCh:
			return
		case packet := <-packetSource.Packets():
			if packet == nil {
				continue
			}
			r.handlePacket(packet)
		}
	}
}

func (r *InterfaceResponder) handlePacket(packet gopacket.Packet) {
	// Handle ARP requests
	if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
		arp := arpLayer.(*layers.ARP)
		if arp.Operation == layers.ARPRequest {
			r.handleARPRequest(packet, arp)
		}
	}

	// Handle IPv6 Neighbor Solicitation
	if icmp6Layer := packet.Layer(layers.LayerTypeICMPv6); icmp6Layer != nil {
		icmp6 := icmp6Layer.(*layers.ICMPv6)
		if icmp6.TypeCode.Type() == layers.ICMPv6TypeNeighborSolicitation {
			r.handleNeighborSolicitation(packet, icmp6)
		}
	}
}

func (r *InterfaceResponder) handleARPRequest(packet gopacket.Packet, arp *layers.ARP) {
	targetIP := net.IP(arp.DstProtAddress)
	
	r.mu.RLock()
	networks := r.networks["v4"]
	r.mu.RUnlock()

	// Check if target IP is in any of our managed networks
	for _, network := range networks {
		if network != nil && network.Contains(targetIP) {
			r.sendARPReply(packet, arp)
			return
		}
	}
}

func (r *InterfaceResponder) handleNeighborSolicitation(packet gopacket.Packet, icmp6 *layers.ICMPv6) {
	// Parse the target address from the NS packet
	if len(icmp6.Payload) < 20 {
		return
	}
	
	targetIP := net.IP(icmp6.Payload[4:20])
	
	r.mu.RLock()
	networks := r.networks["v6"]
	r.mu.RUnlock()

	// Check if target IP is in any of our managed networks
	for _, network := range networks {
		if network != nil && network.Contains(targetIP) {
			r.sendNeighborAdvertisement(packet, targetIP)
			return
		}
	}
}

func (r *InterfaceResponder) sendARPReply(requestPacket gopacket.Packet, requestARP *layers.ARP) {
	// Get the interface's MAC address
	iface, err := net.InterfaceByName(r.interfaceName)
	if err != nil {
		return
	}

	// Extract source information from request
	ethLayer := requestPacket.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return
	}
	srcEth := ethLayer.(*layers.Ethernet)

	// Build ARP reply
	eth := &layers.Ethernet{
		SrcMAC:       iface.HardwareAddr,
		DstMAC:       srcEth.SrcMAC,
		EthernetType: layers.EthernetTypeARP,
	}

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPReply,
		SourceHwAddress:   iface.HardwareAddr,
		SourceProtAddress: requestARP.DstProtAddress,
		DstHwAddress:      requestARP.SourceHwAddress,
		DstProtAddress:    requestARP.SourceProtAddress,
	}

	// Serialize and send
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	gopacket.SerializeLayers(buffer, opts, eth, arp)
	
	r.handle.WritePacketData(buffer.Bytes())
}

func (r *InterfaceResponder) sendNeighborAdvertisement(requestPacket gopacket.Packet, targetIP net.IP) {
	// Get the interface's MAC address
	iface, err := net.InterfaceByName(r.interfaceName)
	if err != nil {
		return
	}

	// Extract source information from request
	ethLayer := requestPacket.Layer(layers.LayerTypeEthernet)
	ipv6Layer := requestPacket.Layer(layers.LayerTypeIPv6)
	if ethLayer == nil || ipv6Layer == nil {
		return
	}
	
	srcEth := ethLayer.(*layers.Ethernet)
	srcIPv6 := ipv6Layer.(*layers.IPv6)

	// Build Neighbor Advertisement
	eth := &layers.Ethernet{
		SrcMAC:       iface.HardwareAddr,
		DstMAC:       srcEth.SrcMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	ipv6 := &layers.IPv6{
		Version:    6,
		NextHeader: layers.IPProtocolICMPv6,
		HopLimit:   255,
		SrcIP:      targetIP,
		DstIP:      srcIPv6.SrcIP,
	}

	// Build ICMPv6 Neighbor Advertisement
	icmp6 := &layers.ICMPv6{
		TypeCode: layers.CreateICMPv6TypeCode(layers.ICMPv6TypeNeighborAdvertisement, 0),
	}

	// NA payload: flags (4 bytes) + target address (16 bytes) + TLV options
	naPayload := make([]byte, 24) // 4 + 16 + 8 (for TLV option)
	
	// Flags: Solicited flag set
	naPayload[0] = 0x40
	
	// Target address
	copy(naPayload[4:20], targetIP)
	
	// Target Link-layer Address option (Type=2, Length=1, MAC)
	naPayload[20] = 2 // Type
	naPayload[21] = 1 // Length (in units of 8 bytes)
	copy(naPayload[22:28], iface.HardwareAddr)

	icmp6.Payload = naPayload

	// Serialize and send
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	gopacket.SerializeLayers(buffer, opts, eth, ipv6, icmp6)
	
	r.handle.WritePacketData(buffer.Bytes())
}