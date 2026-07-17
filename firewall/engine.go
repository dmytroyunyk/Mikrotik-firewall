package firewall

import (
	"fmt"
	"net/netip"
	"sync"
	"time"

	"github.com/dmytroyunyk/mikrotik-defender/config"
	"github.com/dmytroyunyk/mikrotik-defender/mikrotik"
)

type Engine struct {
	client    *mikrotik.Client
	rules     []Rule
	whitelist *Whitelist

	ipv6Thresholds map[int]int

	mu       sync.Mutex
	counters map[string][]time.Time
}

func NewEngine(client *mikrotik.Client, whitelist *Whitelist, cfg config.FirewallConfig) *Engine {
	return &Engine{
		client:    client,
		rules:     DefaultRules(),
		whitelist: whitelist,
		ipv6Thresholds: map[int]int{
			64: cfg.IPv6.Prefix64Threshold,
			56: cfg.IPv6.Prefix56Threshold,
			48: cfg.IPv6.Prefix48Threshold,
		},
		counters: make(map[string][]time.Time),
	}
}

func (e *Engine) ProcessEvent(ip, eventType string) (string, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return "", nil
	}

	if e.whitelist.Contains(addr) {
		return "", nil
	}

	rule, found := e.findRule(eventType)
	if !found {
		return "", nil
	}

	if addr.Is4() || addr.Is4In6() {
		count := e.recordAndCount(addr.String(), eventType, rule.Window)
		if count < rule.Threshold {
			return "", nil
		}
		if err := e.client.BlockIP(ip, rule.BlockReason, 60*time.Minute); err != nil {
			return "", fmt.Errorf("failed to block IP %s: %w", ip, err)
		}
		return ip, nil
	}

	for _, bits := range []int{64, 56, 48} {
		prefix, err := addr.Prefix(bits)
		if err != nil {
			continue
		}
		count := e.recordAndCount(prefix.String(), eventType, rule.Window)
		threshold := e.ipv6Thresholds[bits]

		if threshold <= 0 {
			continue
		}

		if count >= threshold {
			blockTarget := prefix.String()
			if err := e.client.BlockIP(blockTarget, rule.BlockReason, 60*time.Minute); err != nil {
				return "", fmt.Errorf("failed to block %s: %w", blockTarget, err)
			}
			return blockTarget, nil
		}
	}
	return "", nil
}

func (e *Engine) findRule(eventType string) (Rule, bool) {
	for _, rule := range e.rules {
		if rule.EventType == eventType {
			return rule, true
		}
	}
	return Rule{}, false
}

func (e *Engine) recordAndCount(key, eventType string, window time.Duration) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	fullKey := key + ":" + eventType
	now := time.Now()

	e.counters[fullKey] = append(e.counters[fullKey], now)

	cutoff := now.Add(-window)
	var recent []time.Time
	for _, t := range e.counters[fullKey] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	e.counters[fullKey] = recent

	return len(recent)
}
