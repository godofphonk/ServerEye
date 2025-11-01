package bot

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
