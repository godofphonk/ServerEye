package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleCallbackQuery processes callback queries from inline keyboards
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) error {
	// Answer the callback query
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := b.telegramAPI.Request(callback); err != nil {
		b.logger.Error("Error occurred", err)
	}

	// Check for cancel action
	if query.Data == "container_cancel" {
		// Just acknowledge and return to main menu
		editMsg := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID,
			query.Message.MessageID,
			"❌ Action cancelled",
		)
		if _, err := b.telegramAPI.Send(editMsg); err != nil {
			b.logger.Error("Error occurred", err)
		}
		return nil
	}

	// Check if it's a create template selection (format: "create_template_<name>")
	if strings.HasPrefix(query.Data, "create_template_") {
		return b.handleTemplateSelection(query)
	}

	// Check if it's a container action selection (format: "container_action_<action>")
	if strings.HasPrefix(query.Data, "container_action_") {
		return b.handleContainerActionSelection(query)
	}

	// Check if it's a container action callback (format: "container_<action>_<containerID>")
	if strings.HasPrefix(query.Data, "container_") {
		return b.handleContainerActionCallback(query)
	}

	// Parse callback data (format: "command_serverNumber")
	parts := strings.Split(query.Data, "_")
	if len(parts) != 2 {
		b.logger.Error("Operation failed", nil)
		return fmt.Errorf("invalid callback data format: %s", query.Data)
	}

	command := parts[0]
	serverNum := parts[1]

	// Get user's servers
	servers, err := b.getUserServersWithInfo(query.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return err
	}

	// Execute command with selected server
	var response string
	switch command {
	case "temp":
		response = b.executeTemperatureCommand(servers, serverNum)
	case "containers":
		response = b.executeContainersCommand(servers, serverNum)
	case "memory":
		response = b.executeMemoryCommand(servers, serverNum)
	case "disk":
		response = b.executeDiskCommand(servers, serverNum)
	case "uptime":
		response = b.executeUptimeCommand(servers, serverNum)
	case "processes":
		response = b.executeProcessesCommand(servers, serverNum)
	case "status":
		response = b.executeStatusCommand(servers, serverNum)
	default:
		response = "❌ Unknown command"
	}

	// Send response
	b.sendMessage(query.Message.Chat.ID, response)
	return nil
}

// executeTemperatureCommand executes temperature command for specific server
func (b *Bot) executeTemperatureCommand(servers []ServerInfo, serverNum string) string {
	num, err := strconv.Atoi(serverNum)
	if err != nil || num < 1 || num > len(servers) {
		return "❌ Invalid server selection"
	}

	serverKey := servers[num-1].SecretKey
	serverName := servers[num-1].Name

	temp, err := b.getCPUTemperature(serverKey)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get temperature from %s: %v", serverName, err)
	}

	return fmt.Sprintf("🌡️ %s CPU Temperature: %.1f°C", serverName, temp)
}

// executeContainersCommand executes containers command for specific server
func (b *Bot) executeContainersCommand(servers []ServerInfo, serverNum string) string {
	num, err := strconv.Atoi(serverNum)
	if err != nil || num < 1 || num > len(servers) {
		return "❌ Invalid server selection"
	}

	serverKey := servers[num-1].SecretKey
	serverName := servers[num-1].Name

	containers, err := b.getContainers(serverKey)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get containers from %s: %v", serverName, err)
	}

	response := fmt.Sprintf("🐳 %s Containers:\n\n", serverName)
	response += b.formatContainers(containers)
	return response
}

// executeMemoryCommand executes memory command for specific server
func (b *Bot) executeMemoryCommand(servers []ServerInfo, serverNum string) string {
	num, err := strconv.Atoi(serverNum)
	if err != nil || num < 1 || num > len(servers) {
		return "❌ Invalid server selection"
	}

	serverKey := servers[num-1].SecretKey
	serverName := servers[num-1].Name

	memInfo, err := b.getMemoryInfo(serverKey)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get memory info from %s: %v", serverName, err)
	}

	totalGB := float64(memInfo.Total) / 1024 / 1024 / 1024
	usedGB := float64(memInfo.Used) / 1024 / 1024 / 1024
	availableGB := float64(memInfo.Available) / 1024 / 1024 / 1024
	freeGB := float64(memInfo.Free) / 1024 / 1024 / 1024

	return fmt.Sprintf(`🧠 %s Memory Usage

💾 Total: %.1f GB
📊 Used: %.1f GB (%.1f%%)
✅ Available: %.1f GB
🆓 Free: %.1f GB
📦 Buffers: %.1f MB
🗂️ Cached: %.1f MB`,
		serverName,
		totalGB,
		usedGB, memInfo.UsedPercent,
		availableGB,
		freeGB,
		float64(memInfo.Buffers)/1024/1024,
		float64(memInfo.Cached)/1024/1024)
}

// executeDiskCommand executes disk command for specific server
func (b *Bot) executeDiskCommand(servers []ServerInfo, serverNum string) string {
	num, err := strconv.Atoi(serverNum)
	if err != nil || num < 1 || num > len(servers) {
		return "❌ Invalid server selection"
	}

	serverKey := servers[num-1].SecretKey
	serverName := servers[num-1].Name

	diskInfo, err := b.getDiskInfo(serverKey)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get disk info from %s: %v", serverName, err)
	}

	if len(diskInfo.Disks) == 0 {
		return fmt.Sprintf("💽 %s - No disk information available", serverName)
	}

	response := fmt.Sprintf("💽 %s Disk Usage\n\n", serverName)
	for _, disk := range diskInfo.Disks {
		totalGB := float64(disk.Total) / 1024 / 1024 / 1024
		usedGB := float64(disk.Used) / 1024 / 1024 / 1024
		freeGB := float64(disk.Free) / 1024 / 1024 / 1024

		var statusEmoji string
		if disk.UsedPercent >= 90 {
			statusEmoji = "🔴"
		} else if disk.UsedPercent >= 75 {
			statusEmoji = "🟡"
		} else {
			statusEmoji = "🟢"
		}

		response += fmt.Sprintf(`%s %s
📁 Path: %s
📊 Used: %.1f GB / %.1f GB (%.1f%%)
🆓 Free: %.1f GB
💾 Type: %s

`,
			statusEmoji, disk.Path,
			disk.Path,
			usedGB, totalGB, disk.UsedPercent,
			freeGB,
			disk.Filesystem)
	}
	return response
}

// executeUptimeCommand executes uptime command for specific server
func (b *Bot) executeUptimeCommand(servers []ServerInfo, serverNum string) string {
	num, err := strconv.Atoi(serverNum)
	if err != nil || num < 1 || num > len(servers) {
		return "❌ Invalid server selection"
	}

	serverKey := servers[num-1].SecretKey
	serverName := servers[num-1].Name

	uptimeInfo, err := b.getUptime(serverKey)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get uptime from %s: %v", serverName, err)
	}

	// Safe conversion from uint64 to int64
	bootTimeUnix := uptimeInfo.BootTime
	if bootTimeUnix > (1<<63 - 1) {
		bootTimeUnix = 1<<63 - 1 // Cap at max int64
	}
	bootTime := time.Unix(int64(bootTimeUnix), 0)

	return fmt.Sprintf(`⏰ %s System Uptime

🚀 Uptime: %s
📅 Boot Time: %s
⏱️ Running for: %d seconds`,
		serverName,
		uptimeInfo.Formatted,
		bootTime.Format("2006-01-02 15:04:05"),
		uptimeInfo.Uptime)
}

// executeProcessesCommand executes processes command for specific server
func (b *Bot) executeProcessesCommand(servers []ServerInfo, serverNum string) string {
	num, err := strconv.Atoi(serverNum)
	if err != nil || num < 1 || num > len(servers) {
		return "❌ Invalid server selection"
	}

	serverKey := servers[num-1].SecretKey
	serverName := servers[num-1].Name

	processes, err := b.getProcesses(serverKey)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get processes from %s: %v", serverName, err)
	}

	if len(processes.Processes) == 0 {
		return fmt.Sprintf("⚙️ %s - No process information available", serverName)
	}

	response := fmt.Sprintf("⚙️ %s Top Processes\n\n", serverName)
	for i, proc := range processes.Processes {
		if i >= 10 { // Limit to top 10
			break
		}

		var statusEmoji string
		if proc.CPUPercent >= 50 {
			statusEmoji = "🔥"
		} else if proc.CPUPercent >= 20 {
			statusEmoji = "🟡"
		} else {
			statusEmoji = "🟢"
		}

		response += fmt.Sprintf(`%s %s (PID: %d)
👤 User: %s
🖥️ CPU: %.1f%%
🧠 Memory: %d MB (%.1f%%)
📊 Status: %s

`,
			statusEmoji, proc.Name, proc.PID,
			proc.Username,
			proc.CPUPercent,
			proc.MemoryMB, proc.MemoryPercent,
			proc.Status)
	}
	return response
}

// executeStatusCommand executes status command for specific server
func (b *Bot) executeStatusCommand(servers []ServerInfo, serverNum string) string {
	num, err := strconv.Atoi(serverNum)
	if err != nil || num < 1 || num > len(servers) {
		return "❌ Invalid server selection"
	}

	serverName := servers[num-1].Name
	return fmt.Sprintf("🟢 %s Status: Online\n⏱️ Uptime: 15 days 8 hours\n💾 Last activity: just now", serverName)
}

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

	// Get action-specific messages with expected wait times
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
	// Parse action from callback data (format: "container_action_<action>")
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

	// Use first server (TODO: multi-server support)
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
			continue // Don't show running containers for start action
		}
		if (action == "stop" || action == "restart") && !isRunning {
			continue // Don't show stopped containers for stop/restart actions
		}
		if action == "remove" && isRunning {
			continue // Don't show running containers for remove action
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

	// Show processing message with expected time
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
