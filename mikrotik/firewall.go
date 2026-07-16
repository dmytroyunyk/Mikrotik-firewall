package mikrotik

import (
	"fmt"
	"time"
)

type BlockedEntry struct {
	IP        string
	Reason    string        // SSH brute-force
	BlockedAt time.Time     // when blocked
	Duration  time.Duration // how long was it blocked
}

func (c *Client) BlockIP(address, reason string, duration time.Duration) error {

	if !c.IsConnect() {
		return fmt.Errorf("no connection to router")
	}

	timeout := time.Now().Add(duration).Format("15:04:05")

	_, err := c.conn.Run(
		"/ip/firewall/address-list/add", // command mikroTik
		"=list=blacklist",
		"=address="+address, // IP add
		"=comment="+reason,  // comment (problem)
		"=timeout="+timeout, // when unblock
	)
	if err != nil {
		return fmt.Errorf("cant block ip %s: %w", address, err)
	}
	return nil
}

func (c *Client) Unblock(address string) error {
	if !c.IsConnect() {
		return fmt.Errorf("don't connected to router")
	}

	reply, err := c.conn.Run(
		"/ip/firewall/address-list/print",
		"?list=blacklist",   // blacklist
		"?address="+address, // IPs
		"=.proplist=.id",    // return ID
	)
	if err != nil {
		return fmt.Errorf("can't find IP %s in blacklist: %w", address, err)
	}

	if len(reply.Re) == 0 {
		return fmt.Errorf("IP %s don't find in blacklist", address)
	}

	id := reply.Re[0].Map[".id"]

	_, err = c.conn.Run(
		"/ip/firewall/address-list/remove",
		"=.id="+id,
	)

	if err != nil {
		return fmt.Errorf("can't unblock IP %s: %w", address, err)
	}

	return nil
}
func (c *Client) GetBlockedIPs() ([]BlockedEntry, error) {
	if !c.IsConnect() {
		return nil, fmt.Errorf("don't connect to router")
	}

	reply, err := c.conn.Run(
		"/ip/firewall/address-list/print",
		"?list=blacklist",
	)

	if err != nil {
		return nil, fmt.Errorf("can't get list blocked IP: %w", err)
	}

	entries := make([]BlockedEntry, 0, len(reply.Re))

	for _, re := range reply.Re {
		entry := BlockedEntry{
			IP:        re.Map["address"],
			Reason:    re.Map["comment"],
			BlockedAt: time.Now(),
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
