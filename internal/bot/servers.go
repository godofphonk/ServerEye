package bot

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleServers handles the /servers command
func (b *Bot) handleServers(message *tgbotapi.Message) string {
	servers, err := b.getUserServersWithInfo(message.From.ID)
	if err != nil {
		return "❌ Error retrieving servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected.\n\n💡 To connect a server:\n1. Install ServerEye agent\n2. Use /add srv_your_key MyServerName"
	}

	if len(servers) == 1 {
		statusIcon := "🟢"
		if servers[0].Status == "offline" {
			statusIcon = "🔴"
		}
		return fmt.Sprintf("📋 Your servers:\n%s **%s** (%s)\n\n💡 All commands will use this server automatically.\n\n🔧 Management:\n/rename_server 1 <name> - Rename server\n/remove_server 1 - Remove server", 
			statusIcon, servers[0].Name, servers[0].SecretKey[:12]+"...")
	}

	// Multiple servers - show list with numbers
	response := "📋 Your servers:\n\n"
	for i, server := range servers {
		statusIcon := "🟢"
		if server.Status == "offline" {
			statusIcon = "🔴"
		}
		response += fmt.Sprintf("%d. %s **%s** (%s)\n", i+1, statusIcon, server.Name, server.SecretKey[:12]+"...")
	}
	response += "\n💡 Commands will show buttons to select server:\n"
	response += "Just use /temp or /containers - no numbers needed!\n\n"
	response += "🔧 Management:\n"
	response += "/rename_server <#> <name> - Rename server\n"
	response += "/remove_server <#> - Remove server"
	
	return response
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
		b.logger.WithError(err).Error("Failed to connect server")
		return "❌ Failed to connect server. Please check your key or server may already be connected."
	}

	return fmt.Sprintf("✅ Server '%s' connected successfully!\n🟢 Status: Online\n\nUse /temp to get CPU temperature.", serverName)
}

// handleServerKey handles server key registration (deprecated)
func (b *Bot) handleServerKey(message *tgbotapi.Message) string {
	serverKey := strings.TrimSpace(message.Text)

	if err := b.connectServer(message.From.ID, serverKey); err != nil {
		b.logger.WithError(err).Error("Failed to connect server")
		return "❌ Failed to connect server. Please check your key."
	}

	return "✅ Server connected successfully!\n🟢 Status: Online\n\nUse /temp to get CPU temperature."
}
