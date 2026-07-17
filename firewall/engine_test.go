package firewall

import (
	"net/netip"
	"testing"
	"time"
)

func TestEngine_ProcessEvent_WhiteListed(t *testing.T) {
	Whitelist, _ := NewWhitelist([]string{"192.168.88.0/24"})

	engine := NewEngine(nil, Whitelist)

	for i := 0; i < 100; i++ {
		blockedIP, err := engine.ProcessEvent("192.168.88.55", "ssh_brute_force")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if blockedIP != "" {
			t.Errorf("whitelisted IP should never be blocked, got: %s", blockedIP)
		}
	}
}

func TestEngine_ProcessEvent_UnkownEventType(t *testing.T) {
	Whitelist, _ := NewWhitelist([]string{})
	engine := NewEngine(nil, Whitelist)

	blockedIP, err := engine.ProcessEvent("1.2.3.4", "unknown_event_type")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if blockedIP != "" {
		t.Errorf("unknown event type should not trigger block, got: %s", blockedIP)
	}
}

func TestEngine_findRule(t *testing.T) {
	Whitelist, _ := NewWhitelist([]string{})
	engine := NewEngine(nil, Whitelist)

	rule, found := engine.findRule("ssh_brute_force")
	if !found {
		t.Error("expected to find ssh_brute_force rule")
	}
	if rule.EventType != "ssh_brute_force" {
		t.Errorf("wrong rule returned: %s", rule.EventType)
	}

	_, found = engine.findRule("non_existend_type")
	if found {
		t.Error("should not find non-existent_type")
	}
}

func TestEngine_recordAndCount(t *testing.T) {
	Whitelist, _ := NewWhitelist([]string{})
	engine := NewEngine(nil, Whitelist)

	addr := netip.MustParseAddr("1.2.3.4")
	eventType := "ssh_brute_force"
	window := 60 * time.Second

	for i := 1; i <= 5; i++ {
		count := engine.recordAndCount(addr, eventType, window)
		if count != i {
			t.Errorf("iteration %d: expected count %d, got %d", i, i, count)
		}
	}
}

func TestEngine_recordAndCount_DifferentIPs(t *testing.T) {
	Whitelist, _ := NewWhitelist([]string{})
	engine := NewEngine(nil, Whitelist)

	window := 60 * time.Second

	ip1 := netip.MustParseAddr("1.1.1.1")
	ip2 := netip.MustParseAddr("2.2.2.2")

	engine.recordAndCount(ip1, "ssh_brute_force", window)
	engine.recordAndCount(ip1, "ssh_brute_force", window)
	engine.recordAndCount(ip2, "ssh_brute_force", window)

	count1 := engine.recordAndCount(ip1, "ssh_brute_force", window)
	if count1 != 3 {
		t.Errorf("IP 1.1.1.1: expected count 3, got %d", count1)
	}

	count2 := engine.recordAndCount(ip2, "ssh_brute_force", window)
	if count2 != 2 {
		t.Errorf("IP 2.2.2.2: expected count 2, got %d", count2)
	}
}

func TestEngine_recordAndCount_OldEventDropped(t *testing.T) {
	Whitelist, _ := NewWhitelist([]string{})
	engine := NewEngine(nil, Whitelist)

	addr := netip.MustParseAddr("1.2.3.4")
	eventType := "ssh_brute_force"

	shortWindow := 10 * time.Millisecond

	engine.recordAndCount(addr, eventType, shortWindow)

	time.Sleep(20 * time.Millisecond)

	count := engine.recordAndCount(addr, eventType, shortWindow)

	if count != 1 {
		t.Errorf("expected count 1 (old event dropped), got %d", count)
	}
}

func TestEngine_recordAndCount_IPv6Aggregation(t *testing.T) {
	Whitelist, _ := NewWhitelist([]string{})
	engine := NewEngine(nil, Whitelist)

	window := 60 * time.Second

	addr1 := netip.MustParseAddr("2001:db8::1")
	addr2 := netip.MustParseAddr("2001:db8::2")
	addr3 := netip.MustParseAddr("2001:db8::ffff")

	engine.recordAndCount(addr1, "ssh_brute_forse", window)
	engine.recordAndCount(addr2, "ssh_brute_forse", window)
	count := engine.recordAndCount(addr3, "ssh_brute_forse", window)

	if count != 3 {
		t.Errorf("expected 3 aggregated by /64, got %d", count)
	}
}

func TestEngine_recordAndCount_IPv6DiferentPrefix(t *testing.T) {
	Whitelist, _ := NewWhitelist([]string{})
	engine := NewEngine(nil, Whitelist)

	window := 60 * time.Second

	addr1 := netip.MustParseAddr("2001:db8:1::1")
	addr2 := netip.MustParseAddr("2001:db8:2::1")

	engine.recordAndCount(addr1, "shh_brute_forse", window)
	count := engine.recordAndCount(addr2, "ssh_brute_forse", window)

	if count != 1 {
		t.Errorf("expected 1 diferent /64 prefixes, got %d", count)
	}
}
