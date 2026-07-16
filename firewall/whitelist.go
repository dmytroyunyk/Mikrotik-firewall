package firewall

import (
	"fmt"
	"net"
)

type Whitelist struct {
	networks []*net.IPNet
	ips      map[string]bool
}

func NewWhitelist(entries []string) (*Whitelist, error) {
	w := &Whitelist{
		ips: make(map[string]bool),
	}

	for _, entry := range entries {
		_, network, err := net.ParseCIDR(entry)
		if err == nil {
			w.networks = append(w.networks, network)
			continue
		}

		ip := net.ParseIP(entry)
		if ip == nil {
			return nil, fmt.Errorf("invalid whitelist entry: %s", entry)
		}

		w.ips[entry] = true
	}

	return w, nil
}

func (w *Whitelist) Contains(ipStr string) bool {
	if w.ips[ipStr] {
		return true
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, network := range w.networks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
