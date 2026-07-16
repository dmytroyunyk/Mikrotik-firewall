package firewall

import (
	"testing"
)

func TestNewWhitelist_ValidEntires(t *testing.T) {
	_, err := NewWhitelist([]string{
		"192.168.88.0/24",
		"127.0.0.1",
		"10.0.0.0/8",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestNewWhitelsit_InvalidEntry(t *testing.T) {
	_, err := NewWhitelist([]string{
		"not-an-ip",
	})

	if err == nil {
		t.Fatal("expected error for invalid entry, got nil")
	}
}

func TestWhitelist_Contains_SingleIP(t *testing.T) {
	w, err := NewWhitelist([]string{"127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}

	if !w.Contains("127.0.0.1") {
		t.Error("expected 127.0.0.1 to be whitelisted")
	}

	if w.Contains("1.2.3.4") {
		t.Error("expected 1.2.3.4 to not be whitelisted")
	}
}

func TestWhitelist_Contains_CIDRRange(t *testing.T) {
	w, err := NewWhitelist([]string{"192.168.88.0/24"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		ip       string
		expected bool
	}{
		{"192.168.88.1", true},
		{"192.168.88.55", true},
		{"192.168.88.255", true},
		{"192.168.89.1", false},
		{"10.0.0.1", false},
		{"1.2.3.4", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := w.Contains(tt.ip)
			if result != tt.expected {
				t.Errorf("contains(%s) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestWhitelist_Contatins_EmptyWhitelist(t *testing.T) {
	w, err := NewWhitelist([]string{})
	if err != nil {
		t.Fatal(err)
	}

	if w.Contains("192.168.88.1") {
		t.Error("empty whitelist should not contain any IP")
	}
}
