package internalservice

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
	runningInterface map[string]*IPNetSynced
	responders       map[string]*InterfaceResponder
	mu               sync.RWMutex
}

type InterfaceResponder struct {
	interfaceName   string
	networks        ResponderNetworks // [v4/v6][networks...]
	workingNetworks ResponderNetworks
	ipv4Base        *models.IPNetWrapper
	ipv6Base        *models.IPNetWrapper
	handle          *pcap.Handle
	stopCh          chan struct{}
	mu              sync.RWMutex
}

type ResponderNetworks struct {
	V4Networks []models.IPNetWrapper
	V6Networks []models.IPNetWrapper
	V4Offsets  []models.IPNetWrapper
	V6Offsets  []models.IPNetWrapper
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
				models.NetworksEqual(runningIF.IPNet.V6Offsets, newNets.V6Offsets)) {
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
		s.runningInterface[ifname] = &IPNetSynced{
			ExistsInNewConfig: false,
			IPNet:             af,
		}
		s.responders[ifname] = NewInterfaceResponder(ifname, af)
	}
	for ifname, af := range updateIF {
		s.responders[ifname].UpdateNetworks(af)
	}
	for ifname := range delIF {
		s.responders[ifname].Stop()
		delete(s.runningInterface, ifname)
		delete(s.responders, ifname)
	}
}

func (s *PseudoBridgeService) UpdateBaseNets(ifname string, ipv4net, ipv6net *models.IPNetWrapper) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	responder, ok := s.responders[ifname]
	if !ok {
		return fmt.Errorf("responder for interface %v not found", ifname)
	}
	responder.UpdateBaseNets(ipv4net, ipv6net)
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

	log.Println("Pseudo-bridge Service stopped")
}

func NewInterfaceResponder(interfaceName string, networks ResponderNetworks) *InterfaceResponder {
	ifr := &InterfaceResponder{
		interfaceName: interfaceName,
		networks:      networks,
		stopCh:        make(chan struct{}),
	}
	go ifr.mainLoop()
	return ifr
}

func (r *InterfaceResponder) mainLoop() {
	var handle *pcap.Handle
	defer func() {
		if handle != nil {
			handle.Close()
		}
		close(r.stopCh)
		fmt.Printf("Responder for %v stopped", r.interfaceName)
	}()
	log.Println("Starting Pseudo-bridge Service")
	for {
		// Try to open pcap handle for the interface
		if handle == nil {
			handle, err := pcap.OpenLive(r.interfaceName, 1600, true, pcap.BlockForever)
			if err != nil {
				log.Printf("Failed to open pcap handle for %s, retrying in 5 seconds: %v", r.interfaceName, err)
				time.Sleep(5 * time.Second)

				continue
			}
			// Set BPF filter for ARP and ICMPv6
			filter := "arp or (icmp6 and ip6[40] == 135)" // ARP or Neighbor Solicitation
			if err := handle.SetBPFFilter(filter); err != nil {
				handle.Close()
				log.Printf("Failed to set BPF filter for %s, retrying in 5 seconds: %v", r.interfaceName, err)
				time.Sleep(5 * time.Second)
				continue
			}
		} else {
			packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
			select {
			case <-r.stopCh:
				return
			case packet := <-packetSource.Packets():
				if packet == nil {
					// Interface might have disappeared, restart with retry logic
					log.Printf("Packet source returned nil for %s, retry listening after 5 second", r.interfaceName)
					handle.Close()
					handle = nil
					time.Sleep(5 * time.Second)
				}
				r.handlePacket(packet)
			}
		}
		return
	}
}

func (r *InterfaceResponder) Stop() {
	r.stopCh <- struct{}{}
}

func (r *InterfaceResponder) UpdateNetworks(networks ResponderNetworks) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.networks = networks
	r.offset2workingNet()
}
func (r *InterfaceResponder) UpdateBaseNets(ipv4net, ipv6net *models.IPNetWrapper) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ipv4Base = ipv4net
	r.ipv6Base = ipv6net
	r.offset2workingNet()
}

func (r *InterfaceResponder) offset2workingNet() {
	r.workingNetworks.V4Networks = r.networks.V4Networks
	r.workingNetworks.V6Networks = r.networks.V6Networks
	if r.ipv4Base != nil {
		for _, v4offset := range r.networks.V4Offsets {
			newnet, err := r.ipv4Base.GetNetByOffset(&v4offset)
			if err != nil {
				log.Printf("Failed to get net by offset: %v from base: %v, err: %v", v4offset, r.ipv4Base, err)
				continue
			}
			r.workingNetworks.V4Networks = append(r.workingNetworks.V4Networks, *newnet)
		}
	}
	if r.ipv6Base != nil {
		for _, v6offset := range r.networks.V6Offsets {
			newnet, err := r.ipv6Base.GetNetByOffset(&v6offset)
			if err != nil {
				log.Printf("Failed to get net by offset: %v from base: %v, err: %v", v6offset, r.ipv6Base, err)
				continue
			}
			r.workingNetworks.V6Networks = append(r.workingNetworks.V6Networks, *newnet)
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
	networks := r.workingNetworks.V4Networks
	r.mu.RUnlock()

	// Check if target IP is in any of our managed networks
	for _, network := range networks {
		if network.Contains(targetIP) {
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
	networks := r.workingNetworks.V6Networks
	r.mu.RUnlock()

	// Check if target IP is in any of our managed networks
	for _, network := range networks {
		if network.Contains(targetIP) {
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
