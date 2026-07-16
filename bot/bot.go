package bot

import (
	"fmt"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/dmytroyunyk/mikrotik-defender/config"
	"github.com/dmytroyunyk/mikrotik-defender/mikrotik"
	"github.com/dmytroyunyk/mikrotik-defender/storage"
	"github.com/dmytroyunyk/mikrotik-defender/utils"
)

type Bot struct {
	bot    *tele.Bot
	db     *storage.DB
	client *mikrotik.Client
	logger *utils.Logger
	chatID tele.ChatID
}

func New(
	cfg *config.Config,
	db *storage.DB,
	client *mikrotik.Client,
	logger *utils.Logger,
) (*Bot, error) {
	settings := tele.Settings{
		Token:  cfg.Telegram.Token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	teleBot, err := tele.NewBot(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	var chatID tele.ChatID
	_, err = fmt.Sscanf(cfg.Telegram.ChatID, "%d", &chatID)
	if err != nil {
		return nil, fmt.Errorf("invalid telegram chat_id: %w", err)
	}

	b := &Bot{
		bot:    teleBot,
		db:     db,
		client: client,
		logger: logger,
		chatID: chatID,
	}

	b.registerHandlers()

	return b, nil
}

func (b *Bot) registerHandlers() {
	b.bot.Handle("/start", b.handleStart)
	b.bot.Handle("/status", b.handleStatus)
	b.bot.Handle("/blocked", b.handleBlocked)
	b.bot.Handle("/unban", b.handleUnban)
	b.bot.Handle("/top", b.handleTop)
}

func (b *Bot) Start() {
	b.logger.Info("Telegram bot started")
	go b.bot.Start()
}

func (b *Bot) Stop() {
	b.bot.Stop()
	b.logger.Info("Telegram bot stopped")
}
