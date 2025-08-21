package models

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net"
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

func ParseCIDRAf(cidr string, af int) (ipnet *IPNetWrapper, err error) {
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

		} else {
			cidr = cidr + "/128"
		}
	}
	ipnet, err := ParseCIDR(cidr)
	return ipnet, err
}

func ParseCIDRFromIPAf(cidr string, af int) (ipnet *IPNetWrapper, err error) {
	ipnet, err = ParseCIDRFromIP(cidr)
	if err != nil {
		return
	}
	if ipnet.Version != af {
		return nil, fmt.Errorf("%s is not a IPv%d address/network", cidr, af)
	}
	return
}

func (w IPNetWrapper) String() string {
	masklen := w.Masklen()
	return w.IP.String() + "/" + strconv.Itoa(masklen)
}

func (w IPNetWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(w.String())
}

func (w *IPNetWrapper) UnmarshalJSON(data []byte) error {
	var cidr string
	if err := json.Unmarshal(data, &cidr); err != nil {
		return err
	}

	IPNet, err := ParseCIDR(cidr)
	if err != nil {
		return err
	}
	w.Version = IPNet.Version
	w.IP = IPNet.IP
	w.BaseNet = IPNet.BaseNet

	return nil
}

func (w IPNetWrapper) Masklen() int {
	masklen, _ := w.BaseNet.Mask.Size()
	return masklen
}

func (w IPNetWrapper) NetworkStr() string {
	return w.BaseNet.String()
}

func (w IPNetWrapper) Network() IPNetWrapper {
	return IPNetWrapper{
		Version: w.Version,
		IP:      w.BaseNet.IP,
		BaseNet: w.BaseNet,
	}
}

func (w *IPNetWrapper) Contains(ip net.IP) bool {
	if w == nil {
		return false
	}
	return w.BaseNet.Contains(ip)
}

func (w *IPNetWrapper) GetOffset() (IPWrapper, error) {
	if w == nil {
		return IPWrapper{}, fmt.Errorf("IPNetWrapper is nil")
	}
	if !w.BaseNet.Contains(w.IP) {
		return IPWrapper{}, fmt.Errorf("IP: %v exceeds the network range: %v", w.IP, w.BaseNet)
	}
	mask := w.BaseNet.Mask
	tip := w.IP
	networkIP := tip.Mask(mask)

	offset := make(IPWrapper, len(tip))
	for i := 0; i < len(tip); i++ {
		offset[i] = tip[i] ^ networkIP[i]
	}
	return offset, nil
}

func (w *IPNetWrapper) CheckOffsetValid(offset IPWrapper) error {
	if w == nil {
		return fmt.Errorf("base ipnet is nil")
	}
	// Normalize offset length to match w.IP
	if len(w.IP) == 4 && len(offset) == 16 {
		offset = offset.To4()
	} else if len(w.IP) == 16 && len(offset) == 4 {
		offset = offset.To16()
	}
	if offset == nil || len(w.IP) != len(offset) {
		return fmt.Errorf("IP:%v and offset:%v length mismatch", w.IP, offset)
	}
	// Check if offset exceeds the max value based on the mask
	mask := w.BaseNet.Mask
	masklen, bits := mask.Size()
	if masklen < 0 || bits < 0 {
		return fmt.Errorf("invalid mask: %v", mask)
	}
	// Check that offset does not exceed the host bits
	for i := 0; i < len(offset); i++ {
		if offset[i]&mask[i] != 0 {
			return fmt.Errorf("offset %v exceeds host bits for mask %v", offset, mask)
		}
	}
	return nil
}

func (w *IPNetWrapper) GetByOffset(offset IPWrapper) (*IPNetWrapper, error) {
	// Normalize offset length to match w.IP
	if w == nil {
		return nil, fmt.Errorf("base ipnet is nil")
	}
	if len(w.IP) == 4 && len(offset) == 16 {
		offset = offset.To4()
	} else if len(w.IP) == 16 && len(offset) == 4 {
		offset = offset.To16()
	}
	if offset == nil || len(w.IP) != len(offset) {
		return nil, fmt.Errorf("IP:%v and offset:%v length mismatch", w.IP, offset)
	}

	// Check if offset exceeds the max value based on the mask
	err := w.CheckOffsetValid(offset)
	if err != nil {
		return nil, err
	}

	result := make(net.IP, len(w.IP))
	for i := 0; i < len(w.IP); i++ {
		result[i] = w.BaseNet.IP[i] | offset[i]
	}
	return &IPNetWrapper{
		Version: w.Version,
		IP:      result,
		BaseNet: w.BaseNet,
	}, nil
}

func (w *IPNetWrapper) IsOverlap(w2 *IPNetWrapper) bool {
	if w == nil || w2 == nil {
		return false
	}
	if w.Version != w2.Version {
		return false
	}
	return w.BaseNet.Contains(w2.BaseNet.IP) || w2.BaseNet.Contains(w.BaseNet.IP)
}

func IPNetLess(pw, pw2 *IPNetWrapper) bool {
	if pw == nil {
		return true
	}
	if pw2 == nil {
		return false
	}
	w := *pw
	w2 := *pw2
	if w.Version != w2.Version {
		return w.Version < w2.Version
	}
	for i, v := range w.IP {
		v2 := w2.IP[i]
		if v < v2 {
			return true
		}
	}
	return false
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

func (w *IPNetWrapper) Equal(w2 *IPNetWrapper) bool {
	if w == nil && w2 == nil {
		return true
	}
	if w == nil || w2 == nil {
		return false
	}
	if w.Version != w2.Version {
		return false
	}
	if !w.IP.Equal(w2.IP) {
		return false
	}
	if !w.BaseNet.IP.Equal(w2.BaseNet.IP) {
		return false
	}
	if w.Masklen() != w2.Masklen() {
		return false
	}
	return true
}
