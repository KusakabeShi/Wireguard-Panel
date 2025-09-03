package models

import (
	"testing"
)

func TestNetworksEqual(t *testing.T) {
	// Helper function to create IPNetWrapper from CIDR
	mustParseCIDR := func(cidr string) *IPNetWrapper {
		ipnet, err := ParseCIDR(cidr)
		if err != nil {
			t.Fatalf("Failed to parse CIDR %s: %v", cidr, err)
		}
		return ipnet
	}

	tests := []struct {
		name     string
		s1       []*IPNetWrapper
		s2       []*IPNetWrapper
		expected bool
	}{
		{
			name:     "Both nil slices",
			s1:       nil,
			s2:       nil,
			expected: true,
		},
		{
			name:     "One nil, one empty",
			s1:       nil,
			s2:       []*IPNetWrapper{},
			expected: false,
		},
		{
			name:     "Both empty slices",
			s1:       []*IPNetWrapper{},
			s2:       []*IPNetWrapper{},
			expected: true,
		},
		{
			name:     "Different lengths",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("10.0.0.0/8")},
			expected: false,
		},
		{
			name:     "Same single network",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24")},
			expected: true,
		},
		{
			name:     "Same networks in same order",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("10.0.0.0/8")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("10.0.0.0/8")},
			expected: true,
		},
		{
			name:     "Same networks in different order",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("10.0.0.0/8")},
			s2:       []*IPNetWrapper{mustParseCIDR("10.0.0.0/8"), mustParseCIDR("192.168.1.0/24")},
			expected: true,
		},
		{
			name:     "Different networks",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.2.0/24")},
			expected: false,
		},
		{
			name:     "Same network different mask lengths",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/25")},
			expected: false,
		},
		{
			name:     "IPv4 vs IPv6 networks",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24")},
			s2:       []*IPNetWrapper{mustParseCIDR("2001:db8::/32")},
			expected: false,
		},
		{
			name:     "Mixed IPv4 and IPv6 same order",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("2001:db8::/32")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("2001:db8::/32")},
			expected: true,
		},
		{
			name:     "Mixed IPv4 and IPv6 different order",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("2001:db8::/32")},
			s2:       []*IPNetWrapper{mustParseCIDR("2001:db8::/32"), mustParseCIDR("192.168.1.0/24")},
			expected: true,
		},
		{
			name:     "Duplicate networks in s1",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("192.168.1.0/24")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24")},
			expected: false,
		},
		{
			name:     "Duplicate networks in both slices same order",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("192.168.1.0/24")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("192.168.1.0/24")},
			expected: true,
		},
		{
			name:     "Duplicate networks in both slices different order",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("10.0.0.0/8"), mustParseCIDR("192.168.1.0/24")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("192.168.1.0/24"), mustParseCIDR("10.0.0.0/8")},
			expected: true,
		},
		{
			name:     "Complex mix with duplicates",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), mustParseCIDR("10.0.0.0/8"), mustParseCIDR("172.16.0.0/12"), mustParseCIDR("10.0.0.0/8")},
			s2:       []*IPNetWrapper{mustParseCIDR("10.0.0.0/8"), mustParseCIDR("172.16.0.0/12"), mustParseCIDR("192.168.1.0/24"), mustParseCIDR("10.0.0.0/8")},
			expected: true,
		},
		{
			name:     "IPv6 networks same",
			s1:       []*IPNetWrapper{mustParseCIDR("2001:db8::/32"), mustParseCIDR("fe80::/10")},
			s2:       []*IPNetWrapper{mustParseCIDR("fe80::/10"), mustParseCIDR("2001:db8::/32")},
			expected: true,
		},
		{
			name:     "IPv6 networks different",
			s1:       []*IPNetWrapper{mustParseCIDR("2001:db8::/32")},
			s2:       []*IPNetWrapper{mustParseCIDR("2001:db8::/64")},
			expected: false,
		},
		{
			name:     "Large set with mixed types",
			s1: []*IPNetWrapper{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("10.0.0.0/8"),
				mustParseCIDR("172.16.0.0/12"),
				mustParseCIDR("2001:db8::/32"),
				mustParseCIDR("fe80::/10"),
				mustParseCIDR("192.168.2.0/24"),
			},
			s2: []*IPNetWrapper{
				mustParseCIDR("fe80::/10"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("2001:db8::/32"),
				mustParseCIDR("172.16.0.0/12"),
				mustParseCIDR("10.0.0.0/8"),
				mustParseCIDR("192.168.1.0/24"),
			},
			expected: true,
		},
		{
			name:     "One slice has nil element",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), nil},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24")},
			expected: false,
		},
		{
			name:     "Both slices have nil elements same position",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), nil},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), nil},
			expected: true,
		},
		{
			name:     "Both slices have nil elements different positions",
			s1:       []*IPNetWrapper{nil, mustParseCIDR("192.168.1.0/24")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), nil},
			expected: true,
		},
		{
			name:     "Multiple nil elements",
			s1:       []*IPNetWrapper{nil, nil, mustParseCIDR("192.168.1.0/24")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.0/24"), nil, nil},
			expected: true,
		},
		{
			name:     "Same IP different base networks",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.100/24")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.100/25")},
			expected: false,
		},
		{
			name:     "Host addresses vs network addresses",
			s1:       []*IPNetWrapper{mustParseCIDR("192.168.1.1/32")},
			s2:       []*IPNetWrapper{mustParseCIDR("192.168.1.1/32")},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NetworksEqual(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("NetworksEqual() = %v, expected %v", result, tt.expected)
				t.Logf("s1: %v", formatNetworkSlice(tt.s1))
				t.Logf("s2: %v", formatNetworkSlice(tt.s2))
			}
		})
	}
}

// Helper function to format network slices for debugging
func formatNetworkSlice(slice []*IPNetWrapper) []string {
	if slice == nil {
		return nil
	}
	result := make([]string, len(slice))
	for i, net := range slice {
		if net == nil {
			result[i] = "<nil>"
		} else {
			result[i] = net.String()
		}
	}
	return result
}

// Benchmark test to ensure performance is reasonable
func BenchmarkNetworksEqual(b *testing.B) {
	// Create test data
	mustParseCIDR := func(cidr string) *IPNetWrapper {
		ipnet, err := ParseCIDR(cidr)
		if err != nil {
			b.Fatalf("Failed to parse CIDR %s: %v", cidr, err)
		}
		return ipnet
	}

	s1 := []*IPNetWrapper{
		mustParseCIDR("192.168.1.0/24"),
		mustParseCIDR("10.0.0.0/8"),
		mustParseCIDR("172.16.0.0/12"),
		mustParseCIDR("2001:db8::/32"),
		mustParseCIDR("fe80::/10"),
		mustParseCIDR("192.168.2.0/24"),
		mustParseCIDR("203.0.113.0/24"),
		mustParseCIDR("198.51.100.0/24"),
	}

	s2 := []*IPNetWrapper{
		mustParseCIDR("fe80::/10"),
		mustParseCIDR("198.51.100.0/24"),
		mustParseCIDR("192.168.2.0/24"),
		mustParseCIDR("203.0.113.0/24"),
		mustParseCIDR("2001:db8::/32"),
		mustParseCIDR("172.16.0.0/12"),
		mustParseCIDR("10.0.0.0/8"),
		mustParseCIDR("192.168.1.0/24"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NetworksEqual(s1, s2)
	}
}

// Test that NetworksEqual doesn't modify original slices
func TestNetworksEqualNoMutation(t *testing.T) {
	mustParseCIDR := func(cidr string) *IPNetWrapper {
		ipnet, err := ParseCIDR(cidr)
		if err != nil {
			t.Fatalf("Failed to parse CIDR %s: %v", cidr, err)
		}
		return ipnet
	}

	// Create test slices in reverse order
	original1 := []*IPNetWrapper{
		mustParseCIDR("192.168.2.0/24"),
		mustParseCIDR("192.168.1.0/24"),
		mustParseCIDR("10.0.0.0/8"),
	}
	original2 := []*IPNetWrapper{
		mustParseCIDR("10.0.0.0/8"),
		mustParseCIDR("192.168.1.0/24"),
		mustParseCIDR("192.168.2.0/24"),
	}

	// Create copies to compare against later
	backup1 := make([]*IPNetWrapper, len(original1))
	backup2 := make([]*IPNetWrapper, len(original2))
	copy(backup1, original1)
	copy(backup2, original2)

	// Call NetworksEqual
	result := NetworksEqual(original1, original2)

	// Verify the function returned correct result
	if !result {
		t.Errorf("NetworksEqual should return true for equivalent slices")
	}

	// Verify original slices weren't modified
	for i, net := range original1 {
		if !net.Equal(backup1[i]) {
			t.Errorf("Original slice 1 was modified at index %d: expected %s, got %s", 
				i, backup1[i].String(), net.String())
		}
	}

	for i, net := range original2 {
		if !net.Equal(backup2[i]) {
			t.Errorf("Original slice 2 was modified at index %d: expected %s, got %s", 
				i, backup2[i].String(), net.String())
		}
	}
}

// Test edge cases that might cause panics
func TestNetworksEqualEdgeCases(t *testing.T) {
	mustParseCIDR := func(cidr string) *IPNetWrapper {
		ipnet, err := ParseCIDR(cidr)
		if err != nil {
			t.Fatalf("Failed to parse CIDR %s: %v", cidr, err)
		}
		return ipnet
	}

	// Test with all nil elements
	t.Run("All nil elements", func(t *testing.T) {
		s1 := []*IPNetWrapper{nil, nil, nil}
		s2 := []*IPNetWrapper{nil, nil, nil}
		result := NetworksEqual(s1, s2)
		if !result {
			t.Errorf("Expected true for slices with all nil elements")
		}
	})

	// Test sorting stability
	t.Run("Sorting stability", func(t *testing.T) {
		// Create identical networks to test sorting stability
		net1a := mustParseCIDR("192.168.1.0/24")
		net1b := mustParseCIDR("192.168.1.0/24")
		net2 := mustParseCIDR("10.0.0.0/8")

		s1 := []*IPNetWrapper{net1a, net2, net1b}
		s2 := []*IPNetWrapper{net1b, net1a, net2}

		result := NetworksEqual(s1, s2)
		if !result {
			t.Errorf("Expected true for slices with identical networks regardless of reference equality")
		}
	})
}