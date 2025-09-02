package utils

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"syscall"
	"unicode/utf8"
	"wg-panel/internal/models"

	"github.com/vishvananda/netlink"
)

func GetInterfaceIP(ifname string) (*models.IPNetWrapper, *models.IPNetWrapper, error) {
	link, err := netlink.LinkByName(ifname)
	if err != nil {
		return nil, nil, fmt.Errorf("get link %s: %w", ifname, err)
	}

	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, nil, fmt.Errorf("list addrs: %w", err)
	}

	var v4s, v6s []netlink.Addr
	for _, addr := range addrs {
		// Skip tentative/deprecated/dadfailed
		if addr.Flags&(syscall.IFA_F_TENTATIVE|syscall.IFA_F_DEPRECATED|syscall.IFA_F_DADFAILED) != 0 {
			continue
		}
		if addr.IP == nil || addr.IPNet == nil {
			continue
		}
		if addr.IP.To4() != nil {
			v4s = append(v4s, addr)
		} else if addr.IP.To16() != nil {
			v6s = append(v6s, addr)
		}
	}

	var bestV4, bestV6 *models.IPNetWrapper
	if len(v4s) > 0 {
		sort.Slice(v4s, func(i, j int) bool { return better(v4s[i], v4s[j]) })
		bestV4 = &models.IPNetWrapper{
			Version: 4,
			IP:      v4s[0].IP,
			BaseNet: *v4s[0].IPNet,
		}
	}
	if len(v6s) > 0 {
		sort.Slice(v6s, func(i, j int) bool { return better(v6s[i], v6s[j]) })
		bestV6 = &models.IPNetWrapper{
			Version: 6,
			IP:      v6s[0].IP,
			BaseNet: *v6s[0].IPNet,
		}
	}

	return bestV4, bestV6, nil
}

func GetInterfaceIPs(ifname string) ([]net.IP, []net.IP, error) {
	link, err := netlink.LinkByName(ifname)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get link by name %v: %w", ifname, err)
	}

	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list addresses: %w", err)
	}

	var ipv4s, ipv6s []net.IP
	for _, addr := range addrs {
		if addr.IP == nil {
			continue
		}
		if addr.IP.To4() != nil {
			ipv4s = append(ipv4s, addr.IP)
		} else if addr.IP.To16() != nil {
			ipv6s = append(ipv6s, addr.IP)
		}
	}

	return ipv4s, ipv6s, nil
}

// === Sorting helpers ===

func better(a, b netlink.Addr) bool {
	// 1. Prefix length priority
	if cmp := comparePrefix(a, b); cmp != 0 {
		return cmp < 0
	}
	// 2. Public > ULA/private > link local
	if cmp := compareScope(a, b); cmp != 0 {
		return cmp < 0
	}
	// 3. Flags Permanent > Temporary > Secondary
	if cmp := compareFlags(a, b); cmp != 0 {
		return cmp < 0
	}
	// 4. Smaller IP first
	return bytesCompare(a.IP, b.IP) < 0
}

func comparePrefix(a, b netlink.Addr) int {
	_, aNet, _ := net.ParseCIDR(a.IPNet.String())
	_, bNet, _ := net.ParseCIDR(b.IPNet.String())
	aLen, _ := aNet.Mask.Size()
	bLen, _ := bNet.Mask.Size()

	isV6 := a.IP.To4() == nil
	if isV6 {
		// prefer >= /64
		if aLen >= 64 && bLen < 64 {
			return -1
		}
		if bLen >= 64 && aLen < 64 {
			return 1
		}
		// both >= /64 → larger prefix (more specific) wins
		if aLen >= 64 && bLen >= 64 && aLen != bLen {
			return bLen - aLen
		}
		// both < /64 → smaller prefix wins
		if aLen != bLen {
			return aLen - bLen
		}
	} else {
		// IPv4 case
		if aLen >= 24 && bLen < 24 {
			return -1
		}
		if bLen >= 24 && aLen < 24 {
			return 1
		}
		if aLen >= 24 && bLen >= 24 && aLen != bLen {
			return bLen - aLen
		}
		if aLen != bLen {
			return aLen - bLen
		}
	}
	return 0
}

func compareScope(a, b netlink.Addr) int {
	scopeRank := func(ip net.IP) int {
		if ip.IsGlobalUnicast() && !isPrivate(ip) {
			return 0 // best
		}
		if isPrivate(ip) || isULA(ip) {
			return 1
		}
		if ip.IsLinkLocalUnicast() {
			return 2
		}
		return 3
	}
	ra, rb := scopeRank(a.IP), scopeRank(b.IP)
	if ra != rb {
		return ra - rb
	}
	return 0
}

func compareFlags(a, b netlink.Addr) int {
	rank := func(flags int) int {
		if flags&syscall.IFA_F_PERMANENT != 0 {
			return 0
		}
		if flags&syscall.IFA_F_TEMPORARY != 0 {
			return 1
		}
		if flags&syscall.IFA_F_SECONDARY != 0 {
			return 2
		}
		return 3
	}
	ra, rb := rank(a.Flags), rank(b.Flags)
	if ra != rb {
		return ra - rb
	}
	return 0
}

func bytesCompare(a, b net.IP) int {
	return bytesCompareRaw(a.To16(), b.To16())
}

func bytesCompareRaw(a, b []byte) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func isPrivate(ip net.IP) bool {
	return ip.IsPrivate()
}

func isULA(ip net.IP) bool {
	// fc00::/7
	return ip.To16() != nil && ip[0]&0xfe == 0xfc
}

// isValidIfName checks whether the input string is a valid Linux interface name.
var ifNameRegexp = regexp.MustCompile("^[A-Za-z0-9_-]{1,12}$")

func IsValidIfname(ifname string) bool {
	return ifNameRegexp.MatchString(ifname)
}

var allowedIfNameChars = regexp.MustCompile(`^[A-Za-z0-9._@-]+$`)

// IsValidPhyIfName validates a Linux interface name and returns a detailed error if invalid.
func IsValidPhyIfName(ifname string) error {
	if ifname == "" {
		return fmt.Errorf("interface name cannot be empty")
	}

	length := utf8.RuneCountInString(ifname)
	if length > 15 {
		return fmt.Errorf("interface name %q is too long: got %d characters, max allowed is 15", ifname, length)
	}

	if !allowedIfNameChars.MatchString(ifname) {
		return fmt.Errorf("interface name %q contains invalid characters: allowed are letters, digits, '.', '_', '-', '@'", ifname)
	}

	return nil
}

func IsIfExists(ifname string) error {
	_, err := netlink.LinkByName(ifname)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return fmt.Errorf("interface %s does not exist", ifname)
		}
		return fmt.Errorf("error checking interface %s: %v", ifname, err)
	}
	return nil
}

func IsIfaceLayer2(ifname string) error {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return fmt.Errorf("interface %q not found: %w", ifname, err)
	}

	// Check if it has a hardware (MAC) address
	if len(iface.HardwareAddr) == 0 {
		return fmt.Errorf("interface %q is not a Layer 2 device (no MAC address)", ifname)
	}

	return nil
}
