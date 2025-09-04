package models

import (
	"testing"
)

func TestGetNetByOffset(t *testing.T) {
	tests := []struct {
		name        string
		base        string
		offset      string
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:     "IPv4 valid offset within range",
			base:     "192.168.0.0/16",
			offset:   "0.0.10.0/24",
			expected: "192.168.10.0/24",
		},
		{
			name:     "IPv4 zero offset same mask",
			base:     "192.168.0.0/16",
			offset:   "0.0.0.0/16",
			expected: "192.168.0.0/16",
		},
		{
			name:        "IPv4 zero offset same mask2",
			base:        "192.168.0.0/16",
			offset:      "0.0.0.1/16",
			expectError: true,
			expected:    "same size of network and offset, offset IP must be zero",
		},
		{
			name:        "IPv4 offset exceeds base network",
			base:        "192.168.0.0/16",
			offset:      "0.1.0.0/24",
			expectError: true,
			errorMsg:    "exceeds allowed range",
		},
		{
			name:        "IPv4 offset smaller mask than base",
			base:        "192.168.0.0/24",
			offset:      "10.0.0.0/16",
			expectError: true,
			errorMsg:    "must <= basenet block siz",
		},
		{
			name:     "IPv6 valid offset within range",
			base:     "2001:db8::/32",
			offset:   "0:0:1::/64",
			expected: "2001:db8:1::/64",
		},
		{
			name:     "IPv6 valid offset within range",
			base:     "2001:db8::/64",
			offset:   "0:0:0:0:8000::/65",
			expected: "2001:db8:0:0:8000::/65",
		},
		{
			name:        "IPv6 valid offset within range",
			base:        "2001:db8::/64",
			offset:      "0:0:0:1::/65",
			expectError: true,
			errorMsg:    "exceeds allowed range for base network",
		},
		{
			name:     "IPv6 zero offset same mask",
			base:     "2001:db8::/32",
			offset:   "::/32",
			expected: "2001:db8::/32",
		},
		{
			name:        "IPv6 offset not aligned for mask",
			base:        "2001:db8::/32",
			offset:      "::1:0/64",
			expectError: true,
			errorMsg:    "not properly aligned",
		},
		{
			name:        "IPv6 different address family",
			base:        "2001:db8::/32",
			offset:      "10.0.0.0/24",
			expectError: true,
			errorMsg:    "different address family",
		},
		{
			name:        "IPv4 different address family",
			base:        "192.168.0.0/16",
			offset:      "2001:db8::/64",
			expectError: true,
			errorMsg:    "different address family",
		},
		{
			name:        "Same mask with non-zero IP",
			base:        "192.168.0.0/16",
			offset:      "10.0.0.0/16",
			expectError: true,
			errorMsg:    "offset IP must be zero",
		},
		{
			name:     "IPv4 /30 to /32 conversion",
			base:     "192.168.1.0/30",
			offset:   "0.0.0.2/32",
			expected: "192.168.1.2/32",
		},
		{
			name:     "IPv6 /126 to /128 conversion",
			base:     "2001:db8::/126",
			offset:   "::2/128",
			expected: "2001:db8::2/128",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, err := ParseCIDR(tt.base)
			if err != nil {
				t.Fatalf("Failed to parse base CIDR %s: %v", tt.base, err)
			}

			offset, err := ParseCIDR(tt.offset)
			if err != nil {
				t.Fatalf("Failed to parse offset CIDR %s: %v", tt.offset, err)
			}

			result, err := base.GetSubnetByOffset(offset)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none. Result: %v", result)
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				} else if result == nil {
					t.Errorf("Expected result but got nil")
				} else if result.String() != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result.String())
				}
			}
		})
	}
}

func TestGetNetByOffsetNilCases(t *testing.T) {
	t.Run("Nil base network", func(t *testing.T) {
		var base *IPNetWrapper = nil
		offset, _ := ParseCIDR("10.0.0.0/24")

		result, err := base.GetSubnetByOffset(offset)
		if err == nil {
			t.Errorf("Expected error for nil base, got result: %v", result)
		}
		if err.Error() != "w is nil" {
			t.Errorf("Expected 'w is nil' error, got: %v", err)
		}
	})

	t.Run("Nil offset returns base", func(t *testing.T) {
		base, _ := ParseCIDR("192.168.0.0/16")

		result, err := base.GetSubnetByOffset(nil)
		if err != nil {
			t.Errorf("Expected no error for nil offset, got: %v", err)
		}
		if result != base {
			t.Errorf("Expected same base network, got different result")
		}
	})
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(findSubstring(s, substr) != -1)))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
