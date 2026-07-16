package utils

import (
	"net"
	"strings"
)

func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func IsPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

func SaintizeIP(raw string) string {
	ip := strings.TrimSpace(raw)

	host, _, err := net.SplitHostPort(ip)
	if err == nil {
		return host
	}

	return ip
}

func ExtractIPFromLog(logLine string) string {
	parts := strings.Fields(logLine)

	for i, part := range parts {
		if strings.ToLower(part) == "from" && i+1 < len(parts) {
			candidate := SaintizeIP(parts[i+1])
			if IsValidIP(candidate) {
				return candidate
			}
		}
	}

	return ""
}
