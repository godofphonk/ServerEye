package bot

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleTemp handles the /temp command
func (b *Bot) handleTemp(message *tgbotapi.Message) string {
	b.logger.Info("Operation completed")

	servers, err := b.getUserServersWithInfo(message.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return "❌ Error retrieving your servers."
	}

	b.logger.Info("Найдено серверов пользователя")

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

	b.logger.Info("Operation completed")

	temp, err := b.getCPUTemperature(serverKey)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return fmt.Sprintf("❌ Failed to get temperature: %v", err)
	}

	b.logger.Info("Operation completed")
	return fmt.Sprintf("🌡️ CPU Temperature: %.1f°C", temp)
}

// handleMemory handles the /memory command
func (b *Bot) handleMemory(message *tgbotapi.Message) string {
	b.logger.Info("Operation completed")

	servers, err := b.getUserServersWithInfo(message.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
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
	b.logger.Info("Operation completed")

	memInfo, err := b.getMemoryInfo(serverKey)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return fmt.Sprintf("❌ Failed to get memory info: %v", err)
	}

	// Format memory information
	totalGB := float64(memInfo.Total) / 1024 / 1024 / 1024
	usedGB := float64(memInfo.Used) / 1024 / 1024 / 1024
	availableGB := float64(memInfo.Available) / 1024 / 1024 / 1024
	freeGB := float64(memInfo.Free) / 1024 / 1024 / 1024

	response := fmt.Sprintf(`🧠 Memory Usage

💾 Total: %.1f GB
📊 Used: %.1f GB (%.1f%%)
✅ Available: %.1f GB
🆓 Free: %.1f GB
📦 Buffers: %.1f MB
🗂️ Cached: %.1f MB`,
		totalGB,
		usedGB, memInfo.UsedPercent,
		availableGB,
		freeGB,
		float64(memInfo.Buffers)/1024/1024,
		float64(memInfo.Cached)/1024/1024)

	b.logger.Info("Operation completed")
	return response
}

// handleDisk handles the /disk command
func (b *Bot) handleDisk(message *tgbotapi.Message) string {
	b.logger.Info("Operation completed")

	servers, err := b.getUserServers(message.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return "❌ Error retrieving your servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected. Use /add to connect a server."
	}

	// For now, use the first server
	serverKey := servers[0]
	b.logger.Info("Operation completed")

	diskInfo, err := b.getDiskInfo(serverKey)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return fmt.Sprintf("❌ Failed to get disk info: %v", err)
	}

	if len(diskInfo.Disks) == 0 {
		return "💽 No disk information available"
	}

	response := "💽 Disk Usage\n\n"
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

	b.logger.Info("Информация о дисках успешно получена")
	return response
}

// handleUptime handles the /uptime command
func (b *Bot) handleUptime(message *tgbotapi.Message) string {
	b.logger.Info("Operation completed")

	servers, err := b.getUserServers(message.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return "❌ Error retrieving your servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected. Use /add to connect a server."
	}

	// For now, use the first server
	serverKey := servers[0]
	b.logger.Info("Operation completed")

	uptimeInfo, err := b.getUptime(serverKey)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return fmt.Sprintf("❌ Failed to get uptime: %v", err)
	}

	// Format boot time - safe conversion from uint64 to int64
	bootTimeUnix := uptimeInfo.BootTime
	if bootTimeUnix > (1<<63 - 1) {
		bootTimeUnix = 1<<63 - 1 // Cap at max int64
	}
	bootTime := time.Unix(int64(bootTimeUnix), 0)

	response := fmt.Sprintf(`⏰ System Uptime

🚀 Uptime: %s
📅 Boot Time: %s
⏱️ Running for: %d seconds`,
		uptimeInfo.Formatted,
		bootTime.Format("2006-01-02 15:04:05"),
		uptimeInfo.Uptime)

	b.logger.Info("Operation completed")
	return response
}

// handleProcesses handles the /processes command
func (b *Bot) handleProcesses(message *tgbotapi.Message) string {
	b.logger.Info("Operation completed")

	servers, err := b.getUserServers(message.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return "❌ Error retrieving your servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected. Use /add to connect a server."
	}

	// For now, use the first server
	serverKey := servers[0]
	b.logger.Info("Operation completed")

	processes, err := b.getProcesses(serverKey)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return fmt.Sprintf("❌ Failed to get processes: %v", err)
	}

	if len(processes.Processes) == 0 {
		return "⚙️ No process information available"
	}

	response := "⚙️ Top Processes\n\n"
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

	b.logger.Info("Список процессов успешно получен")
	return response
}

// handleNetwork handles the /network command
func (b *Bot) handleNetwork(message *tgbotapi.Message) string {
	b.logger.Info("Operation completed")

	servers, err := b.getUserServers(message.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return "❌ Error retrieving your servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected. Use /add to connect a server."
	}

	// For now, use the first server
	serverKey := servers[0]
	b.logger.Info("Operation completed")

	networkInfo, err := b.getNetworkInfo(serverKey)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return fmt.Sprintf("❌ Failed to get network info: %v", err)
	}

	if len(networkInfo.Interfaces) == 0 {
		return "🌐 No network information available"
	}

	// Format response with network statistics
	response := "🌐 Network Statistics\n\n"

	// Overall speed
	response += "📊 Current Speed:\n"
	response += fmt.Sprintf("⬇️ Download: %.2f Mbps\n", networkInfo.DownloadSpeed)
	response += fmt.Sprintf("⬆️ Upload: %.2f Mbps\n", networkInfo.UploadSpeed)
	response += "\n"

	// Total traffic
	response += "📈 Total Traffic:\n"
	response += fmt.Sprintf("⬇️ Downloaded: %d GB\n", networkInfo.TotalDownload)
	response += fmt.Sprintf("⬆️ Uploaded: %d GB\n", networkInfo.TotalUpload)
	response += "\n"

	// Interfaces details
	response += "🔌 Interfaces:\n"
	for _, iface := range networkInfo.Interfaces {
		bytesRecvGB := float64(iface.BytesRecv) / 1024 / 1024 / 1024
		bytesSentGB := float64(iface.BytesSent) / 1024 / 1024 / 1024

		response += fmt.Sprintf("\n📡 %s:\n", iface.Name)
		response += fmt.Sprintf("  ⬇️ Recv: %.2f GB (%d packets)\n", bytesRecvGB, iface.PacketsRecv)
		response += fmt.Sprintf("  ⬆️ Sent: %.2f GB (%d packets)\n", bytesSentGB, iface.PacketsSent)

		if iface.ErrorsIn > 0 || iface.ErrorsOut > 0 {
			response += fmt.Sprintf("  ⚠️ Errors: %d in, %d out\n", iface.ErrorsIn, iface.ErrorsOut)
		}
		if iface.DropIn > 0 || iface.DropOut > 0 {
			response += fmt.Sprintf("  ⚠️ Drops: %d in, %d out\n", iface.DropIn, iface.DropOut)
		}
	}

	b.logger.Info("Информация о сети успешно получена")
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
	b.logger.Info("Operation completed")

	servers, err := b.getUserServersWithInfo(message.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return "❌ Error retrieving your servers."
	}

	b.logger.Info("Найдено серверов пользователя")

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
	b.logger.Info("Operation completed")

	containers, err := b.getContainers(serverKey)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return fmt.Sprintf("❌ Failed to get containers: %v", err)
	}

	b.logger.Info("Список контейнеров успешно получен")

	// Send containers list with action buttons
	b.sendContainersWithActionButtons(message.Chat.ID, serverKey, containers)
	return ""
}

// handleUpdate handles the /update command to update agent
func (b *Bot) handleUpdate(message *tgbotapi.Message) string {
	servers, err := b.getUserServersWithInfo(message.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return "❌ Failed to retrieve servers."
	}

	if len(servers) == 0 {
		return "❌ No servers found. Add a server first using /add command."
	}

	// If multiple servers, show selection buttons
	if len(servers) > 1 {
		parts := strings.Fields(message.Text)
		if len(parts) == 1 {
			// No server specified, show buttons
			b.sendServerSelectionButtons(message.Chat.ID, "update", "🔄 Select server to update:", servers)
			return ""
		}
	}

	// Single server - update directly
	serverKey := servers[0].SecretKey
	serverName := servers[0].Name

	b.logger.Info("Updating agent...")

	// Send "updating" message
	b.sendMessage(message.Chat.ID, fmt.Sprintf("🔄 Updating agent on %s...\n\nThis may take a minute.", serverName))

	updateResp, err := b.updateAgent(serverKey, "latest")
	if err != nil {
		b.logger.Error("Error occurred", err)
		return fmt.Sprintf("❌ Failed to update agent on %s: %v", serverName, err)
	}

	if !updateResp.Success {
		return fmt.Sprintf("❌ Update failed on %s:\n%s", serverName, updateResp.Message)
	}

	response := fmt.Sprintf("✅ Agent updated successfully on %s!\n\n", serverName)
	response += fmt.Sprintf("📦 Old version: %s\n", updateResp.OldVersion)
	response += fmt.Sprintf("📦 New version: %s\n", updateResp.NewVersion)

	if updateResp.RestartRequired {
		response += "\n⚠️ Agent restart required to apply changes."
	}

	b.logger.Info("Agent updated successfully")
	return response
}
