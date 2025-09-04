package models

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"sort"
	"strconv"
	"strings"
)

type IPNetWrapper struct {
	Version int       // 4 or 6
	IP      net.IP    // IPNet with IP and Mask
	BaseNet net.IPNet // Base Network and Mask
}

func ParseIP(ipStr string) (net.IP, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}
	if ip.To4() != nil {
		return ip.To4(), nil
	}
	return ip, nil
}

func ParseCIDR(cidr string) (*IPNetWrapper, error) {
	version := 6

	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	ip4 := ip.To4()
	if ip4 != nil {
		version = 4
		ip = ip4
	}
	return &IPNetWrapper{
		Version: version,
		IP:      ip,
		BaseNet: *ipnet,
	}, nil
}

func (basenet *IPNetWrapper) EqualZero(af int) bool {
	if basenet == nil {
		return false
	}
	if af == 4 {
		zerov4, _ := ParseCIDR("0.0.0.0/32")
		return basenet.Equal(zerov4)
	}
	if af == 6 {
		zerov6, _ := ParseCIDR("::/128")
		return basenet.Equal(zerov6)
	}
	return false
}

func ParseCIDRAf(af int, cidr string) (ipnet *IPNetWrapper, err error) {
	ipnet, err = ParseCIDRFromIP(cidr)
	if err != nil {
		return
	}
	if ipnet.Version != af {
		return nil, fmt.Errorf("%s is not a IPv%d network", cidr, af)
	}
	return
}

func ParseCIDRFromIP(cidr string) (*IPNetWrapper, error) {
	if !strings.Contains(cidr, "/") {
		ip, err := ParseIP(cidr)
		if err != nil {
			return nil, err
		}
		if ip.To4() != nil {
			cidr = cidr + "/32"
		} else {
			cidr = cidr + "/128"
		}
	}
	ipnet, err := ParseCIDR(cidr)
	return ipnet, err
}

func ParseCIDRFromIPAf(af int, cidr string) (ipnet *IPNetWrapper, err error) {
	ipnet, err = ParseCIDRFromIP(cidr)
	if err != nil {
		return
	}
	if ipnet.Version != af {
		return nil, fmt.Errorf("%s is not a IPv%d address/network", cidr, af)
	}
	return
}

func (basenet IPNetWrapper) String() string {
	masklen := basenet.Masklen()
	return basenet.IP.String() + "/" + strconv.Itoa(masklen)
}

func (basenet IPNetWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(basenet.String())
}

func (basenet *IPNetWrapper) UnmarshalJSON(data []byte) error {
	var cidr string
	if err := json.Unmarshal(data, &cidr); err != nil {
		return err
	}

	IPNet, err := ParseCIDR(cidr)
	if err != nil {
		return err
	}
	basenet.Version = IPNet.Version
	basenet.IP = IPNet.IP
	basenet.BaseNet = IPNet.BaseNet

	return nil
}

func (basenet IPNetWrapper) Masklen() int {
	masklen, _ := basenet.BaseNet.Mask.Size()
	return masklen
}

func (basenet IPNetWrapper) NetworkStr() string {
	return basenet.BaseNet.String()
}

func (basenet IPNetWrapper) Network() IPNetWrapper {
	return IPNetWrapper{
		Version: basenet.Version,
		IP:      basenet.BaseNet.IP,
		BaseNet: basenet.BaseNet,
	}
}

func (basenet *IPNetWrapper) Contains(ip net.IP) bool {
	if basenet == nil {
		return false
	}
	return basenet.BaseNet.Contains(ip)
}

func (basenet *IPNetWrapper) GetOffset() (IPWrapper, error) {
	if basenet == nil {
		return IPWrapper{}, fmt.Errorf("IPNetWrapper is nil")
	}
	if !basenet.BaseNet.Contains(basenet.IP) {
		return IPWrapper{}, fmt.Errorf("IP: %v exceeds the network range: %v", basenet.IP, basenet.BaseNet)
	}
	mask := basenet.BaseNet.Mask
	tip := basenet.IP
	networkIP := tip.Mask(mask)

	offset := make(IPWrapper, len(tip))
	for i := 0; i < len(tip); i++ {
		offset[i] = tip[i] ^ networkIP[i]
	}
	return offset, nil
}

func (basenet *IPNetWrapper) CheckOffsetValid(offset IPWrapper) error {
	if basenet == nil {
		return fmt.Errorf("base ipnet is nil")
	}
	// Normalize offset length to match w.IP
	if len(basenet.IP) == 4 && len(offset) == 16 {
		offset = offset.To4()
	} else if len(basenet.IP) == 16 && len(offset) == 4 {
		offset = offset.To16()
	}
	if offset == nil || len(basenet.IP) != len(offset) {
		return fmt.Errorf("IP:%v and offset:%v length mismatch", basenet.IP, offset)
	}
	// Check if offset exceeds the max value based on the mask
	mask := basenet.BaseNet.Mask
	masklen, bits := mask.Size()
	if masklen < 0 || bits < 0 {
		return fmt.Errorf("invalid mask: %v", mask)
	}
	// Check that offset does not exceed the host bits
	for i := 0; i < len(offset); i++ {
		if offset[i]&mask[i] != 0 {
			return fmt.Errorf("offset %v exceeds host bits for mask %v.", offset, mask)
		}
	}
	return nil
}

func (basenet *IPNetWrapper) GetByOffset(offset IPWrapper) (*IPNetWrapper, error) {
	// Normalize offset length to match w.IP
	if basenet == nil {
		return nil, fmt.Errorf("base ipnet is nil")
	}
	if len(basenet.IP) == 4 && len(offset) == 16 {
		offset = offset.To4()
	} else if len(basenet.IP) == 16 && len(offset) == 4 {
		offset = offset.To16()
	}
	if offset == nil || len(basenet.IP) != len(offset) {
		return nil, fmt.Errorf("IP:%v and offset:%v length mismatch", basenet.IP, offset)
	}

	// Check if offset exceeds the max value based on the mask
	err := basenet.CheckOffsetValid(offset)
	if err != nil {
		return nil, err
	}

	result := make(net.IP, len(basenet.IP))
	for i := 0; i < len(basenet.IP); i++ {
		result[i] = basenet.BaseNet.IP[i] | offset[i]
	}
	return &IPNetWrapper{
		Version: basenet.Version,
		IP:      result,
		BaseNet: basenet.BaseNet,
	}, nil
}

func (offset *IPNetWrapper) IpExceed2PowerN(powern int) bool {
	var totalBits int
	if offset.Version == 4 {
		totalBits = 32
	} else {
		totalBits = 128
	}

	// Create mask that allows only the offset bits to be set
	offsetMask := net.CIDRMask(powern, totalBits)

	// Check if offset.IP has any bits set outside the allowed range
	for i := 0; i < len(offset.IP); i++ {
		if i < len(offsetMask) && (offset.IP[i]&offsetMask[i]) != 0 {
			return true
		}
	}
	return false
}

func (offset *IPNetWrapper) IsHostbitAllZero() bool {
	offsetMask := net.CIDRMask(offset.Masklen(), len(offset.IP)*8)
	return offset.IP.Equal(offset.IP.Mask(offsetMask))
}

func (basenet *IPNetWrapper) GetSubnetByOffset(offset *IPNetWrapper) (*IPNetWrapper, error) {
	// Get smaller block of networks from w based on offset.
	// Example: w= 2a0d:3a87::/64. offset= ::980d:0/112, returns 2a0d:3a87::980d:0/112
	// Raise error if offset is not aligned, like w= 2a0d:3a87::/64. offset= ::980d:0/96, returns 2a0d:3a87::980d:0/96 but 2a0d:3a87::980d:0/96 is not a valid base network
	if basenet == nil {
		return nil, fmt.Errorf("w is nil")
	}
	if offset == nil {
		return basenet, nil
	}
	if basenet.Version != offset.Version {
		return nil, fmt.Errorf("basenet and offset in different address family")
	}
	if offset.Masklen() < basenet.Masklen() {
		return nil, fmt.Errorf("offset block size(/%v) must <= basenet block size(/%v)", offset.Masklen(), basenet.Masklen())
	}
	if offset.Masklen() == basenet.Masklen() {
		if !offset.IP.IsUnspecified() {
			return nil, fmt.Errorf("same size of network and offset, offset IP must be zero")
		}
		return basenet, nil
	}

	// offset.IP must not exceed the available host bits in the base network
	// for ipv4: if base is /16 and offset is /24, offset.IP must be in 0.0.0.0~0.0.255.0
	// for ipv6: if base is /64 and offset is /96, offset.IP must be in ::~::ffff:ffff:0

	// Create a mask for the allowed offset bits
	if offset.IpExceed2PowerN(basenet.Masklen()) {
		return nil, fmt.Errorf("offset %s exceeds allowed range for base network %s", offset, basenet.BaseNet.String())
	}

	// Verify that the offset network's base IP is properly aligned for its mask
	if !offset.IsHostbitAllZero() {
		return nil, fmt.Errorf("non-zero host bits: offset %s is not properly aligned for its mask", offset.String())
	}

	new_IP, err := basenet.GetByOffset(IPWrapper(offset.IP))
	if err != nil {
		return nil, err
	}
	var bits int
	if new_IP.Version == 4 {
		bits = 32
	} else {
		bits = 128
	}
	mask := net.CIDRMask(offset.Masklen(), bits)
	resultIP := new_IP.IP.Mask(mask)

	// Verify the result is within the original network bounds
	if !basenet.BaseNet.Contains(resultIP) {
		return nil, fmt.Errorf("resulting network %s/%d exceeds original network bounds %s", resultIP, offset.Masklen(), basenet.BaseNet.String())
	}

	return &IPNetWrapper{
		Version: new_IP.Version,
		IP:      new_IP.IP,
		BaseNet: net.IPNet{
			IP:   resultIP,
			Mask: mask,
		},
	}, nil
}

func (basenet *IPNetWrapper) IsOverlap(w2 *IPNetWrapper) bool {
	if basenet == nil || w2 == nil {
		return false
	}
	if basenet.Version != w2.Version {
		return false
	}
	return basenet.BaseNet.Contains(w2.BaseNet.IP) || w2.BaseNet.Contains(basenet.BaseNet.IP)
}

func IPNetLess(pw, pw2 *IPNetWrapper) bool {
	if pw == nil {
		return pw2 != nil // nil is less than non-nil
	}
	if pw2 == nil {
		return false // non-nil is not less than nil
	}
	w := *pw
	w2 := *pw2

	// First compare by version (IPv4 < IPv6)
	if w.Version != w2.Version {
		return w.Version < w2.Version
	}

	// Then compare by IP address bytes
	for i, v := range w.IP {
		if i >= len(w2.IP) {
			return false // w is longer, so it's greater
		}
		v2 := w2.IP[i]
		if v < v2 {
			return true
		}
		if v > v2 {
			return false
		}
		// if v == v2, continue to next byte
	}

	// Handle case where w2.IP is longer than w.IP
	if len(w.IP) < len(w2.IP) {
		return true // w is shorter, so it's less
	}

	// IPs are equal, compare by mask length (more specific is "greater", so reverse comparison)
	return w.Masklen() > w2.Masklen()
}

func IPLess(po1, po2 *net.IP) bool {
	if po1 == nil {
		return true
	}
	if po2 == nil {
		return false
	}
	o1 := *po1
	o2 := *po2
	if len(o1) != len(o2) {
		return len(o1) < len(o2)
	}
	for i, v := range o1 {
		v2 := o2[i]
		if v < v2 {
			return true
		}
	}
	return false
}

func IsIPv4(ip net.IP) bool {
	return ip.To4() != nil
}

func IsIPv6(ip net.IP) bool {
	return ip.To4() == nil && ip.To16() != nil
}

func IncrementIP2Power(ip net.IP, power int) net.IP {
	ipLen := len(ip)
	if ip.To4() != nil {
		ip = ip.To4()
		ipLen = 4
	} else {
		ip = ip.To16()
		ipLen = 16
	}

	// Convert IP to big.Int
	ipInt := new(big.Int).SetBytes(ip)

	// Add 2^power
	step := new(big.Int).Lsh(big.NewInt(1), uint(power)) // 2^power
	ipInt.Add(ipInt, step)

	// Convert back to []byte
	result := ipInt.Bytes()
	if len(result) < ipLen {
		padded := make([]byte, ipLen)
		copy(padded[ipLen-len(result):], result)
		return net.IP(padded)
	}
	return net.IP(result[len(result)-ipLen:])
}

func (basenet *IPNetWrapper) Equal(w2 *IPNetWrapper) bool {
	if basenet == nil && w2 == nil {
		return true
	}
	if basenet == nil || w2 == nil {
		return false
	}
	if basenet.Version != w2.Version {
		return false
	}
	if !basenet.IP.Equal(w2.IP) {
		return false
	}
	if !basenet.BaseNet.IP.Equal(w2.BaseNet.IP) {
		return false
	}
	if basenet.Masklen() != w2.Masklen() {
		return false
	}
	return true
}

func NetworksEqual(s1, s2 []*IPNetWrapper) bool {
	// Compare two slices of IPNetWrapper for equality, ignoring order
	if s1 == nil && s2 == nil {
		return true
	}
	if s1 == nil || s2 == nil {
		return false
	}
	if len(s1) != len(s2) {
		return false
	}

	// Create copies to avoid modifying original slices
	s1Copy := make([]*IPNetWrapper, len(s1))
	s2Copy := make([]*IPNetWrapper, len(s2))
	copy(s1Copy, s1)
	copy(s2Copy, s2)

	// Sort both copied slices
	sort.Slice(s1Copy, func(i, j int) bool {
		return IPNetLess(s1Copy[i], s1Copy[j])
	})
	sort.Slice(s2Copy, func(i, j int) bool {
		return IPNetLess(s2Copy[i], s2Copy[j])
	})

	// Compare each element
	for i := 0; i < len(s1Copy); i++ {
		if !s1Copy[i].Equal(s2Copy[i]) {
			return false
		}
	}

	return true
}

func IPsEqual(s1, s2 []net.IP) bool {
	if s1 == nil && s2 == nil {
		return true
	}
	if s1 == nil || s2 == nil {
		return false
	}
	if len(s1) != len(s2) {
		return false
	}

	// Create copies to avoid modifying original slices
	s1Copy := make([]net.IP, len(s1))
	s2Copy := make([]net.IP, len(s2))
	copy(s1Copy, s1)
	copy(s2Copy, s2)

	// Sort both copied slices
	sort.Slice(s1Copy, func(i, j int) bool {
		return IPLess(&s1Copy[i], &s1Copy[j])
	})
	sort.Slice(s2Copy, func(i, j int) bool {
		return IPLess(&s2Copy[i], &s2Copy[j])
	})

	// Compare each element
	for i := 0; i < len(s1Copy); i++ {
		if !s1Copy[i].Equal(s2Copy[i]) {
			return false
		}
	}

	return true
}

func NetworksEqualNP(s1, s2 []IPNetWrapper) bool {
	var s1p []*IPNetWrapper
	var s2p []*IPNetWrapper
	for i := range s1 {
		s1p = append(s1p, &s1[i])
	}
	for i := range s2 {
		s2p = append(s2p, &s2[i])
	}
	return NetworksEqual(s1p, s2p)

}
