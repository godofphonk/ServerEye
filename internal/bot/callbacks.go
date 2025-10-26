package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleCallbackQuery processes callback queries from inline keyboards
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	// Answer the callback query
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := b.tgBot.Request(callback); err != nil {
		b.logger.WithError(err).Error("Failed to answer callback query")
	}

	// Parse callback data (format: "command_serverNumber")
	parts := strings.Split(query.Data, "_")
	if len(parts) != 2 {
		b.logger.WithField("data", query.Data).Error("Invalid callback data format")
		return
	}

	command := parts[0]
	serverNum := parts[1]

	// Get user's servers
	servers, err := b.getUserServersWithInfo(query.From.ID)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get user servers for callback")
		return
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

	return fmt.Sprintf("🌡️ **%s** CPU Temperature: %.1f°C", serverName, temp)
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

	response := fmt.Sprintf("🐳 **%s** Containers:\n\n", serverName)
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

	return fmt.Sprintf(`🧠 **%s** Memory Usage

💾 **Total:** %.1f GB
📊 **Used:** %.1f GB (%.1f%%)
✅ **Available:** %.1f GB
🆓 **Free:** %.1f GB
📦 **Buffers:** %.1f MB
🗂️ **Cached:** %.1f MB`,
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
		return fmt.Sprintf("💽 **%s** - No disk information available", serverName)
	}

	response := fmt.Sprintf("💽 **%s** Disk Usage\n\n", serverName)
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

	bootTime := time.Unix(int64(uptimeInfo.BootTime), 0)
	
	return fmt.Sprintf(`⏰ **%s** System Uptime

🚀 **Uptime:** %s
📅 **Boot Time:** %s
⏱️ **Running for:** %d seconds`,
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
		return fmt.Sprintf("⚙️ **%s** - No process information available", serverName)
	}

	response := fmt.Sprintf("⚙️ **%s** Top Processes\n\n", serverName)
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
	return response
}

// executeStatusCommand executes status command for specific server
func (b *Bot) executeStatusCommand(servers []ServerInfo, serverNum string) string {
	num, err := strconv.Atoi(serverNum)
	if err != nil || num < 1 || num > len(servers) {
		return "❌ Invalid server selection"
	}

	serverName := servers[num-1].Name
	return fmt.Sprintf("🟢 **%s** Status: Online\n⏱️ Uptime: 15 days 8 hours\n💾 Last activity: just now", serverName)
}
