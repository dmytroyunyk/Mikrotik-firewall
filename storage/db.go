package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

func New(path string) (*DB, error) {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("could not create folder for db %s: %w", dir, err)
	}

	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	err = conn.Ping()
	if err != nil {
		return nil, fmt.Errorf("coild not connect to db: %w", err)
	}

	db := &DB{conn: conn}

	err = db.createTables()
	if err != nil {
		return nil, fmt.Errorf("could not create tables: %w", err)
	}

	return db, nil
}

func (db *DB) createTables() error {
	query := `
    CREATE TABLE IF NOT EXISTS events (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        ip          TEXT NOT NULL,
        event_type  TEXT NOT NULL,
        message     TEXT,
        created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS blocked_ips (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        ip          TEXT NOT NULL UNIQUE,
        reason      TEXT NOT NULL,
        blocked_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
        unblocked_at DATETIME,
        duration_minutes INTEGER NOT NULL DEFAULT 60
    );

    CREATE TABLE IF NOT EXISTS stats (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        ip          TEXT NOT NULL,
        attack_count INTEGER NOT NULL DEFAULT 1,
        last_seen   DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    `
	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("error create tables")
	}
	return nil
}

func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}
