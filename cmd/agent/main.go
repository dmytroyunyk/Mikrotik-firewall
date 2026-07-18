package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmytroyunyk/mikrotik-defender/api"
	"github.com/dmytroyunyk/mikrotik-defender/bot"
	"github.com/dmytroyunyk/mikrotik-defender/config"
	"github.com/dmytroyunyk/mikrotik-defender/firewall"
	"github.com/dmytroyunyk/mikrotik-defender/metrics"
	"github.com/dmytroyunyk/mikrotik-defender/mikrotik"
	"github.com/dmytroyunyk/mikrotik-defender/storage"
	"github.com/dmytroyunyk/mikrotik-defender/utils"
)

// @title           Mikrotik Intelligent Defender API
// @version         1.0
// @description     REST API for managing the Mikrotik network security system
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
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

	m := metrics.New(db, logger)
	logger.Info("metrics initalized")

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

	engine := firewall.NewEngine(client, whitelist, cfg.Firewall)
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

	apiServer := api.New(db, client, logger, cfg.API.Key, m)
	go func() {
		if err := apiServer.Start(cfg.API.Port); err != nil {
			logger.Error("API server error", "error", err)
		}
	}()
	defer apiServer.Stop()
	logger.Info("API server started", "port", cfg.API.Port)

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
				if teleBot != nil {
					teleBot.NotifyError("firewall engin", err.Error())
				}
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

				m.RecordBlock()

				if teleBot != nil {
					if err := teleBot.NotifyBlocked(blockedIP, event.Message, time.Duration(cfg.Firewall.BanDuration)*time.Minute); err != nil {
						logger.Error("failed to send block notification", "error", err)
					}
				}
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	stopMetrics := make(chan struct{})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.Update()
			case <-stopMetrics:
				return
			}
		}
	}()

	logger.Info("system is running, press Ctrl+C to stop")
	<-quit

	close(stopMetrics)

	logger.Info("shutting down...")

	if teleBot != nil {
		if err := teleBot.NotifyShutdown(); err != nil {
			logger.Error("failed to send shutdown notification", "error", err)
		}
	}
	logger.Info("system stopped")
}
