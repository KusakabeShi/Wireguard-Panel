package models

import (
	"net"
	"net/netip"
)

const hexDigit = "0123456789abcdef"

type IPWrapper []byte

// Is p all zeros?
func isZeros(p IPWrapper) bool {
	for i := 0; i < len(p); i++ {
		if p[i] != 0 {
			return false
		}
	}
	return true
}

// To4 converts the IPv4 address ip to a 4-byte representation.
// If ip is not an IPv4 address, To4 returns nil.
func (ip IPWrapper) To4() IPWrapper {
	if len(ip) == net.IPv4len {
		return ip
	}
	if len(ip) == net.IPv6len &&
		isZeros(ip[0:10]) &&
		ip[10] == 0xff &&
		ip[11] == 0xff {
		return ip[12:16]
	}
	return nil
}

// To16 converts the IP address ip to a 16-byte representation.
// If ip is not an IP address (it is the wrong length), To16 returns nil.
func (ip IPWrapper) To16() IPWrapper {
	if len(ip) == net.IPv4len {
		return IPWrapper(net.IPv4(ip[0], ip[1], ip[2], ip[3]))
	}
	if len(ip) == net.IPv6len {
		return ip
	}
	return nil
}

func (ip IPWrapper) Equal(x IPWrapper) bool {
	return net.IP(ip).Equal(net.IP(x))
}

func hexString(b []byte) string {
	s := make([]byte, len(b)*2)
	for i, tn := range b {
		s[i*2], s[i*2+1] = hexDigit[tn>>4], hexDigit[tn&0xf]
	}
	return string(s)
}

// appendTo appends the string representation of ip to b and returns the expanded b
// If len(ip) != IPv4len or IPv6len, it appends nothing.
func (ip IPWrapper) appendTo(b []byte) []byte {
	// If IPv4, use dotted notation.
	if p4 := ip.To4(); len(p4) == net.IPv4len {
		ip = p4
	}
	addr, _ := netip.AddrFromSlice(ip)
	return addr.AppendTo(b)
}

// AppendText implements the [encoding.TextAppender] interface.
// The encoding is the same as returned by [IP.String], with one exception:
// When len(ip) is zero, it appends nothing.
func (ip IPWrapper) AppendText(b []byte) ([]byte, error) {
	if len(ip) == 0 {
		return b, nil
	}
	if len(ip) != net.IPv4len && len(ip) != net.IPv6len {
		return b, &net.AddrError{Err: "invalid IP address", Addr: hexString(ip)}
	}

	return ip.appendTo(b), nil
}

// MarshalText implements the [encoding.TextMarshaler] interface.
// The encoding is the same as returned by [IP.String], with one exception:
// When len(ip) is zero, it returns an empty slice.
func (ip IPWrapper) MarshalText() ([]byte, error) {
	// 24 is satisfied with all IPv4 addresses and short IPv6 addresses
	b, err := ip.AppendText(make([]byte, 0, 24))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
// The IP address is expected in a form accepted by [ParseIP].
func (ip *IPWrapper) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*ip = nil
		return nil
	}
	s := string(text)
	x := net.ParseIP(s)
	if x == nil {
		return &net.ParseError{Type: "IP address", Text: s}
	}
	ip4 := x.To4()
	if ip4 != nil {
		x = ip4
	}
	*ip = IPWrapper(x)
	return nil
}
