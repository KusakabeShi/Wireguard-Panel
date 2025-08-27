package models

import (
	"fmt"
	"net"
	"time"
)

type SnatConfig struct {
	Enabled                bool          `json:"enabled"`
	SnatIPNet              *IPNetWrapper `json:"snatIpNet"`
	SnatExcludedNetwork    *IPNetWrapper `json:"snatExcludedNetwork"`
	RoamingMasterInterface *string       `json:"roamingMasterInterface"`
	RoamingPseudoBridge    bool          `json:"roamingPseudoBridge"`
}

type ServerNetworkConfig struct {
	Enabled                     bool           `json:"enabled"`
	Network                     *IPNetWrapper  `json:"network"`
	PseudoBridgeMasterInterface *string        `json:"pseudoBridgeMasterInterface"`
	Snat                        *SnatConfig    `json:"snat"`
	RoutedNetworks              []IPNetWrapper `json:"routedNetworks"`
	RoutedNetworksFirewall      bool           `json:"routedNetworksFirewall"`
	CommentString               string         `json:"commentString"`
}

func (src *ServerNetworkConfig) Copy() (dst *ServerNetworkConfig) {
	dst = &ServerNetworkConfig{}
	*dst = *src
	*dst.Network = *src.Network
	*dst.PseudoBridgeMasterInterface = *src.PseudoBridgeMasterInterface
	*dst.Snat = *src.Snat
	copy(dst.RoutedNetworks, src.RoutedNetworks)
	return
}

type WGState struct {
	LatestHandshake *time.Time `json:"latestHandshake"`
	Endpoint        *string    `json:"endpoint"`
	TransferRx      *int64     `json:"transferRx"`
	TransferTx      *int64     `json:"transferTx"`
}

type Interface struct {
	ID         string    `json:"id"`
	Ifname     string    `json:"ifname"`
	VRFName    *string   `json:"vrfName"`
	FwMark     *string   `json:"fwMark"`
	Endpoint   string    `json:"endpoint"`
	Port       int       `json:"port"`
	MTU        int       `json:"mtu"`
	PrivateKey string    `json:"privateKey,omitempty"`
	PublicKey  string    `json:"publicKey"`
	Servers    []*Server `json:"servers,omitempty"`
}

type Server struct {
	ID      string               `json:"id"`
	Name    string               `json:"name"`
	Enabled bool                 `json:"enabled"`
	DNS     []string             `json:"dns"`
	IPv4    *ServerNetworkConfig `json:"ipv4"`
	IPv6    *ServerNetworkConfig `json:"ipv6"`
	Clients []*Client            `json:"clients,omitempty"`
}

type Client struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Enabled      bool      `json:"enabled"`
	IPv4Offset   IPWrapper `json:"ipv4offset"`
	IPv6Offset   IPWrapper `json:"ipv6offset"`
	DNS          []string  `json:"dns"`
	PrivateKey   *string   `json:"privateKey,omitempty"`
	PublicKey    string    `json:"publicKey"`
	PresharedKey *string   `json:"presharedKey,omitempty"`
	Keepalive    *uint     `json:"keepalive"`
}

type ClientFrontend struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	IPv4         net.IP   `json:"ip"`
	IPv6         net.IP   `json:"ipv6"`
	DNS          []string `json:"dns"`
	PrivateKey   *string  `json:"privateKey,omitempty"`
	PublicKey    string   `json:"publicKey"`
	PresharedKey *string  `json:"presharedKey,omitempty"`
	Keepalive    *uint    `json:"keepalive"`
}

func (c *Client) ToClientFrontend(server *Server) (*ClientFrontend, error) {
	if c == nil {
		return nil, nil
	}
	clientFrontend := &ClientFrontend{
		ID:           c.ID,
		Name:         c.Name,
		Enabled:      c.Enabled,
		DNS:          c.DNS,
		PrivateKey:   c.PrivateKey,
		PublicKey:    c.PublicKey,
		PresharedKey: c.PresharedKey,
		Keepalive:    c.Keepalive,
	}
	if server != nil {
		v4, _ := c.GetIPv4(server.IPv4.Network)
		v6, _ := c.GetIPv6(server.IPv6.Network)
		if v4 != nil {
			clientFrontend.IPv4 = v4.IP
		}
		if v6 != nil {
			clientFrontend.IPv6 = v6.IP
		}
	}

	return clientFrontend, nil
}

// Helper methods for Client IP management
func (c *Client) GetIPv4(serverNet *IPNetWrapper) (*IPNetWrapper, error) {
	if serverNet == nil || c.IPv4Offset == nil {
		return nil, nil
	}
	return serverNet.GetByOffset(c.IPv4Offset)
}

func (c *Client) GetIPv6(serverNet *IPNetWrapper) (*IPNetWrapper, error) {
	if serverNet == nil || c.IPv6Offset == nil {
		return nil, nil
	}
	return serverNet.GetByOffset(c.IPv6Offset)
}

func (c *Client) SetIP(af int, serverNet *IPNetWrapper, ip net.IP, otherclients []*Client) error {
	if serverNet == nil {
		switch af {
		case 4:
			c.IPv4Offset = nil
		case 6:
			c.IPv6Offset = nil
		}
		return nil
	}
	if af != serverNet.Version {
		return fmt.Errorf("address family %d not match server network %s", af, serverNet.String())
	}
	if ip == nil {
		switch af {
		case 4:
			c.IPv4Offset = nil
		case 6:
			c.IPv6Offset = nil
		}
		return nil
	}
	if af == 4 {
		ip4 := ip.To4()
		if ip4 == nil {
			return fmt.Errorf("%s is not a valid IPv4", ip)
		}
		ip = ip4
	}
	if !serverNet.Contains(ip) {
		return fmt.Errorf("ip %s out if server network %s", ip, serverNet.BaseNet.String())
	}
	// Create temporary wrapper to calculate offset
	tempWrapper := &IPNetWrapper{
		Version: af,
		IP:      ip,
		BaseNet: serverNet.BaseNet,
	}
	offset, err := tempWrapper.GetOffset()
	if err != nil {
		return err
	}
	if serverNet.IP.Equal(ip) {
		return fmt.Errorf("ip %s conflic with server ip %s", ip, serverNet.String())
	}
	for _, client := range otherclients {
		if client.ID == c.ID {
			continue
		}
		otherclient_offset := IPWrapper{}
		switch af {
		case 4:
			otherclient_offset = client.IPv4Offset
			otherclient_offset = otherclient_offset.To4()
		case 6:
			otherclient_offset = client.IPv6Offset
		}

		if client != nil && otherclient_offset != nil && offset.Equal(otherclient_offset) {
			return fmt.Errorf("ip %s conflic with client %s", ip, client.Name)
		}
	}
	switch af {
	case 4:
		c.IPv4Offset = offset
	case 6:
		c.IPv6Offset = offset
	}

	return nil
}

func (s *Server) GetNetwork(af int) *IPNetWrapper {
	if s == nil {
		return nil
	}
	switch af {
	case 4:
		if s.IPv4 != nil {
			return s.IPv4.Network
		}
	case 6:
		if s.IPv6 != nil {
			return s.IPv6.Network
		}
	}
	return nil
}
