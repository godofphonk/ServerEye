package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleAddServerCallback shows instructions for adding a server
func (b *Bot) handleAddServerCallback(query *tgbotapi.CallbackQuery) error {
	text := `➕ **Add New Server**

To connect a new server:

1️⃣ **Install ServerEye agent** on your server:
` + "```bash" + `
wget -qO- https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent.sh | sudo bash
` + "```" + `

2️⃣ **Copy the server key** from installation output

3️⃣ **Use the command below**:
/add srv_YOUR_KEY MyServerName

💡 **Example:**
/add srv_684eab33c7... WebServer`

	editMsg := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		text,
	)
	editMsg.ParseMode = "Markdown"

	// Add back button
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "back_to_servers"),
		),
	)
	editMsg.ReplyMarkup = &keyboard

	if _, err := b.telegramAPI.Send(editMsg); err != nil {
		b.logger.Error("Failed to send message", err)
		return err
	}

	return nil
}

// handleServerStatusCallback shows detailed status of all servers
func (b *Bot) handleServerStatusCallback(query *tgbotapi.CallbackQuery) error {
	servers, err := b.getUserServersWithInfo(query.From.ID)
	if err != nil || len(servers) == 0 {
		text := "❌ No servers found."
		editMsg := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID,
			query.Message.MessageID,
			text,
		)
		if _, sendErr := b.telegramAPI.Send(editMsg); sendErr != nil {
			b.logger.Error("Failed to send message", sendErr)
		}
		return err
	}

	// Build detailed status message
	text := "📊 **Server Status**\n\n"
	for i, server := range servers {
		statusIcon := "🟢 Online"
		if server.Status == "offline" {
			statusIcon = "🔴 Offline"
		}

		keyPreview := server.SecretKey
		if len(keyPreview) > 12 {
			keyPreview = keyPreview[:12] + "..."
		}

		text += fmt.Sprintf("%d. **%s**\n", i+1, server.Name)
		text += fmt.Sprintf("   Status: %s\n", statusIcon)
		text += fmt.Sprintf("   Key: `%s`\n", keyPreview)
		text += "\n"
	}

	editMsg := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		text,
	)
	editMsg.ParseMode = "Markdown"

	// Add back button
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "back_to_servers"),
		),
	)
	editMsg.ReplyMarkup = &keyboard

	if _, err := b.telegramAPI.Send(editMsg); err != nil {
		b.logger.Error("Failed to send message", err)
		return err
	}

	return nil
}

// handleServerRenameCallback shows rename instructions with server selection
func (b *Bot) handleServerRenameCallback(query *tgbotapi.CallbackQuery) error {
	servers, err := b.getUserServersWithInfo(query.From.ID)
	if err != nil || len(servers) == 0 {
		text := "❌ No servers found."
		editMsg := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID,
			query.Message.MessageID,
			text,
		)
		if _, sendErr := b.telegramAPI.Send(editMsg); sendErr != nil {
			b.logger.Error("Failed to send message", sendErr)
		}
		return err
	}

	// Build message with server list
	text := "✏️ **Rename Server**\n\nYour servers:\n\n"
	for i, server := range servers {
		statusIcon := "🟢"
		if server.Status == "offline" {
			statusIcon = "🔴"
		}
		text += fmt.Sprintf("%d. %s **%s**\n", i+1, statusIcon, server.Name)
	}

	text += "\n💡 **Usage:**\n/rename_server <#> <new_name>\n\n"
	text += "**Example:**\n/rename_server 1 MyWebServer"

	editMsg := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		text,
	)
	editMsg.ParseMode = "Markdown"

	// Add back button
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "back_to_servers"),
		),
	)
	editMsg.ReplyMarkup = &keyboard

	if _, err := b.telegramAPI.Send(editMsg); err != nil {
		b.logger.Error("Failed to send message", err)
		return err
	}

	return nil
}

// handleServerRemoveCallback shows remove instructions with server selection
func (b *Bot) handleServerRemoveCallback(query *tgbotapi.CallbackQuery) error {
	servers, err := b.getUserServersWithInfo(query.From.ID)
	if err != nil || len(servers) == 0 {
		text := "❌ No servers found."
		editMsg := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID,
			query.Message.MessageID,
			text,
		)
		if _, sendErr := b.telegramAPI.Send(editMsg); sendErr != nil {
			b.logger.Error("Failed to send message", sendErr)
		}
		return err
	}

	// Build message with server list
	text := "🗑 **Remove Server**\n\n⚠️ **Warning:** This will permanently remove the server!\n\nYour servers:\n\n"
	for i, server := range servers {
		statusIcon := "🟢"
		if server.Status == "offline" {
			statusIcon = "🔴"
		}
		text += fmt.Sprintf("%d. %s **%s**\n", i+1, statusIcon, server.Name)
	}

	text += "\n💡 **Usage:**\n/remove_server <#>\n\n"
	text += "**Example:**\n/remove_server 1"

	editMsg := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		text,
	)
	editMsg.ParseMode = "Markdown"

	// Add back button
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "back_to_servers"),
		),
	)
	editMsg.ReplyMarkup = &keyboard

	if _, err := b.telegramAPI.Send(editMsg); err != nil {
		b.logger.Error("Failed to send message", err)
		return err
	}

	return nil
}

// handleBackToServers returns to the main servers menu
func (b *Bot) handleBackToServers(query *tgbotapi.CallbackQuery) error {
	servers, err := b.getUserServersWithInfo(query.From.ID)
	if err != nil {
		text := "❌ Error retrieving servers."
		editMsg := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID,
			query.Message.MessageID,
			text,
		)
		if _, sendErr := b.telegramAPI.Send(editMsg); sendErr != nil {
			b.logger.Error("Failed to send message", sendErr)
		}
		return err
	}

	if len(servers) == 0 {
		text := "📭 No servers connected.\n\n💡 To connect a server:\n1. Install ServerEye agent\n2. Use /add srv_your_key MyServerName"

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ Add Server", "add_server"),
			),
		)

		editMsg := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID,
			query.Message.MessageID,
			text,
		)
		editMsg.ReplyMarkup = &keyboard
		if _, err := b.telegramAPI.Send(editMsg); err != nil {
			b.logger.Error("Failed to send message", err)
		}
		return nil
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

	editMsg := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		response,
	)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &keyboard

	if _, err := b.telegramAPI.Send(editMsg); err != nil {
		b.logger.Error("Failed to send message", err)
		return err
	}

	return nil
}
