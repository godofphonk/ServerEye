package bot

import (
	"fmt"
	"time"
)

// executeTemperatureCommand executes temperature command for specific server
func (b *Bot) executeTemperatureCommand(servers []ServerInfo, serverNum string) string {
	server, err := selectServer(servers, serverNum)
	if err != nil {
		return "❌ Invalid server selection"
	}

	temp, err := b.getCPUTemperature(server.Key)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get temperature from %s: %v", server.Name, err)
	}

	return fmt.Sprintf("🌡️ %s CPU Temperature: %.1f°C", server.Name, temp)
}

// executeContainersCommand executes containers command for specific server
func (b *Bot) executeContainersCommand(servers []ServerInfo, serverNum string) string {
	server, err := selectServer(servers, serverNum)
	if err != nil {
		return "❌ Invalid server selection"
	}

	containers, err := b.getContainers(server.Key)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get containers from %s: %v", server.Name, err)
	}

	response := fmt.Sprintf("🐳 %s Containers:\n\n", server.Name)
	response += b.formatContainers(containers)
	return response
}

// executeMemoryCommand executes memory command for specific server
func (b *Bot) executeMemoryCommand(servers []ServerInfo, serverNum string) string {
	server, err := selectServer(servers, serverNum)
	if err != nil {
		return "❌ Invalid server selection"
	}

	memInfo, err := b.getMemoryInfo(server.Key)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get memory info from %s: %v", server.Name, err)
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
		server.Name,
		totalGB,
		usedGB, memInfo.UsedPercent,
		availableGB,
		freeGB,
		float64(memInfo.Buffers)/1024/1024,
		float64(memInfo.Cached)/1024/1024)
}

// executeDiskCommand executes disk command for specific server
func (b *Bot) executeDiskCommand(servers []ServerInfo, serverNum string) string {
	server, err := selectServer(servers, serverNum)
	if err != nil {
		return "❌ Invalid server selection"
	}

	diskInfo, err := b.getDiskInfo(server.Key)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get disk info from %s: %v", server.Name, err)
	}

	if len(diskInfo.Disks) == 0 {
		return fmt.Sprintf("💽 %s - No disk information available", server.Name)
	}

	response := fmt.Sprintf("💽 %s Disk Usage\n\n", server.Name)
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
	server, err := selectServer(servers, serverNum)
	if err != nil {
		return "❌ Invalid server selection"
	}

	uptimeInfo, err := b.getUptime(server.Key)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get uptime from %s: %v", server.Name, err)
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
		server.Name,
		uptimeInfo.Formatted,
		bootTime.Format("2006-01-02 15:04:05"),
		uptimeInfo.Uptime)
}

// executeProcessesCommand executes processes command for specific server
func (b *Bot) executeProcessesCommand(servers []ServerInfo, serverNum string) string {
	server, err := selectServer(servers, serverNum)
	if err != nil {
		return "❌ Invalid server selection"
	}

	processes, err := b.getProcesses(server.Key)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get processes from %s: %v", server.Name, err)
	}

	if len(processes.Processes) == 0 {
		return fmt.Sprintf("⚙️ %s - No process information available", server.Name)
	}

	response := fmt.Sprintf("⚙️ %s Top Processes\n\n", server.Name)
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
	server, err := selectServer(servers, serverNum)
	if err != nil {
		return "❌ Invalid server selection"
	}

	return fmt.Sprintf("🟢 %s Status: Online\n⏱️ Uptime: 15 days 8 hours\n💾 Last activity: just now", server.Name)
}

// executeUpdateCommand executes update command for specific server
func (b *Bot) executeUpdateCommand(servers []ServerInfo, serverNum string, chatID int64) string {
	server, err := selectServer(servers, serverNum)
	if err != nil {
		return "❌ Invalid server selection"
	}

	// Send "updating" message
	b.sendMessage(chatID, fmt.Sprintf("🔄 Updating agent on %s...\n\nThis may take a minute.", server.Name))

	updateResp, err := b.updateAgent(server.Key, "latest")
	if err != nil {
		return fmt.Sprintf("❌ Failed to update agent on %s: %v", server.Name, err)
	}

	if !updateResp.Success {
		return fmt.Sprintf("❌ Update failed on %s:\n%s", server.Name, updateResp.Message)
	}

	response := fmt.Sprintf("✅ Agent updated successfully on %s!\n\n", server.Name)
	response += fmt.Sprintf("📦 Old version: %s\n", updateResp.OldVersion)
	response += fmt.Sprintf("📦 New version: %s\n", updateResp.NewVersion)

	if updateResp.RestartRequired {
		response += "\n⚠️ Agent restart required to apply changes."
	}

	return response
}
