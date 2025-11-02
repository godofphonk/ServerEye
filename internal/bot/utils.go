package bot

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/servereye/servereye/pkg/protocol"
)

// sendMessage sends a message to a chat
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.telegramAPI.Send(msg); err != nil {
		b.logger.Error("Error occurred", err)
	}
}

// getServerFromCommand parses server number from command and returns server key
func (b *Bot) getServerFromCommand(command string, servers []string) (string, error) {
	// Check if servers list is empty
	if len(servers) == 0 {
		return "", fmt.Errorf("❌ No servers found. Please add a server first using /add command.")
	}

	parts := strings.Fields(command)

	// If no server number specified, use first server
	if len(parts) == 1 {
		if len(servers) > 1 {
			return "", fmt.Errorf("❌ Multiple servers found. Please use the command again to see server selection buttons.\n\nUse /servers to see your servers.")
		}
		return servers[0], nil
	}

	// Parse server number
	if len(parts) >= 2 {
		serverNum, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", fmt.Errorf("❌ Invalid server number. Use /servers to see available servers.")
		}

		if serverNum < 1 || serverNum > len(servers) {
			return "", fmt.Errorf("❌ Server number %d not found. You have %d servers.\nUse /servers to see available servers.", serverNum, len(servers))
		}

		return servers[serverNum-1], nil
	}

	return servers[0], nil
}

// sendServerSelectionButtons sends inline keyboard with server selection
func (b *Bot) sendServerSelectionButtons(chatID int64, command, text string, servers []ServerInfo) {
	var buttons [][]tgbotapi.InlineKeyboardButton

	for i, server := range servers {
		statusIcon := "🟢"
		if server.Status == "offline" {
			statusIcon = "🔴"
		}

		buttonText := fmt.Sprintf("%s %s", statusIcon, server.Name)
		callbackData := fmt.Sprintf("%s_%d", command, i+1)

		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	if _, err := b.telegramAPI.Send(msg); err != nil {
		b.logger.Error("Error occurred", err)
	}
}

// sendContainersWithButtons sends containers list with action buttons
func (b *Bot) sendContainersWithButtons(chatID int64, serverKey string, containers *protocol.ContainersPayload) {
	if containers.Total == 0 {
		b.sendMessage(chatID, "📦 No Docker containers found on the server.")
		return
	}

	text := fmt.Sprintf("🐳 **Docker Containers (%d total):**\n\n", containers.Total)

	for i, container := range containers.Containers {
		if i >= 10 { // Limit to 10 containers
			text += fmt.Sprintf("... and %d more containers\n", containers.Total-10)
			break
		}

		// Status emoji
		statusEmoji := "🔴" // Red for stopped
		if strings.Contains(strings.ToLower(container.State), "running") {
			statusEmoji = "🟢" // Green for running
		} else if strings.Contains(strings.ToLower(container.State), "paused") {
			statusEmoji = "🟡" // Yellow for paused
		}

		text += fmt.Sprintf("%s **%s**\n", statusEmoji, container.Name)
		text += fmt.Sprintf("📷 Image: `%s`\n", container.Image)
		text += fmt.Sprintf("🔄 Status: %s\n", container.Status)

		if len(container.Ports) > 0 {
			text += fmt.Sprintf("🔌 Ports: %s\n", strings.Join(container.Ports, ", "))
		}

		// Add action buttons for each container
		var buttons []tgbotapi.InlineKeyboardButton

		containerID := container.ID[:12] // Short ID
		if container.Name != "" {
			containerID = container.Name
		}

		// Show appropriate buttons based on container state
		if strings.Contains(strings.ToLower(container.State), "running") {
			// Running: show Stop and Restart
			buttons = append(buttons,
				tgbotapi.NewInlineKeyboardButtonData("⏹️ Stop", fmt.Sprintf("container_stop_%s", containerID)),
				tgbotapi.NewInlineKeyboardButtonData("🔄 Restart", fmt.Sprintf("container_restart_%s", containerID)),
			)
		} else {
			// Stopped: show Start and Delete
			buttons = append(buttons,
				tgbotapi.NewInlineKeyboardButtonData("▶️ Start", fmt.Sprintf("container_start_%s", containerID)),
				tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete", fmt.Sprintf("container_remove_%s", containerID)),
			)
		}

		text += "\n"

		// Send message for this container with buttons
		keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons)
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := b.telegramAPI.Send(msg); err != nil {
			b.logger.Error("Error occurred", err)
		}

		text = "" // Reset for next container
	}
}
