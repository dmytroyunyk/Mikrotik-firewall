package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmytroyunyk/mikrotik-defender/internal/bot"
	"github.com/dmytroyunyk/mikrotik-defender/internal/config"
	"github.com/dmytroyunyk/mikrotik-defender/internal/firewall"
	"github.com/dmytroyunyk/mikrotik-defender/internal/mikrotik"
	"github.com/dmytroyunyk/mikrotik-defender/internal/storage"
	"github.com/dmytroyunyk/mikrotik-defender/pkg/utils"
)

func main() {
	cfg, err := config.Load("configs/config.yml")
	if err != nil {
		logger := utils.NewLogger("info")
		logger.Fatal("failed to load config", "error", err)
	}

	logger := utils.NewLogger(cfg.Log.Level)
	logger.Info("starting Mikrotik Intelligent Defender")

	client := mikrotik.NewClient(
		cfg.MikroTik.Address,
		cfg.MikroTik.Username,
		cfg.MikroTik.Password,
	)
	if err := client.Connect(); err != nil {
		logger.Fatal("failed to connect to MikroTik", "error", err)
	}
	defer client.Disconnect()
	logger.Info("connected to MikroTik", "address", cfg.MikroTik.Address)

	db, err := storage.New(cfg.Storage.Path)
	if err != nil {
		logger.Fatal("failed to open database", "error", err)
	}
	defer db.Close()
	logger.Info("database opened", "path", cfg.Storage.Path)

	whitelist, err := firewall.NewWhitelist(cfg.Firewall.Whitelist)
	if err != nil {
		logger.Fatal("failed to create whitelist", "error", err)
	}
	logger.Info("whitelist loaded", "entries", len(cfg.Firewall.Whitelist))

	engine := firewall.NewEngine(client, whitelist)
	logger.Info("firewall engine initialized")

	teleBot, err := bot.New(cfg, db, client, logger)
	if err != nil {
		logger.Fatal("failed to create telegram bot", "error", err)
	}
	teleBot.Start()
	defer teleBot.Stop()

	if err := teleBot.NotifyStartup(); err != nil {
		logger.Error("failed to send startup notification", "error", err)
	}

	watcher := mikrotik.NewWatcher(client, 100)
	events, err := watcher.Start()
	if err != nil {
		logger.Fatal("failed to start watcher", "error", err)
	}
	defer watcher.Stop()
	logger.Info("watcher started, monitoring router logs")

	go func() {
		for event := range events {
			blockedIP, err := engine.ProcessEvent(event.IP, string(event.EventType))
			if err != nil {
				logger.Error("failed to process event",
					"ip", event.IP,
					"event_type", event.EventType,
					"error", err,
				)
				teleBot.NotifyError("firewall engine", err.Error())
				continue
			}

			if blockedIP != "" {
				logger.Info("IP blocked",
					"ip", blockedIP,
					"event_type", event.EventType,
				)

				if err := db.SaveEvent(blockedIP, string(event.EventType), event.Message); err != nil {
					logger.Error("failed to save event", "error", err)
				}

				if err := db.SaveBlockedIP(blockedIP, event.Message, cfg.Firewall.BanDuration); err != nil {
					logger.Error("failed to save blocked IP", "error", err)
				}

				if err := teleBot.NotifyBlocked(blockedIP, event.Message, time.Duration(cfg.Firewall.BanDuration)*time.Minute); err != nil {
					logger.Error("failed to send block notification", "error", err)
				}
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("system is running, press Ctrl+C to stop")
	<-quit

	logger.Info("shutting down...")
	if err := teleBot.NotifyShutdown(); err != nil {
		logger.Error("failed to send shutdown notification", "error", err)
	}
	logger.Info("system stopped")
}
