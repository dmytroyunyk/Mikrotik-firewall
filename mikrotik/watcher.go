package mikrotik

import (
	"fmt"
	"strings"
	"time"
)

type EventType string

const (
	EventSSHBruteForce EventType = "ssh_brute_force"
	EventPortScan      EventType = "port_scan"
	EventLoginFailed   EventType = "login_failed"
	EventUnknown       EventType = "unknown"
)

type LogEntry struct {
	Time      time.Time
	IP        string
	EventType EventType
	Message   string
}

type Watcher struct {
	client   *Client
	events   chan LogEntry
	stopChan chan struct{}
}

func NewWatcher(client *Client, bufferSize int) *Watcher {
	return &Watcher{
		client:   client,
		events:   make(chan LogEntry, bufferSize),
		stopChan: make(chan struct{}),
	}
}

func (w *Watcher) Start() (<-chan LogEntry, error) {
	if !w.client.IsConnect() {
		return nil, fmt.Errorf("watcher: can't connect to router")
	}

	go w.watch()

	return w.events, nil
}

func (w *Watcher) Stop() {
	close(w.stopChan)
}

func (w *Watcher) watch() {
	listen, err := w.client.conn.Listen("/log/print")
	if err != nil {
		return
	}

	for {
		select {
		case <-w.stopChan:
			return

		case sentence := <-listen.Chan():
			if sentence == nil {
				return
			}

			message := sentence.Map["message"]
			topics := sentence.Map["topics"]

			entry, found := w.parseLogEntry(message, topics)
			if found {
				w.events <- entry
			}
		}
	}
}

func (w *Watcher) parseLogEntry(message, topics string) (LogEntry, bool) {
	msgLower := strings.ToLower(message)

	var eventType EventType
	var ip string

	switch {
	case strings.Contains(msgLower, "ssh") && strings.Contains(msgLower, "login failure"):
		eventType = EventSSHBruteForce
		ip = extractIP(message)

	case strings.Contains(msgLower, "login failure"):
		eventType = EventLoginFailed
		ip = extractIP(message)

	case strings.Contains(topics, "firewall") && strings.Contains(msgLower, "forward"):
		eventType = EventPortScan
		ip = extractIP(message)

	default:
		return LogEntry{}, false
	}

	if ip == "" {
		return LogEntry{}, false
	}

	return LogEntry{
		Time:      time.Now(),
		IP:        ip,
		EventType: eventType,
		Message:   message,
	}, true
}

func extractIP(message string) string {
	parts := strings.Fields(message)

	for i, part := range parts {
		if strings.ToLower(part) == "from" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}
