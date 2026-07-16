package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/dmytroyunyk/mikrotik-defender/bot"
	"github.com/dmytroyunyk/mikrotik-defender/config"
	"github.com/dmytroyunyk/mikrotik-defender/mikrotik"
	"github.com/dmytroyunyk/mikrotik-defender/storage"
	"github.com/dmytroyunyk/mikrotik-defender/utils"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		logger := utils.NewLogger("info")
		logger.Fatal("failed to load config", "error", err)
	}

	logger := utils.NewLogger(cfg.Log.Level)
	logger.Info("starting Telegram bot")

	client := mikrotik.NewClient(
		cfg.MikroTik.Address,
		cfg.MikroTik.Username,
		cfg.MikroTik.Password,
	)
	if err := client.Connect(); err != nil {
		logger.Fatal("failed to connect to MikroTik", "error", err)
	}
	defer client.Disconnect()

	db, err := storage.New(cfg.Storage.Path)
	if err != nil {
		logger.Fatal("failed to open database", "error", err)
	}
	defer db.Close()

	teleBot, err := bot.New(cfg, db, client, logger)
	if err != nil {
		logger.Fatal("failed to create telegram bot", "error", err)
	}
	teleBot.Start()
	defer teleBot.Stop()

	logger.Info("bot is running, press Ctrl+C to stop")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("bot stopped")
}
