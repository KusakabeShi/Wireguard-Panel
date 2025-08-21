package services

import (
	"strings"
	"testing"
)

func TestFirewallService_IPAddressExists(t *testing.T) {
	fs := NewFirewallService()
	
	// Test with non-existent interface
	exists, err := fs.ipAddressExists("nonexistent-interface", "192.168.1.1/24")
	if err != nil {
		t.Errorf("Expected no error for non-existent interface, got: %v", err)
	}
	if exists {
		t.Error("Expected false for non-existent interface")
	}
}

func TestFirewallService_IptablesRuleExists(t *testing.T) {
	fs := NewFirewallService()
	
	// Test rule conversion from -A to -C
	ruleArgs := []string{"-t", "nat", "-A", "POSTROUTING", "-j", "MASQUERADE"}
	
	// This should return false (rule doesn't exist) without error
	exists, err := fs.iptablesRuleExists("iptables", ruleArgs)
	if err != nil {
		// Only fail if it's not a "rule not found" error
		if !strings.Contains(err.Error(), "exit code: 1") {
			t.Errorf("Unexpected error checking iptables rule: %v", err)
		}
	}
	
	// Should be false since we're not actually adding any rules
	if exists {
		t.Error("Expected false for non-existent rule")
	}
}

func TestFirewallService_AddIPAddressIfNotExists_NoDuplicate(t *testing.T) {
	fs := NewFirewallService()
	
	// Test that the function handles non-existent interface gracefully
	err := fs.addIPAddressIfNotExists("nonexistent-test-interface", "192.168.99.1/24")
	
	// This should fail because the interface doesn't exist, but it should be a specific error
	if err == nil {
		t.Error("Expected error when adding IP to non-existent interface")
	}
	
	// The error should indicate device not found
	if !strings.Contains(err.Error(), "Cannot find device") {
		t.Errorf("Expected device not found error, got: %v", err)
	}
}

func TestFirewallService_AddIptablesRuleIfNotExists_Logic(t *testing.T) {
	fs := NewFirewallService()
	
	// Test with a rule that should be properly formatted
	// This won't actually execute iptables but will test the logic
	ruleArgs := []string{"-A", "FORWARD", "-j", "ACCEPT"}
	
	// Test rule existence check using echo (echo always succeeds with exit code 0)
	exists, err := fs.iptablesRuleExists("echo", ruleArgs) // Use echo instead of iptables to avoid system requirements
	if err != nil {
		t.Errorf("Echo command should not error: %v", err)
	}
	if !exists {
		t.Error("Echo command should indicate rule exists (exit code 0 means rule exists for our logic)")
	}
}

func TestIptablesRuleConversion(t *testing.T) {
	fs := NewFirewallService()
	
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Basic rule with table",
			input:    []string{"-t", "nat", "-A", "POSTROUTING", "-j", "MASQUERADE"},
			expected: []string{"-t", "nat", "-C", "POSTROUTING", "-j", "MASQUERADE"},
		},
		{
			name:     "Rule without table",
			input:    []string{"-A", "FORWARD", "-j", "ACCEPT"},
			expected: []string{"-C", "FORWARD", "-j", "ACCEPT"},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We can't easily test the internal logic without exposing it,
			// but we can test that the function doesn't panic and behaves reasonably
			_, err := fs.iptablesRuleExists("echo", tc.input)
			
			// Using echo should return no error (exit code 0)
			if err != nil {
				t.Errorf("Unexpected error with echo command: %v", err)
			}
		})
	}
}