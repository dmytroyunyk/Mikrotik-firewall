package bot

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (b *Bot) handleStart(c tele.Context) error {
	msg := `
	🛡 *Mikrotik Intelligent Defender*

I protect your network from attacks in real time.

*Available commands:*
/status — general system statistics
/blocked — list of blocked IPs
/top — top attackers
/unban <IP> — unblock an IP address
`
	return c.Send(msg, tele.ModeMarkdown)
}

func (b *Bot) handleStatus(c tele.Context) error {
	stats, err := b.db.GetStats()
	if err != nil {
		b.logger.Error("failed to get stats", "error", err)
		return c.Send("❌ Error retrieving statistics")
	}

	msg := fmt.Sprintf(`
🛡 *System statistics*

📊 Total events: *%d*
🚫 Blocked IPs: *%d*
⚡ Events in the last 24 hours: *%d*
`,
		stats["total_events"],
		stats["blocked_ips"],
		stats["events_24h"],
	)

	return c.Send(msg, tele.ModeMarkdown)
}

func (b *Bot) handleBlocked(c tele.Context) error {
	blocked, err := b.db.GetBlockedIPs()
	if err != nil {
		b.logger.Error("failed to get blocked IPs", "error", err)
		return c.Send("❌ Error retrieving table")
	}

	if len(blocked) == 0 {
		return c.Send("✅ No blocked IPs")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🚫 *blocked IP:: %d*\n\n", len(blocked)))

	limit := len(blocked)
	if limit > 20 {
		limit = 20
	}

	for i, entry := range blocked[:limit] {
		sb.WriteString(fmt.Sprintf(
			"%d. `%s`\n   📝 %s\n   🕐 %s\n\n",
			i+1,
			entry.IP,
			entry.Reason,
			entry.BlockedAt.Format("02.01.2006 15:04"),
		))
	}

	if len(blocked) > 20 {
		sb.WriteString(fmt.Sprintf("_...і yet %d IP_", len(blocked)-20))
	}

	return c.Send(sb.String(), tele.ModeMarkdown)
}

func (b *Bot) handleUnban(c tele.Context) error {
	parts := strings.Fields(c.Message().Text)

	if len(parts) < 2 {
		return c.Send("❌ Вкажи IP адресу\nПриклад: `/unban 1.2.3.4`", tele.ModeMarkdown)
	}

	ip := parts[1]

	err := b.client.Unblock(ip)
	if err != nil {
		b.logger.Error("failed to unblock IP via mikrotik", "ip", ip, "error", err)
		return c.Send(fmt.Sprintf("❌ Не вдалося розблокувати `%s` на роутері", ip), tele.ModeMarkdown)
	}

	err = b.db.MarkAsUnblocked(ip)
	if err != nil {
		b.logger.Error("failed to mark IP as unblocked in db", "ip", ip, "error", err)
	}

	b.logger.Info("IP unblocked via bot", "ip", ip)
	return c.Send(fmt.Sprintf("✅ IP `%s` розблоковано", ip), tele.ModeMarkdown)
}

func (b *Bot) handleTop(c tele.Context) error {
	attackers, err := b.db.GetTopAttackers(10)
	if err != nil {
		b.logger.Error("failed to get top attackers", "error", err)
		return c.Send("❌ error get static")
	}

	if len(attackers) == 0 {
		return c.Send("✅ No attacks have been recorded.")
	}

	var sb strings.Builder
	sb.WriteString("🏴‍☠️ *Top Attackers*\n\n")

	for i, a := range attackers {
		status := "🟢 active"
		if a.IsBlocked {
			status = "🔴 blocked"
		}

		sb.WriteString(fmt.Sprintf(
			"%d. `%s`\n   💥 Attcak: *%d* | %s\n   🕐 last time: %s\n\n",
			i+1,
			a.IP,
			a.AttackCount,
			status,
			a.LastSeen.Format("02.01.2006 15:04"),
		))
	}

	return c.Send(sb.String(), tele.ModeMarkdown)
}
