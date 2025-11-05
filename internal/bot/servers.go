package bot

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleServers handles the /servers command
func (b *Bot) handleServers(message *tgbotapi.Message) {
	servers, err := b.getUserServersWithInfo(message.From.ID)
	if err != nil {
		b.sendMessage(message.Chat.ID, "❌ Error retrieving servers.")
		return
	}

	if len(servers) == 0 {
		text := "📭 No servers connected.\n\n💡 To connect a server:\n1. Install ServerEye agent\n2. Use /add srv_your_key MyServerName"

		// Add button
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ Add Server", "add_server"),
			),
		)

		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		msg.ReplyMarkup = keyboard
		if _, err := b.telegramAPI.Send(msg); err != nil {
			b.logger.Error("Failed to send message", err)
		}
		return
	}

	// Build server list text
	var response string
	if len(servers) == 1 {
		statusIcon := "🟢"
		if servers[0].Status == "offline" {
			statusIcon = "🔴"
		}
		keyPreview := servers[0].SecretKey
		if len(keyPreview) > 12 {
			keyPreview = keyPreview[:12] + "..."
		}
		response = fmt.Sprintf("📋 Your server:\n%s **%s** (%s)\n\n💡 All commands will use this server automatically.",
			statusIcon, servers[0].Name, keyPreview)
	} else {
		// Multiple servers
		response = "📋 Your servers:\n\n"
		for i, server := range servers {
			statusIcon := "🟢"
			if server.Status == "offline" {
				statusIcon = "🔴"
			}
			keyPreview := server.SecretKey
			if len(keyPreview) > 12 {
				keyPreview = keyPreview[:12] + "..."
			}
			response += fmt.Sprintf("%d. %s **%s** (%s)\n", i+1, statusIcon, server.Name, keyPreview)
		}
		response += "\n💡 Commands will show buttons to select server."
	}

	// Add management buttons
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Status", "server_status"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Rename", "server_rename"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Remove", "server_remove"),
			tgbotapi.NewInlineKeyboardButtonData("➕ Add", "add_server"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	if _, err := b.telegramAPI.Send(msg); err != nil {
		b.logger.Error("Failed to send message", err)
	}
}

// handleRenameServer handles the /rename_server command
func (b *Bot) handleRenameServer(message *tgbotapi.Message) string {
	parts := strings.Fields(message.Text)
	if len(parts) < 3 {
		return "❌ Usage: /rename_server <server#> <new_name>\nExample: /rename_server 1 MyWebServer"
	}

	servers, err := b.getUserServers(message.From.ID)
	if err != nil || len(servers) == 0 {
		return "❌ No servers found."
	}

	serverNum, err := strconv.Atoi(parts[1])
	if err != nil || serverNum < 1 || serverNum > len(servers) {
		return fmt.Sprintf("❌ Invalid server number. You have %d servers.", len(servers))
	}

	newName := strings.Join(parts[2:], " ")
	if len(newName) > 50 {
		return "❌ Server name too long (max 50 characters)."
	}

	serverKey := servers[serverNum-1]
	if err := b.renameServer(serverKey, newName); err != nil {
		return "❌ Failed to rename server."
	}

	return fmt.Sprintf("✅ Server renamed to: %s", newName)
}

// handleRemoveServer handles the /remove_server command
func (b *Bot) handleRemoveServer(message *tgbotapi.Message) string {
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		return "❌ Usage: /remove_server <server#>\nExample: /remove_server 1\n\n⚠️ This will permanently remove the server!"
	}

	servers, err := b.getUserServers(message.From.ID)
	if err != nil || len(servers) == 0 {
		return "❌ No servers found."
	}

	serverNum, err := strconv.Atoi(parts[1])
	if err != nil || serverNum < 1 || serverNum > len(servers) {
		return fmt.Sprintf("❌ Invalid server number. You have %d servers.", len(servers))
	}

	serverKey := servers[serverNum-1]
	if err := b.removeServer(message.From.ID, serverKey); err != nil {
		return "❌ Failed to remove server."
	}

	return "✅ Server removed successfully."
}

// handleAddServer handles the /add command
func (b *Bot) handleAddServer(message *tgbotapi.Message) string {
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		return "❌ Usage: /add <server_key> [server_name]\nExample: /add srv_684eab33... MyWebServer"
	}

	serverKey := strings.TrimSpace(parts[1])
	if !strings.HasPrefix(serverKey, "srv_") {
		return "❌ Invalid server key. Server key must start with 'srv_'"
	}

	// Optional server name
	serverName := "Server"
	if len(parts) >= 3 {
		serverName = strings.Join(parts[2:], " ")
		if len(serverName) > 50 {
			return "❌ Server name too long (max 50 characters)."
		}
	}

	if err := b.connectServerWithName(message.From.ID, serverKey, serverName); err != nil {
		b.logger.Error("Error occurred", err)
		if err.Error() == "invalid server key: key not found in generated keys" {
			return "❌ Invalid server key. Please make sure you're using a key generated by the ServerEye agent installation."
		}
		return "❌ Failed to connect server. Please check your key or server may already be connected."
	}

	return fmt.Sprintf("✅ Server '%s' connected successfully!\n🟢 Status: Online\n\nUse /temp to get CPU temperature.", serverName)
}

// handleDebug shows debug information about user and servers
func (b *Bot) handleDebug(message *tgbotapi.Message) string {
	userID := message.From.ID

	// Check if user exists in database
	var userExists bool
	err := b.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE telegram_id = $1)", userID).Scan(&userExists)
	if err != nil {
		return fmt.Sprintf("❌ Database error: %v", err)
	}

	// Get user servers count
	var serverCount int
	err = b.db.QueryRow(`
		SELECT COUNT(*) FROM user_servers us 
		JOIN servers s ON us.server_id = s.id 
		WHERE us.user_id = $1
	`, userID).Scan(&serverCount)
	if err != nil {
		return fmt.Sprintf("❌ Error getting servers: %v", err)
	}

	// Get total users and servers in database
	var totalUsers, totalServers, totalKeys, connectedKeys int
	if err := b.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers); err != nil {
		b.logger.Error("Failed to get users count", err)
	}
	if err := b.db.QueryRow("SELECT COUNT(*) FROM servers").Scan(&totalServers); err != nil {
		b.logger.Error("Failed to get servers count", err)
	}
	if err := b.db.QueryRow("SELECT COUNT(*) FROM generated_keys").Scan(&totalKeys); err != nil {
		b.logger.Error("Failed to get keys count", err)
	}
	if err := b.db.QueryRow("SELECT COUNT(*) FROM generated_keys WHERE status = 'connected'").Scan(&connectedKeys); err != nil {
		b.logger.Error("Failed to get connected keys count", err)
	}

	return fmt.Sprintf(`🔍 **Debug Information**

👤 **Your Status:**
• User registered: %v
• Connected servers: %d

📊 **Database Stats:**
• Total users: %d
• Total servers: %d
• Generated keys: %d
• Connected keys: %d

💡 **Tip:** If you have 0 servers after bot restart, use:
/start
/add srv_your_key MyServer`,
		userExists, serverCount, totalUsers, totalServers, totalKeys, connectedKeys)
}

// handleStats shows detailed statistics about generated keys (admin command)
func (b *Bot) handleStats(message *tgbotapi.Message) string {
	// Simple admin check - you can make this more sophisticated
	adminUsers := []int64{1805441944} // Add your Telegram ID here
	isAdmin := false
	for _, adminID := range adminUsers {
		if message.From.ID == adminID {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		return "❌ This command is only available for administrators."
	}

	// Get detailed statistics
	var totalKeys, connectedKeys, generatedToday int
	var firstKeyDate, lastKeyDate string

	b.db.QueryRow("SELECT COUNT(*) FROM generated_keys").Scan(&totalKeys)
	b.db.QueryRow("SELECT COUNT(*) FROM generated_keys WHERE status = 'connected'").Scan(&connectedKeys)
	b.db.QueryRow("SELECT COUNT(*) FROM generated_keys WHERE generated_at >= CURRENT_DATE").Scan(&generatedToday)
	b.db.QueryRow("SELECT MIN(generated_at)::date, MAX(generated_at)::date FROM generated_keys").Scan(&firstKeyDate, &lastKeyDate)

	// Get keys by status
	var statusStats string
	rows, err := b.db.Query(`
		SELECT status, COUNT(*) 
		FROM generated_keys 
		GROUP BY status 
		ORDER BY COUNT(*) DESC
	`)
	if err == nil {
		defer rows.Close()
		statusStats = "\n📊 **Keys by Status:**\n"
		for rows.Next() {
			var status string
			var count int
			rows.Scan(&status, &count)
			statusStats += fmt.Sprintf("• %s: %d\n", status, count)
		}
	}

	connectionRate := float64(0)
	if totalKeys > 0 {
		connectionRate = float64(connectedKeys) / float64(totalKeys) * 100
	}

	return fmt.Sprintf(`📈 **ServerEye Statistics**

🔑 **Key Generation:**
• Total keys generated: %d
• Keys connected: %d (%.1f%%)
• Generated today: %d
• First key: %s
• Latest key: %s
%s
📊 **Usage Insights:**
• Connection rate: %.1f%%
• Active installations: %d

🔧 **System Health:**
This data helps track ServerEye adoption and usage patterns.`,
		totalKeys, connectedKeys, connectionRate, generatedToday,
		firstKeyDate, lastKeyDate, statusStats, connectionRate, connectedKeys)
}
