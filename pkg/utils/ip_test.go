package utils

import "testing"

func TestValidIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"1.2.3.4", true},
		{"192.168.1.1", true},
		{"255.255.255.255", true},
		{"0.0.0.0", true},
		{"::1", true},
		{"2001:db8::1", true},
		{"not-an-ip", false},
		{"999.999.999.999", false},
		{"1.2.3", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := IsValidIP(tt.ip)
			if result != tt.expected {
				t.Errorf("IsValidIP(%s) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
		name     string
	}{
		{"192.168.1.1", true, "192.168 network"},
		{"10.0.0.1", true, "10.x network"},
		{"172.16.0.1", true, "172.16 network"},
		{"127.0.0.1", true, "localhost"},
		{"::1", true, "IPv6 localhost"},
		{"1.2.3.4", false, "public IP"},
		{"8.8.8.8", false, "Google DNS"},
		{"invalid", false, "invalid input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPrivateIP(tt.ip)
			if result != tt.expected {
				t.Errorf("IsPrivateIP(%s) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestSantizeIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		name     string
	}{
		{"1.2.3.4", "1.2.3.4", "clean IP"},
		{"  1.2.3.4  ", "1.2.3.4", "IP with spaces"},
		{"1.2.3.4:8080", "1.2.3.4", "IP with port"},
		{"  1.2.3.4:22  ", "1.2.3.4", "IP with port and spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SaintizeIP(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeIP(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractIpFromLog(t *testing.T) {
	tests := []struct {
		log      string
		expected string
		name     string
	}{
		{
			log:      "login failure for user admin from 1.2.3.4 via ssh",
			expected: "1.2.3.4",
			name:     "typical ssh log",
		},
		{
			log:      "connection FROM 192.168.1.100 rejected",
			expected: "192.168.1.100",
			name:     "uppercase FROM",
		},
		{
			log:      "no ip here",
			expected: "",
			name:     "no ip in log",
		},
		{
			log:      "",
			expected: "",
			name:     "empty log",
		},
		{
			log:      "attack from invalid-text",
			expected: "",
			name:     "from with invalid ip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractIPFromLog(tt.log)
			if result != tt.expected {
				t.Errorf("ExtractIPFromLog(%q) = %q, expected %q", tt.log, result, tt.expected)
			}
		})
	}
}
