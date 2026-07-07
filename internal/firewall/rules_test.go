package firewall

import (
	"testing"
	"time"
)

func TestDefaultRules(t *testing.T) {
	rules := DefaultRules()

	if len(rules) == 0 {
		t.Fatal("expecteed at least one rule, got 0")
	}
}

func TestDefaultRules_RequiredTypes(t *testing.T) {
	rules := DefaultRules()

	required := []string{
		"ssh_brute_forse",
		"login_failed",
		"port_scan",
	}

	for _, eventType := range required {
		found := false
		for _, rule := range rules {
			if rule.EventType == eventType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected rule for event type %s, but not found", eventType)
		}
	}
}

func TestRule_Mathces(t *testing.T) {
	rule := Rule{
		Name:        "Test Rule",
		EventType:   "ssh_brute_force",
		Threshold:   10,
		Window:      60 * time.Second,
		BlockReason: "test",
	}

	tests := []struct {
		count    int
		expected bool
		name     string
	}{
		{0, false, "zero events"},
		{5, false, "below threshold"},
		{9, false, "one below threshold"},
		{10, true, "exactly at threshold"},
		{11, true, "above threshold"},
		{100, true, "way above threshold"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Matches(tt.count)
			if result != tt.expected {
				t.Errorf("Matches(%d) = %v, expected %v", tt.count, result, tt.expected)
			}
		})
	}
}

func TestRule_Fileds(t *testing.T) {
	rules := DefaultRules()

	for _, rule := range rules {
		t.Run(rule.Name, func(t *testing.T) {
			if rule.Name == "" {
				t.Error("rule Name cannot be empty")
			}
			if rule.EventType == "" {
				t.Error("rule EventType cannot be empty")
			}
			if rule.Threshold <= 0 {
				t.Error("rule Threshold must be greater than 0")
			}
			if rule.Window <= 0 {
				t.Error("rule Window must be greater than 0")
			}
			if rule.BlockReason == "" {
				t.Error("rule BlockReason cannot be empty")
			}
		})
	}
}
