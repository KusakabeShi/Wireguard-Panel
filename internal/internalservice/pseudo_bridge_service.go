package internalservice

import (
	"fmt"
	"net"
	"sync"
	"time"
	"wg-panel/internal/logging"
	"wg-panel/internal/models"
	"wg-panel/internal/utils"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type PseudoBridgeService struct {
	runningInterface map[string]*IPNetSynced
	responders       map[string]*InterfaceResponder
	mu               sync.RWMutex
}

type InterfaceResponder struct {
	interfaceName   string
	networks        ResponderNetworks // [v4/v6][networks...]
	workingNetworks ResponderNetworks
	workingSkipIPv4 []net.IP
	workingSkipIPv6 []net.IP
	ipv4Base        *models.IPNetWrapper
	ipv6Base        *models.IPNetWrapper
	bindedIPv4s     []net.IP
	bindedIPv6s     []net.IP
	handle          *pcap.Handle
	stopCh          chan struct{}
	mu              sync.RWMutex
}

type ResponderNetworks struct {
	V4Networks []*models.IPNetWrapper
	V6Networks []*models.IPNetWrapper
	V4Offsets  []*models.IPNetWrapper
	V6Offsets  []*models.IPNetWrapper
	V4Skipped  []net.IP
	V6Skipped  []net.IP
}

type IPNetSynced struct {
	IPNet             ResponderNetworks
	ExistsInNewConfig bool
}

func NewPseudoBridgeService() *PseudoBridgeService {
	return &PseudoBridgeService{
		runningInterface: make(map[string]*IPNetSynced),
		responders:       make(map[string]*InterfaceResponder),
	}
}

func (s *PseudoBridgeService) UpdateConfiguration(waitInterface map[string]ResponderNetworks) {
	logging.LogVerbose("Updating pseudo-bridge configuration for %d interfaces", len(waitInterface))
	s.mu.Lock()
	defer s.mu.Unlock()

	addIF := make(map[string]ResponderNetworks)
	updateIF := make(map[string]ResponderNetworks)
	delIF := make(map[string]*IPNetSynced)

	for _, runningNets := range s.runningInterface {
		runningNets.ExistsInNewConfig = false
	}

	for newIF, newNets := range waitInterface {
		runningIF, ok := s.runningInterface[newIF]
		if ok {
			runningIF.ExistsInNewConfig = true
			if !(models.NetworksEqual(runningIF.IPNet.V4Networks, newNets.V4Networks) &&
				models.NetworksEqual(runningIF.IPNet.V6Networks, newNets.V6Networks) &&
				models.NetworksEqual(runningIF.IPNet.V4Offsets, newNets.V4Offsets) &&
				models.NetworksEqual(runningIF.IPNet.V6Offsets, newNets.V6Offsets)) &&
				models.IPsEqual(runningIF.IPNet.V4Skipped, newNets.V4Skipped) &&
				models.IPsEqual(runningIF.IPNet.V6Skipped, newNets.V6Skipped) {
				// Need update
				runningIF.IPNet = newNets
				updateIF[newIF] = newNets
			}
		} else {
			addIF[newIF] = newNets
		}
	}
	for runningIF, runningNets := range s.runningInterface {
		if !runningNets.ExistsInNewConfig {
			delIF[runningIF] = runningNets
		}
	}
	for ifname, af := range addIF {
		logging.LogVerbose("Adding pseudo-bridge responder for interface: %s", ifname)
		s.runningInterface[ifname] = &IPNetSynced{
			ExistsInNewConfig: false,
			IPNet:             af,
		}
		s.responders[ifname] = NewInterfaceResponder(ifname, af)
	}
	for ifname, af := range updateIF {
		logging.LogVerbose("Updating pseudo-bridge responder for interface: %s", ifname)
		s.responders[ifname].UpdateNetworks(af)
	}
	for ifname := range delIF {
		logging.LogVerbose("Removing pseudo-bridge responder for interface: %s", ifname)
		s.responders[ifname].Stop()
		delete(s.runningInterface, ifname)
		delete(s.responders, ifname)
	}
}

func (s *PseudoBridgeService) UpdateIfaceBindInfo(ifname string, ipv4net, ipv6net *models.IPNetWrapper, ipv4s, ipv6s []net.IP) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	responder, ok := s.responders[ifname]
	if !ok {
		return fmt.Errorf("responder for interface %v not found", ifname)
	}
	responder.UpdateIfaceBinds(ipv4net, ipv6net, ipv4s, ipv6s)
	return nil
}

func (s *PseudoBridgeService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop all responders safely
	for _, responder := range s.responders {
		if responder != nil {
			func() {
				responder.Stop()
			}()
		}
	}
	s.responders = make(map[string]*InterfaceResponder)

	logging.LogInfo("Pseudo-bridge Service stopped")
}

func NewInterfaceResponder(interfaceName string, networks ResponderNetworks) *InterfaceResponder {
	ifr := &InterfaceResponder{
		interfaceName:   interfaceName,
		networks:        networks,
		workingNetworks: networks,
		stopCh:          make(chan struct{}),
	}
	v4s, v6s, err := utils.GetInterfaceIPs(ifr.interfaceName)
	if err != nil {
		logging.LogError("Failed to get interface IPs for %v: %v", ifr.interfaceName, err)
	} else {
		ifr.bindedIPv4s = make([]net.IP, len(v4s))
		copy(ifr.bindedIPv4s, v4s)
		ifr.bindedIPv6s = make([]net.IP, len(v6s))
		copy(ifr.bindedIPv6s, v6s)
	}

	// Start the main loop in a separate goroutine
	go ifr.mainLoop()
	return ifr
}

func (r *InterfaceResponder) mainLoop() {
	var handle *pcap.Handle
	var packetSource *gopacket.PacketSource
	var err error
	defer func() {
		if handle != nil {
			handle.Close()
		}
		close(r.stopCh)
		logging.LogInfo("Pseudo-bridge Responder for %s stopped", r.interfaceName)
	}()
	logging.LogInfo("Pseudo-bridge Responder for %s starting", r.interfaceName)
	for {
		// Try to open pcap handle for the interface
		if handle == nil {
			if err = utils.IsIfaceLayer2(r.interfaceName); err != nil {
				logging.LogError("Interface %s layer2 check failed, retrying in 5 seconds: %v", r.interfaceName, err)
				time.Sleep(5 * time.Second)
				continue
			}
			if handle, err = pcap.OpenLive(r.interfaceName, 9200, false, pcap.BlockForever); err != nil {
				logging.LogError("Failed to open pcap handle for %s, retrying in 5 seconds: %v", r.interfaceName, err)
				time.Sleep(5 * time.Second)
				continue
			}
			// Set BPF filter for ARP and ICMPv6, excluding VLAN-tagged packets
			// This filters at kernel level for better efficiency
			filter := "(arp or (icmp6 and ip6[40] == 135)) and not vlan" // ARP or Neighbor Solicitation, but not VLAN-tagged
			if err = handle.SetBPFFilter(filter); err != nil {
				handle.Close()
				logging.LogError("Failed to set BPF filter for %s, retrying in 5 seconds: %v", r.interfaceName, err)
				time.Sleep(5 * time.Second)
				continue
			}
			packetSource = gopacket.NewPacketSource(handle, handle.LinkType())
			logging.LogInfo("Pseudo-bridge Responder for %s started, listening ARP and NS on main network", r.interfaceName)
		} else if packetSource == nil {
			packetSource = gopacket.NewPacketSource(handle, handle.LinkType())
		} else {
			r.handle = handle
			select {
			case <-r.stopCh:
				return
			case packet := <-packetSource.Packets():
				if packet == nil {
					// Interface might have disappeared, restart with retry logic
					logging.LogError("Packet source returned nil for %s, retry listening after 5 seconds", r.interfaceName)
					handle.Close()
					handle = nil
					packetSource = nil
					r.handle = nil
					time.Sleep(5 * time.Second)
					continue
				}
				r.handlePacket(packet)
			}
		}
	}
}

func (r *InterfaceResponder) Stop() {
	r.stopCh <- struct{}{}
}

func (r *InterfaceResponder) UpdateNetworks(networks ResponderNetworks) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.networks = networks
	r.networks.V4Networks = networks.V4Networks
	r.networks.V6Networks = networks.V6Networks
	r.setupWorkingNets()
}
func (r *InterfaceResponder) UpdateIfaceBinds(ipv4net, ipv6net *models.IPNetWrapper, ipv4s, ipv6s []net.IP) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bindedIPv4s = make([]net.IP, len(ipv4s))
	copy(r.bindedIPv4s, ipv4s)
	r.bindedIPv6s = make([]net.IP, len(ipv6s))
	copy(r.bindedIPv6s, ipv6s)
	r.ipv4Base = ipv4net
	r.ipv6Base = ipv6net
	r.setupWorkingNets()
}

func (r *InterfaceResponder) setupWorkingNets() {
	r.workingNetworks.V4Networks = r.networks.V4Networks
	r.workingNetworks.V6Networks = r.networks.V6Networks
	if r.ipv4Base != nil {
		for _, v4offset := range r.networks.V4Offsets {
			newnet, err := r.ipv4Base.GetNetByOffset(v4offset)
			if err != nil {
				logging.LogError("Failed to get IPv4 net by offset: %v from base: %v, err: %v", v4offset, r.ipv4Base, err)
				continue
			}
			r.workingNetworks.V4Networks = append(r.workingNetworks.V4Networks, newnet)
		}
	}
	if r.ipv6Base != nil {
		for _, v6offset := range r.networks.V6Offsets {
			newnet, err := r.ipv6Base.GetNetByOffset(v6offset)
			if err != nil {
				logging.LogError("Failed to get IPv6 net by offset: %v from base: %v, err: %v", v6offset, r.ipv6Base, err)
				continue
			}
			r.workingNetworks.V6Networks = append(r.workingNetworks.V6Networks, newnet)
		}
	}
	r.workingSkipIPv4 = append(r.bindedIPv4s, r.networks.V4Skipped...)
	r.workingSkipIPv6 = append(r.bindedIPv6s, r.networks.V6Skipped...)

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
	logging.LogVerbose("ARP request detected on %s for target IP: %s", r.interfaceName, targetIP.String())

	r.mu.RLock()
	networks := r.workingNetworks.V4Networks
	skipIPs := r.workingSkipIPv4
	r.mu.RUnlock()

	// Check if target IP is bound to interface - if so, don't respond
	for _, SkipIP := range skipIPs {
		if targetIP.Equal(SkipIP) {
			logging.LogVerbose("ARP request for %s on %s - skipping (IP is bound to interface)", targetIP.String(), r.interfaceName)
			return
		}
	}

	// Check if target IP is in any of our managed networks
	for _, network := range networks {
		if network.Contains(targetIP) {
			logging.LogVerbose("ARP request for %s on %s - matched network %s, sending reply", targetIP.String(), r.interfaceName, network.String())
			r.sendARPReply(packet, arp)
			return
		}
	}
	logging.LogVerbose("ARP request for %s on %s - no matching networks, ignoring", targetIP.String(), r.interfaceName)
}

func (r *InterfaceResponder) handleNeighborSolicitation(packet gopacket.Packet, icmp6 *layers.ICMPv6) {
	// Parse the target address from the NS packet
	if len(icmp6.Payload) < 20 {
		logging.LogVerbose("Neighbor solicitation on %s - invalid payload length", r.interfaceName)
		return
	}

	targetIP := net.IP(icmp6.Payload[4:20])
	logging.LogVerbose("Neighbor solicitation detected on %s for target IP: %s", r.interfaceName, targetIP.String())

	r.mu.RLock()
	networks := r.workingNetworks.V6Networks
	skipIPs := r.workingSkipIPv6
	r.mu.RUnlock()

	// Check if target IP is bound to interface - if so, don't respond
	for _, SkipIP := range skipIPs {
		if targetIP.Equal(SkipIP) {
			logging.LogVerbose("Neighbor solicitation for %s on %s - skipping (IP is bound to interface)", targetIP.String(), r.interfaceName)
			return
		}
	}

	// Check if target IP is in any of our managed networks
	for _, network := range networks {
		if network.Contains(targetIP) {
			logging.LogVerbose("Neighbor solicitation for %s on %s - matched network %s, sending advertisement", targetIP.String(), r.interfaceName, network.String())
			r.sendNeighborAdvertisement(packet, targetIP)
			return
		}
	}
	logging.LogVerbose("Neighbor solicitation for %s on %s - no matching networks, ignoring", targetIP.String(), r.interfaceName)
}

func (r *InterfaceResponder) sendARPReply(requestPacket gopacket.Packet, requestARP *layers.ARP) {
	targetIP := net.IP(requestARP.DstProtAddress)
	// Get the interface's MAC address
	iface, err := net.InterfaceByName(r.interfaceName)
	if err != nil {
		logging.LogError("Failed to get interface %s for ARP reply: %v", r.interfaceName, err)
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

	if r.handle != nil {
		err := r.handle.WritePacketData(buffer.Bytes())
		if err != nil {
			logging.LogError("Failed to send ARP reply for %s on %s: %v", targetIP.String(), r.interfaceName, err)
		} else {
			logging.LogVerbose("Sent ARP reply for %s on %s (MAC: %s)", targetIP.String(), r.interfaceName, iface.HardwareAddr.String())
		}
	}
}

func (r *InterfaceResponder) sendNeighborAdvertisement(requestPacket gopacket.Packet, targetIP net.IP) {
	// Get the interface's MAC address
	iface, err := net.InterfaceByName(r.interfaceName)
	if err != nil {
		logging.LogError("Failed to get interface %s for neighbor advertisement: %v", r.interfaceName, err)
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

	if r.handle != nil {
		err := r.handle.WritePacketData(buffer.Bytes())
		if err != nil {
			logging.LogError("Failed to send neighbor advertisement for %s on %s: %v", targetIP.String(), r.interfaceName, err)
		} else {
			logging.LogVerbose("Sent neighbor advertisement for %s on %s (MAC: %s)", targetIP.String(), r.interfaceName, iface.HardwareAddr.String())
		}
	}
}
