package firewall

import (
	"fmt"
	"net/netip"
)

type Whitelist struct {
	networks []netip.Prefix
	ips      map[netip.Addr]bool
}

func NewWhitelist(entries []string) (*Whitelist, error) {
	w := &Whitelist{
		ips: make(map[netip.Addr]bool),
	}

	for _, entry := range entries {
		if prefix, err := netip.ParsePrefix(entry); err == nil {
			w.networks = append(w.networks, prefix)
			continue
		}

		addr, err := netip.ParseAddr(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid whitelist entery %q: %w", entry, err)
		}
		w.ips[addr] = true
	}

	return w, nil
}

func (w *Whitelist) Contains(ip netip.Addr) bool {
	if w.ips[ip] {
		return true
	}

	for _, n := range w.networks {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}
