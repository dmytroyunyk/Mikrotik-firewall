package storage

import "testing"

func setupTestDB(t *testing.T) *DB {
	dir := t.TempDir()
	path := dir + "/test.db"

	db, err := New(path)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestSaveEvent(t *testing.T) {
	db := setupTestDB(t)

	err := db.SaveEvent("1.2.3.4", "ssh_brute_force", "test message")
	if err != nil {
		t.Fatalf("failed to save event: %v", err)
	}

	events, err := db.GetRecentEvents(10)
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].IP != "1.2.3.4" {
		t.Errorf("wrong IP: %s", events[0].IP)
	}

	if events[0].EventType != "ssh_brute_force" {
		t.Errorf("wrong EventType: %s", events[0].EventType)
	}
}

func TestSaveBlockedIP(t *testing.T) {
	db := setupTestDB(t)

	err := db.SaveBlockedIP("1.2.3.4", "brute force", 60)
	if err != nil {
		t.Fatalf("failed to save blocked IP: %v", err)
	}

	blocked, err := db.GetBlockedIPs()
	if err != nil {
		t.Fatalf("failed to get blocked IPs: %v", err)
	}

	if len(blocked) != 1 {
		t.Fatalf("expected 1 blocked IP, got %d", len(blocked))
	}

	if blocked[0].IP != "1.2.3.4" {
		t.Errorf("wrong IP: %s", blocked[0].IP)
	}
	if blocked[0].Reason != "brute force" {
		t.Errorf("wrong Reason: %s", blocked[0].Reason)
	}
	if blocked[0].DurationMinutes != 60 {
		t.Errorf("wrong DurationMinutes: %d", blocked[0].DurationMinutes)
	}
}

func TestSaveBlockedIP_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	db.SaveBlockedIP("1.2.3.4", "reason 1", 30)

	db.SaveBlockedIP("1.2.3.4", "reason 2", 60)

	blocked, _ := db.GetBlockedIPs()

	if len(blocked) != 1 {
		t.Fatalf("expected 1 blocked IP (upsert), got %d", len(blocked))
	}

	if blocked[0].Reason != "reason 2" {
		t.Errorf("expected updated reason, got: %s", blocked[0].Reason)
	}
	if blocked[0].DurationMinutes != 60 {
		t.Errorf("expected updated duration, got: %d", blocked[0].DurationMinutes)
	}
}

func TestMarkAsUnblocked(t *testing.T) {
	db := setupTestDB(t)

	db.SaveBlockedIP("1.2.3.4", "test", 60)

	err := db.MarkAsUnblocked("1.2.3.4")
	if err != nil {
		t.Fatalf("failed to unblock: %v", err)
	}

	blocked, _ := db.GetBlockedIPs()

	if len(blocked) != 0 {
		t.Errorf("expected 0 active blocked IPs after unblock, got %d", len(blocked))
	}
}

func TestMarkAsUnblocked_NotFound(t *testing.T) {
	db := setupTestDB(t)

	err := db.MarkAsUnblocked("non-existent-ip")
	if err == nil {
		t.Error("expected error for non-existent IP")
	}
}

func TestIsIPBlocked(t *testing.T) {
	db := setupTestDB(t)

	blocked, err := db.IsIPBlocked("1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocked {
		t.Error("IP should not be blocked initially")
	}

	db.SaveBlockedIP("1.2.3.4", "test", 60)

	blocked, _ = db.IsIPBlocked("1.2.3.4")
	if !blocked {
		t.Error("IP should be blocked")
	}

	db.MarkAsUnblocked("1.2.3.4")

	blocked, _ = db.IsIPBlocked("1.2.3.4")
	if blocked {
		t.Error("IP should not be blocked after unblock")
	}
}

func TestGetStats(t *testing.T) {
	db := setupTestDB(t)

	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats["total_events"] != 0 {
		t.Errorf("expected 0 total_events, got %d", stats["total_events"])
	}

	db.SaveEvent("1.1.1.1", "ssh_brute_force", "msg1")
	db.SaveEvent("2.2.2.2", "port_scan", "msg2")
	db.SaveBlockedIP("1.1.1.1", "test", 60)

	stats, _ = db.GetStats()

	if stats["total_events"] != 2 {
		t.Errorf("expected 2 total_events, got %d", stats["total_events"])
	}
	if stats["blocked_ips"] != 1 {
		t.Errorf("expected 1 blocked_ips, got %d", stats["blocked_ips"])
	}
}
