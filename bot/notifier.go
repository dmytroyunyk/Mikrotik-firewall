package bot

import (
	"fmt"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (b *Bot) NotifyBlocked(ip, reason string, duration time.Duration) error {
	msg := fmt.Sprintf("🚨 *IP blocked*\n\n🌐 IP: `%s`\n📝 Reason: %s\n⏱ Duration: %s\n🕐 Time: %s\n",
		ip,
		reason,
		formatDuration(duration),
		time.Now().Format("02.01.2006 15:04:05"),
	)
	return b.sendToChat(msg)
}

func (b *Bot) NotifyError(component, message string) error {
	msg := fmt.Sprintf("⚠️ *System error*\n\n🔧 Component: %s\n❌ Error: %s\n🕐 Time: %s\n",
		component,
		message,
		time.Now().Format("02.01.2006 15:04:05"),
	)
	return b.sendToChat(msg)
}

func (b *Bot) NotifyStartup() error {
	msg := fmt.Sprintf("✅ *System started*\n\n🛡 Mikrotik Intelligent Defender active\n🕐 Startup time: %s\n",
		time.Now().Format("02.01.2006 15:04:05"),
	)
	return b.sendToChat(msg)
}

func (b *Bot) NotifyShutdown() error {
	msg := fmt.Sprintf("🔴 *System stopped*\n\n🛡 Mikrotik Intelligent Defender disabled\n🕐 Time: %s\n",
		time.Now().Format("02.01.2006 15:04:05"),
	)
	return b.sendToChat(msg)
}

func (b *Bot) sendToChat(msg string) error {
	chat := &tele.Chat{ID: int64(b.chatID)}

	_, err := b.bot.Send(chat, msg, tele.ModeMarkdown)
	if err != nil {
		b.logger.Error("failed to send telegram message", "error", err)
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	return nil
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%d hour %d minute", hours, minutes)
	}

	if hours > 0 {
		return fmt.Sprintf("%d hour", hours)
	}

	return fmt.Sprintf("%d minute", minutes)
}
