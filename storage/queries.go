package storage

import (
	"fmt"
	"time"
)

type AttackStat struct {
	IP          string
	AttackCount int
	LastSeen    time.Time
	IsBlocked   bool
}

func (db *DB) GetTopAttackers(limit int) ([]AttackStat, error) {
	query := `
		SELECT
			e.ip,
			COUNT(e.id)        as attack_count,
			MAX(e.created_at)  as last_seen,
			CASE WHEN b.ip IS NOT NULL THEN 1 ELSE 0 END as is_blocked
		FROM events e
		LEFT JOIN blocked_ips b ON e.ip = b.ip AND b.unblocked_at IS NULL
		GROUP BY e.ip
		ORDER BY attack_count DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top attackers: %w", err)
	}
	defer rows.Close()

	var stats []AttackStat

	for rows.Next() {
		var s AttackStat
		var isBlocked int

		err := rows.Scan(
			&s.IP,
			&s.AttackCount,
			&s.LastSeen,
			&isBlocked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attacker stat: %w", err)
		}

		s.IsBlocked = isBlocked == 1
		stats = append(stats, s)
	}

	return stats, nil
}

func (db *DB) GetAttackCount(ip string, since time.Time) (int, error) {
	query := `
		SELECT COUNT(id)
		FROM events
		WHERE ip = ?
		AND created_at >= ?
	`

	var count int

	err := db.conn.QueryRow(query, ip, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get attack count for IP %s: %w", ip, err)
	}

	return count, nil
}

func (db *DB) IsIPBlocked(ip string) (bool, error) {
	query := `
		SELECT COUNT(id)
		FROM blocked_ips
		WHERE ip = ?
		AND unblocked_at IS NULL
	`

	var count int

	err := db.conn.QueryRow(query, ip).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if IP %s is blocked: %w", ip, err)
	}

	return count > 0, nil
}

func (db *DB) GetStats() (map[string]int, error) {
	stats := make(map[string]int)

	var totalEvents int
	err := db.conn.QueryRow(`SELECT COUNT(id) FROM events`).Scan(&totalEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get total events: %w", err)
	}
	stats["total_events"] = totalEvents

	var blockedCount int
	err = db.conn.QueryRow(`
		SELECT COUNT(id) FROM blocked_ips WHERE unblocked_at IS NULL
	`).Scan(&blockedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get blocked count: %w", err)
	}
	stats["blocked_ips"] = blockedCount

	var recentEvents int
	err = db.conn.QueryRow(`
		SELECT COUNT(id) FROM events
		WHERE created_at >= datetime('now', '-24 hours')
	`).Scan(&recentEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events count: %w", err)
	}
	stats["events_24h"] = recentEvents

	return stats, nil
}
