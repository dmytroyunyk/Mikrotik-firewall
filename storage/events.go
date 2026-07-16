package storage

import (
	"fmt"
	"time"
)

type Event struct {
	ID        int64
	IP        string
	EventType string
	Message   string
	CreatedAt time.Time
}

type BlockedIP struct {
	ID              int64
	IP              string
	Reason          string
	BlockedAt       time.Time
	UnblockedAt     *time.Time
	DurationMinutes int
}

func (db *DB) SaveEvent(ip, eventType, message string) error {
	query := `
		INSERT INTO events (ip, event_type, message)
		VALUES (?, ?, ?)
	`

	_, err := db.conn.Exec(query, ip, eventType, message)
	if err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	return nil
}

func (db *DB) SaveBlockedIP(ip, reason string, duration int) error {
	query := `
		INSERT INTO blocked_ips (ip, reason, duration_minutes)
		VALUES (?, ?, ?)
		ON CONFLICT(ip) DO UPDATE SET
			reason = excluded.reason,
			blocked_at = CURRENT_TIMESTAMP,
			unblocked_at = NULL,
			duration_minutes = excluded.duration_minutes
	`

	_, err := db.conn.Exec(query, ip, reason, duration)
	if err != nil {
		return fmt.Errorf("failed to save blocked IP %s: %w", ip, err)
	}

	return nil
}

func (db *DB) MarkAsUnblocked(ip string) error {
	query := `
		UPDATE blocked_ips
		SET unblocked_at = CURRENT_TIMESTAMP
		WHERE ip = ?
	`

	result, err := db.conn.Exec(query, ip)
	if err != nil {
		return fmt.Errorf("failed to update IP status %s: %w", ip, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check query result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("IP %s not found in database", ip)
	}

	return nil
}

func (db *DB) GetBlockedIPs() ([]BlockedIP, error) {
	query := `
		SELECT id, ip, reason, blocked_at, unblocked_at, duration_minutes
		FROM blocked_ips
		WHERE unblocked_at IS NULL
		ORDER BY blocked_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get blocked IPs: %w", err)
	}
	defer rows.Close()

	var blocked []BlockedIP

	for rows.Next() {
		var b BlockedIP
		err := rows.Scan(
			&b.ID,
			&b.IP,
			&b.Reason,
			&b.BlockedAt,
			&b.UnblockedAt,
			&b.DurationMinutes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		blocked = append(blocked, b)
	}

	return blocked, nil
}

func (db *DB) GetRecentEvents(limit int) ([]Event, error) {
	query := `
		SELECT id, ip, event_type, message, created_at
		FROM events
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	defer rows.Close()

	var events []Event

	for rows.Next() {
		var e Event
		err := rows.Scan(
			&e.ID,
			&e.IP,
			&e.EventType,
			&e.Message,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}
