package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	MikroTik MikroTikConfig `yaml:"mikrotik"`
	Firewall FirewallConfig `yaml:"firewall"`
	Telegram TelegramConfig `yaml:"telegram"`
	Storage  StorageConfig  `yaml:"storage"`
	Log      LogConfig      `yaml:"log"`
}

type MikroTikConfig struct {
	Address  string `yaml:"address"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type FirewallConfig struct {
	BanThreshold int      `yaml:"ban_threshold"`
	BanDuration  int      `yaml:"ban_duration_minutes"`
	Whitelist    []string `yaml:"whitelist"`
}

type TelegramConfig struct {
	Token  string `yaml:"token"`
	ChatID string `yaml:"chat_id"`
}

type StorageConfig struct {
	Path string `yaml:"path"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("can't read config file %s: %w", path, err)
	}

	cfg := &Config{}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("can't unparsing config: %w", err)
	}

	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		cfg.Telegram.Token = token
	}
	if chatID := os.Getenv("TELEGRAM_CHAT_ID"); chatID != "" {
		cfg.Telegram.ChatID = chatID
	}
	if password := os.Getenv("MIKROTIK_PASSWORD"); password != "" {
		cfg.MikroTik.Password = password
	}

	err = cfg.validate()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.MikroTik.Address == "" {
		return fmt.Errorf("config: mikrotik.address cannot be empty")
	}
	if c.MikroTik.Username == "" {
		return fmt.Errorf("config: mikrotik.username cannot be empty")
	}
	if c.MikroTik.Password == "" {
		return fmt.Errorf("config: mikrotik.password нcannot be empty")
	}
	if c.Telegram.Token == "" {
		return fmt.Errorf("config: telegram.token cannot be empty")
	}
	if c.Telegram.ChatID == "" {
		return fmt.Errorf("config: telegram.chat_id cannot be empty")
	}
	if c.Storage.Path == "" {
		return fmt.Errorf("config: storage.path cannot be empty")
	}
	if c.Firewall.BanThreshold <= 0 {
		return fmt.Errorf("config: firewall.ban_threshold must be greater than 0")
	}
	return nil
}
