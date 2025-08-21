package services

import (
	"testing"
	"time"
	"wg-panel/internal/models"
	"github.com/vishvananda/netlink"
)

func TestInterfaceRobustness(t *testing.T) {
	// Test that the service handles nil interfaces gracefully
	listener := NewInterfaceIPNetListener("nonexistent-interface", []*SNATRoamingConfig{})
	
	// This should not panic
	listener.syncFirewallRules()
	
	// Test with nil link
	result := listener.isOurInterface(nil)
	if result {
		t.Error("Should return false for nil interface")
	}
	
	// Test address update with invalid link index
	addrUpdate := netlink.AddrUpdate{LinkIndex: 999999} // Non-existent interface
	result = listener.isOurInterfaceAddr(addrUpdate)
	if result {
		t.Error("Should return false for non-existent interface")
	}
}

func TestChannelClosureRecovery(t *testing.T) {
	configs := []*SNATRoamingConfig{
		{
			Version: 4,
			ServerNetwork: &models.IPNetWrapper{},
			CommentString: "test",
		},
	}
	
	listener := NewInterfaceIPNetListener("lo", configs)
	
	// Test stopping multiple times (should not panic)
	listener.Stop()
	listener.Stop() // Second stop should be safe
}

func TestNilConfigHandling(t *testing.T) {
	// Test with nil config
	configs := []*SNATRoamingConfig{nil, &SNATRoamingConfig{}}
	listener := NewInterfaceIPNetListener("lo", configs)
	
	// Should not panic with nil configs
	listener.syncFirewallRules()
}

// This test ensures robustness in real scenarios
func TestPanicRecovery(t *testing.T) {
	listener := NewInterfaceIPNetListener("test-interface", []*SNATRoamingConfig{
		{
			Version: 4,
			ServerNetwork: &models.IPNetWrapper{},
			CommentString: "test-comment",
		},
	})

	// Test that panic recovery works
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Unrecovered panic: %v", r)
			}
			done <- true
		}()
		
		// This will likely trigger some error conditions but should not panic
		listener.syncFirewallRules()
	}()
	
	select {
	case <-done:
		// Test completed successfully
	case <-time.After(5 * time.Second):
		t.Error("Test timed out - possible deadlock or infinite loop")
	}
}