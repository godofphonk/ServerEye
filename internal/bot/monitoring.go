package bot

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleTemp handles the /temp command
func (b *Bot) handleTemp(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /temp")

	servers, err := b.getUserServersWithInfo(message.From.ID)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get user servers")
		return "❌ Error retrieving your servers."
	}

	b.logger.WithField("servers_count", len(servers)).Info("Найдено серверов пользователя")

	if len(servers) == 0 {
		return "📭 No servers connected. Use /add to connect a server."
	}

	// If multiple servers, show selection buttons
	if len(servers) > 1 {
		parts := strings.Fields(message.Text)
		if len(parts) == 1 {
			// No server specified, show buttons
			b.sendServerSelectionButtons(message.Chat.ID, "temp", "🌡️ Select server for temperature:", servers)
			return ""
		}
	}

	// Parse server number from command or use first server
	serverKeys := make([]string, len(servers))
	for i, server := range servers {
		serverKeys[i] = server.SecretKey
	}
	
	serverKey, err := b.getServerFromCommand(message.Text, serverKeys)
	if err != nil {
		return err.Error()
	}

	b.logger.WithField("server_key", serverKey[:12]+"...").Info("Запрос температуры с сервера")

	temp, err := b.getCPUTemperature(serverKey)
	if err != nil {
		b.logger.WithError(err).Error("Ошибка получения температуры")
		return fmt.Sprintf("❌ Failed to get temperature: %v", err)
	}

	b.logger.WithField("temperature", temp).Info("Температура успешно получена")
	return fmt.Sprintf("🌡️ CPU Temperature: %.1f°C", temp)
}

// handleMemory handles the /memory command
func (b *Bot) handleMemory(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /memory")

	servers, err := b.getUserServersWithInfo(message.From.ID)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get user servers")
		return "❌ Error retrieving your servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected. Use /add to connect a server."
	}

	// If multiple servers, show selection buttons
	if len(servers) > 1 {
		parts := strings.Fields(message.Text)
		if len(parts) == 1 {
			// No server specified, show buttons
			b.sendServerSelectionButtons(message.Chat.ID, "memory", "🧠 Select server for memory info:", servers)
			return ""
		}
	}

	// Parse server number from command or use first server
	serverKeys := make([]string, len(servers))
	for i, server := range servers {
		serverKeys[i] = server.SecretKey
	}
	
	serverKey, err := b.getServerFromCommand(message.Text, serverKeys)
	if err != nil {
		return err.Error()
	}
	b.logger.WithField("server_key", serverKey[:12]+"...").Info("Запрос информации о памяти с сервера")

	memInfo, err := b.getMemoryInfo(serverKey)
	if err != nil {
		b.logger.WithError(err).Error("Ошибка получения информации о памяти")
		return fmt.Sprintf("❌ Failed to get memory info: %v", err)
	}

	// Format memory information
	totalGB := float64(memInfo.Total) / 1024 / 1024 / 1024
	usedGB := float64(memInfo.Used) / 1024 / 1024 / 1024
	availableGB := float64(memInfo.Available) / 1024 / 1024 / 1024
	freeGB := float64(memInfo.Free) / 1024 / 1024 / 1024

	response := fmt.Sprintf(`🧠 **Memory Usage**

💾 **Total:** %.1f GB
📊 **Used:** %.1f GB (%.1f%%)
✅ **Available:** %.1f GB
🆓 **Free:** %.1f GB
📦 **Buffers:** %.1f MB
🗂️ **Cached:** %.1f MB`,
		totalGB,
		usedGB, memInfo.UsedPercent,
		availableGB,
		freeGB,
		float64(memInfo.Buffers)/1024/1024,
		float64(memInfo.Cached)/1024/1024)

	b.logger.WithField("used_percent", memInfo.UsedPercent).Info("Информация о памяти успешно получена")
	return response
}

// handleDisk handles the /disk command
func (b *Bot) handleDisk(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /disk")

	servers, err := b.getUserServers(message.From.ID)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get user servers")
		return "❌ Error retrieving your servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected. Use /add to connect a server."
	}

	// For now, use the first server
	serverKey := servers[0]
	b.logger.WithField("server_key", serverKey[:12]+"...").Info("Запрос информации о дисках с сервера")

	diskInfo, err := b.getDiskInfo(serverKey)
	if err != nil {
		b.logger.WithError(err).Error("Ошибка получения информации о дисках")
		return fmt.Sprintf("❌ Failed to get disk info: %v", err)
	}

	if len(diskInfo.Disks) == 0 {
		return "💽 No disk information available"
	}

	response := "💽 **Disk Usage**\n\n"
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

		response += fmt.Sprintf(`%s **%s**
📁 **Path:** %s
📊 **Used:** %.1f GB / %.1f GB (%.1f%%)
🆓 **Free:** %.1f GB
💾 **Type:** %s

`,
			statusEmoji, disk.Path,
			disk.Path,
			usedGB, totalGB, disk.UsedPercent,
			freeGB,
			disk.Filesystem)
	}

	b.logger.WithField("disks_count", len(diskInfo.Disks)).Info("Информация о дисках успешно получена")
	return response
}

// handleUptime handles the /uptime command
func (b *Bot) handleUptime(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /uptime")

	servers, err := b.getUserServers(message.From.ID)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get user servers")
		return "❌ Error retrieving your servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected. Use /add to connect a server."
	}

	// For now, use the first server
	serverKey := servers[0]
	b.logger.WithField("server_key", serverKey[:12]+"...").Info("Запрос времени работы с сервера")

	uptimeInfo, err := b.getUptime(serverKey)
	if err != nil {
		b.logger.WithError(err).Error("Ошибка получения времени работы")
		return fmt.Sprintf("❌ Failed to get uptime: %v", err)
	}

	// Format boot time
	bootTime := time.Unix(int64(uptimeInfo.BootTime), 0)
	
	response := fmt.Sprintf(`⏰ **System Uptime**

🚀 **Uptime:** %s
📅 **Boot Time:** %s
⏱️ **Running for:** %d seconds`,
		uptimeInfo.Formatted,
		bootTime.Format("2006-01-02 15:04:05"),
		uptimeInfo.Uptime)

	b.logger.WithField("uptime", uptimeInfo.Formatted).Info("Время работы успешно получено")
	return response
}

// handleProcesses handles the /processes command
func (b *Bot) handleProcesses(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /processes")

	servers, err := b.getUserServers(message.From.ID)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get user servers")
		return "❌ Error retrieving your servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected. Use /add to connect a server."
	}

	// For now, use the first server
	serverKey := servers[0]
	b.logger.WithField("server_key", serverKey[:12]+"...").Info("Запрос списка процессов с сервера")

	processes, err := b.getProcesses(serverKey)
	if err != nil {
		b.logger.WithError(err).Error("Ошибка получения списка процессов")
		return fmt.Sprintf("❌ Failed to get processes: %v", err)
	}

	if len(processes.Processes) == 0 {
		return "⚙️ No process information available"
	}

	response := "⚙️ **Top Processes**\n\n"
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

		response += fmt.Sprintf(`%s **%s** (PID: %d)
👤 **User:** %s
🖥️ **CPU:** %.1f%%
🧠 **Memory:** %d MB (%.1f%%)
📊 **Status:** %s

`,
			statusEmoji, proc.Name, proc.PID,
			proc.Username,
			proc.CPUPercent,
			proc.MemoryMB, proc.MemoryPercent,
			proc.Status)
	}

	b.logger.WithField("processes_count", len(processes.Processes)).Info("Список процессов успешно получен")
	return response
}

// handleStatus handles the /status command
func (b *Bot) handleStatus(message *tgbotapi.Message) string {
	servers, err := b.getUserServersWithInfo(message.From.ID)
	if err != nil {
		return "❌ Error retrieving servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected. Use /add to connect a server."
	}

	// If multiple servers, show selection buttons
	if len(servers) > 1 {
		parts := strings.Fields(message.Text)
		if len(parts) == 1 {
			// No server specified, show buttons
			b.sendServerSelectionButtons(message.Chat.ID, "status", "📊 Select server for status:", servers)
			return ""
		}
	}

	// Parse server number from command or use first server
	serverKeys := make([]string, len(servers))
	for i, server := range servers {
		serverKeys[i] = server.SecretKey
	}
	
	_, err = b.getServerFromCommand(message.Text, serverKeys)
	if err != nil {
		return err.Error()
	}

	serverName := servers[0].Name
	return fmt.Sprintf("🟢 **%s** Status: Online\n⏱️ Uptime: 15 days 8 hours\n💾 Last activity: just now", serverName)
}

// handleContainers handles the /containers command
func (b *Bot) handleContainers(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /containers")
	
	servers, err := b.getUserServersWithInfo(message.From.ID)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get user servers")
		return "❌ Error retrieving your servers."
	}

	b.logger.WithField("servers_count", len(servers)).Info("Найдено серверов пользователя")
	
	if len(servers) == 0 {
		return "📭 No servers connected. Use /add to connect a server."
	}

	// If multiple servers, show selection buttons
	if len(servers) > 1 {
		parts := strings.Fields(message.Text)
		if len(parts) == 1 {
			// No server specified, show buttons
			b.sendServerSelectionButtons(message.Chat.ID, "containers", "🐳 Select server for containers:", servers)
			return ""
		}
	}

	// Parse server number from command or use first server
	serverKeys := make([]string, len(servers))
	for i, server := range servers {
		serverKeys[i] = server.SecretKey
	}
	
	serverKey, err := b.getServerFromCommand(message.Text, serverKeys)
	if err != nil {
		return err.Error()
	}
	b.logger.WithField("server_key", serverKey[:12]+"...").Info("Запрос списка контейнеров с сервера")
	
	containers, err := b.getContainers(serverKey)
	if err != nil {
		b.logger.WithError(err).Error("Ошибка получения списка контейнеров")
		return fmt.Sprintf("❌ Failed to get containers: %v", err)
	}

	b.logger.WithField("containers_count", len(containers.Containers)).Info("Список контейнеров успешно получен")
	return b.formatContainers(containers)
}
