package firewall

import (
	"fmt"
	"sync"
	"time"

	"github.com/dmytroyunyk/mikrotik-defender/mikrotik"
)

type Engine struct {
	client    *mikrotik.Client
	rules     []Rule
	whitelist *Whitelist

	mu       sync.Mutex
	counters map[string][]time.Time
}

func NewEngine(client *mikrotik.Client, whitelist *Whitelist) *Engine {
	return &Engine{
		client:    client,
		rules:     DefaultRules(),
		whitelist: whitelist,
		counters:  make(map[string][]time.Time),
	}
}

func (e *Engine) ProcessEvent(ip, eventType string) (string, error) {
	if e.whitelist.Contains(ip) {
		return "", nil
	}

	rule, found := e.findRule(eventType)
	if !found {
		return "", nil
	}

	count := e.recordAndCount(ip, eventType, rule.Window)

	if !rule.Matches(count) {
		return "", nil
	}

	err := e.client.BlockIP(ip, rule.BlockReason, 60*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to block IP %s: %w", ip, err)
	}

	return ip, nil
}

func (e *Engine) findRule(eventType string) (Rule, bool) {
	for _, rule := range e.rules {
		if rule.EventType == eventType {
			return rule, true
		}
	}
	return Rule{}, false
}

func (e *Engine) recordAndCount(ip, eventType string, window time.Duration) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := ip + ":" + eventType
	now := time.Now()

	e.counters[key] = append(e.counters[key], now)

	cutoff := now.Add(-window)
	var recent []time.Time
	for _, t := range e.counters[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	e.counters[key] = recent

	return len(recent)
}
