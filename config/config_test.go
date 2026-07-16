package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	content := `
mikrotik:
  address: "192.168.88.1:8728"
  username: "admin"
  password: "test-password"

firewall:
  ban_threshold: 10
  ban_duration_minutes: 60
  whitelist:
    - "192.168.88.0/24"

telegram:
  token: "test-token"
  chat_id: "123456"

storage:
  path: "./data/test.db"

api:
  port: 8080
  key: "test-key"

log:
  level: "info"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.MikroTik.Address != "192.168.88.1:8728" {
		t.Errorf("wrong Address: %s", cfg.MikroTik.Address)
	}
	if cfg.Firewall.BanThreshold != 10 {
		t.Errorf("wrong BanThreshold: %d", cfg.Firewall.BanThreshold)
	}
	if cfg.Telegram.Token != "test-token" {
		t.Errorf("wrong Token: %s", cfg.Telegram.Token)
	}
	if cfg.API.Port != 8080 {
		t.Errorf("wrong API Port: %d", cfg.API.Port)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yml")

	if err == nil {
		t.Fatal("expected error for non-exist file, got nil")
	}
}

func TestLoad_InvalidYML(t *testing.T) {
	content := `this is not valid yml: [[[`

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err = Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YML, got nil")
	}
}

func TestLoad_MissingRequiredFailed(t *testing.T) {
	content := `
mikrotik:
  address: "192.168.88.1:8728"
  username: "admin"
  password: ""

firewall:
  ban_threshold: 10

telegram:
  token: "test-token"
  chat_id: "123456"

storage:
  path: "./data/test.db"

log:
  level: "info"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	os.Unsetenv("MIKROTIK_PASSWORD")

	_, err = Load(path)
	if err == nil {
		t.Fatal("expected validation error for missing password, got nil")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	content := `
mikrotik:
  address: "192.168.88.1:8728"
  username: "admin"
  password: "from-config"

firewall:
  ban_threshold: 10
  ban_duration_minutes: 60
  whitelist:
    - "127.0.0.1"

telegram:
  token: "from-config"
  chat_id: "123"

storage:
  path: "./data/test.db"

log:
  level: "info"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	err := os.WriteFile(path, []byte(content), 0664)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	os.Setenv("MIKROTIK_PASSWORD", "from-env")
	os.Setenv("TELEGRAM_TOKEN", "env-token")
	defer os.Unsetenv("MIKROTIK_PASSWORD")
	defer os.Unsetenv("TELEGRAM_TOKEN")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MikroTik.Password != "from-env" {
		t.Errorf("expected password from env, got: %s", cfg.MikroTik.Password)
	}
	if cfg.Telegram.Token != "env-token" {
		t.Errorf("expected token from env, got: %s", cfg.Telegram.Token)
	}
}
