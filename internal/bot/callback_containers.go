package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleContainerActionCallback handles container action button clicks
func (b *Bot) handleContainerActionCallback(query *tgbotapi.CallbackQuery) error {
	// Parse callback data (format: "container_action_containerID")
	parts := strings.SplitN(query.Data, "_", 3)
	if len(parts) != 3 {
		b.sendMessage(query.Message.Chat.ID, "❌ Invalid callback format")
		return fmt.Errorf("invalid container callback format: %s", query.Data)
	}

	action := parts[1]      // start, stop, restart
	containerID := parts[2] // container ID or name

	// Get user's servers
	servers, err := b.getUserServers(query.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		b.sendMessage(query.Message.Chat.ID, "❌ Error getting your servers")
		return err
	}

	if len(servers) == 0 {
		b.sendMessage(query.Message.Chat.ID, "❌ No servers found")
		return fmt.Errorf("no servers found")
	}

	// Get action-specific messages
	var processingMsg string
	switch action {
	case "start":
		processingMsg = "▶️ Starting `%s`..."
	case "stop":
		processingMsg = "⏹️ Stopping `%s`..."
	case "restart":
		processingMsg = "🔄 Restarting `%s`..."
	case "remove":
		processingMsg = "🗑️ Deleting `%s`..."
	default:
		processingMsg = "⏳ Processing container `%s`...\n\n_Please wait..._"
	}

	// Show processing message
	editMsg := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		fmt.Sprintf(processingMsg, containerID),
	)
	editMsg.ParseMode = "Markdown"
	if _, err := b.telegramAPI.Send(editMsg); err != nil {
		b.logger.Error("Error occurred", err)
	}

	// Execute action
	response := b.handleContainerAction(query.From.ID, containerID, action)

	// Update message with result
	editMsg = tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		response,
	)
	editMsg.ParseMode = "Markdown"
	if _, err := b.telegramAPI.Send(editMsg); err != nil {
		b.logger.Error("Error occurred", err)
	}

	return nil
}

// handleContainerActionSelection shows list of containers to select for action
//
//nolint:gocyclo // Complex but clear logic for container action UI
func (b *Bot) handleContainerActionSelection(query *tgbotapi.CallbackQuery) error {
	// Parse action from callback data
	action := strings.TrimPrefix(query.Data, "container_action_")

	// Get user's servers
	servers, err := b.getUserServers(query.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		b.sendMessage(query.Message.Chat.ID, "❌ Error getting your servers")
		return err
	}

	if len(servers) == 0 {
		b.sendMessage(query.Message.Chat.ID, "❌ No servers found")
		return fmt.Errorf("no servers found")
	}

	// Use first server
	serverKey := servers[0]

	// Get containers list
	containers, err := b.getContainers(serverKey)
	if err != nil {
		b.logger.Error("Error occurred", err)
		b.sendMessage(query.Message.Chat.ID, fmt.Sprintf("❌ Failed to get containers: %v", err))
		return err
	}

	if containers.Total == 0 {
		b.sendMessage(query.Message.Chat.ID, "📦 No containers found")
		return nil
	}

	// Build action-specific message
	var actionText string
	switch action {
	case "start":
		actionText = "▶️ Select container to START:"
	case "stop":
		actionText = "⏹️ Select container to STOP:"
	case "restart":
		actionText = "🔄 Select container to RESTART:"
	case "remove":
		actionText = "🗑️ Select container to DELETE:"
	case "create":
		return b.handleContainerCreateTemplates(query)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	// Filter and build buttons for each container based on action
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, container := range containers.Containers {
		containerID := container.Name
		if containerID == "" {
			containerID = container.ID[:12]
		}

		isRunning := strings.Contains(strings.ToLower(container.State), "running")

		// Filter containers based on action
		if action == "start" && isRunning {
			continue
		}
		if (action == "stop" || action == "restart") && !isRunning {
			continue
		}
		if action == "remove" && isRunning {
			continue
		}

		// Status emoji
		statusEmoji := "🔴"
		if isRunning {
			statusEmoji = "🟢"
		}

		buttonText := fmt.Sprintf("%s %s", statusEmoji, container.Name)
		callbackData := fmt.Sprintf("container_%s_%s", action, containerID)

		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}

	// Check if no containers match the filter
	if len(buttons) == 0 {
		var message string
		switch action {
		case "start":
			message = "✅ All containers are already running"
		case "stop", "restart":
			message = "⏹️ No running containers found"
		case "remove":
			message = "✅ No stopped containers to delete"
		}
		editMsg := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID,
			query.Message.MessageID,
			message,
		)
		if _, err := b.telegramAPI.Send(editMsg); err != nil {
			b.logger.Error("Error occurred", err)
		}
		return nil
	}

	// Add cancel button
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "container_cancel"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	editMsg := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		actionText,
	)
	editMsg.ReplyMarkup = &keyboard

	if _, err := b.telegramAPI.Send(editMsg); err != nil {
		b.logger.Error("Error occurred", err)
	}

	return nil
}

// handleContainerCreateTemplates shows template selection for creating containers
func (b *Bot) handleContainerCreateTemplates(query *tgbotapi.CallbackQuery) error {
	text := "📦 Select container template:\n\nChoose a pre-configured template to quickly deploy a container:"

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🌐 Nginx", "create_template_nginx"),
			tgbotapi.NewInlineKeyboardButtonData("🐘 PostgreSQL", "create_template_postgres"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔴 Redis", "create_template_redis"),
			tgbotapi.NewInlineKeyboardButtonData("🟢 MongoDB", "create_template_mongo"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🐰 RabbitMQ", "create_template_rabbitmq"),
			tgbotapi.NewInlineKeyboardButtonData("🐳 MySQL", "create_template_mysql"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "container_cancel"),
		},
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	editMsg := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		text,
	)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &keyboard

	if _, err := b.telegramAPI.Send(editMsg); err != nil {
		b.logger.Error("Error occurred", err)
	}

	return nil
}

// handleTemplateSelection handles template selection and creates container
func (b *Bot) handleTemplateSelection(query *tgbotapi.CallbackQuery) error {
	// Parse template name
	template := strings.TrimPrefix(query.Data, "create_template_")

	// Get user's servers
	servers, err := b.getUserServers(query.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		b.sendMessage(query.Message.Chat.ID, "❌ Error getting your servers")
		return err
	}

	if len(servers) == 0 {
		b.sendMessage(query.Message.Chat.ID, "❌ No servers found")
		return fmt.Errorf("no servers found")
	}

	serverKey := servers[0]

	// Show processing message
	var templateName string
	switch template {
	case "nginx":
		templateName = "Nginx"
	case "postgres":
		templateName = "PostgreSQL"
	case "redis":
		templateName = "Redis"
	case "mongo":
		templateName = "MongoDB"
	case "rabbitmq":
		templateName = "RabbitMQ"
	case "mysql":
		templateName = "MySQL"
	default:
		templateName = template
	}

	editMsg := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		fmt.Sprintf("📦 Creating %s...", templateName),
	)
	editMsg.ParseMode = "Markdown"
	if _, err := b.telegramAPI.Send(editMsg); err != nil {
		b.logger.Error("Error occurred", err)
	}

	// Create container based on template
	response := b.createContainerFromTemplate(query.From.ID, serverKey, template)

	// Update message with result
	editMsg = tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		response,
	)
	editMsg.ParseMode = "Markdown"
	if _, err := b.telegramAPI.Send(editMsg); err != nil {
		b.logger.Error("Error occurred", err)
	}

	return nil
}
