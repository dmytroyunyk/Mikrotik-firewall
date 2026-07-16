package firewall

import (
	"time"
)

type Rule struct {
	Name        string
	EventType   string
	Threshold   int
	Window      time.Duration
	BlockReason string
}

func DefaultRules() []Rule {
	return []Rule{
		{
			Name:        "SSH Brute Force",
			EventType:   "ssh_brute_force",
			Threshold:   10,
			Window:      60 * time.Second,
			BlockReason: "SSH brute-force attack detected",
		},

		{
			Name:        "Repeated Login Failures",
			EventType:   "login_failed",
			Threshold:   15,
			Window:      120 * time.Second,
			BlockReason: "Multiple failed login attempts",
		},

		{
			Name:        "Port Scan",
			EventType:   "port_scan",
			Threshold:   20,
			Window:      30 * time.Second,
			BlockReason: "Port scanning activity detected",
		},
	}
}

func (r Rule) Matches(count int) bool {
	return count >= r.Threshold
}
