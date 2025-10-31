package bot

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleMessage processes a single message
func (b *Bot) handleMessage(message *tgbotapi.Message) error {
	b.logger.Info("Получено сообщение от пользователя")

	var response string

	switch {
	case strings.HasPrefix(message.Text, "/start"):
		b.logger.Info("Info message")
		response = b.handleStart(message)
	case strings.HasPrefix(message.Text, "/temp"):
		b.logger.Info("Info message")
		response = b.handleTemp(message)
	case strings.HasPrefix(message.Text, "/memory"):
		b.logger.Info("Info message")
		response = b.handleMemory(message)
	case strings.HasPrefix(message.Text, "/disk"):
		b.logger.Info("Info message")
		response = b.handleDisk(message)
	case strings.HasPrefix(message.Text, "/uptime"):
		b.logger.Info("Info message")
		response = b.handleUptime(message)
	case strings.HasPrefix(message.Text, "/processes"):
		b.logger.Info("Info message")
		response = b.handleProcesses(message)
	case strings.HasPrefix(message.Text, "/containers"):
		b.logger.Info("Info message")
		response = b.handleContainers(message)
	case strings.HasPrefix(message.Text, "/start_container"):
		b.logger.Info("Info message")
		response = b.handleStartContainer(message)
	case strings.HasPrefix(message.Text, "/stop_container"):
		b.logger.Info("Info message")
		response = b.handleStopContainer(message)
	case strings.HasPrefix(message.Text, "/restart_container"):
		b.logger.Info("Info message")
		response = b.handleRestartContainer(message)
	case strings.HasPrefix(message.Text, "/status"):
		b.logger.Info("Info message")
		response = b.handleStatus(message)
	case strings.HasPrefix(message.Text, "/servers"):
		b.logger.Info("Info message")
		response = b.handleServers(message)
	case strings.HasPrefix(message.Text, "/help"):
		b.logger.Info("Info message")
		response = b.handleHelp(message)
	case strings.HasPrefix(message.Text, "/rename_server"):
		b.logger.Info("Info message")
		response = b.handleRenameServer(message)
	case strings.HasPrefix(message.Text, "/remove_server"):
		b.logger.Info("Info message")
		response = b.handleRemoveServer(message)
	case strings.HasPrefix(message.Text, "/add"):
		b.logger.Info("Info message")
		response = b.handleAddServer(message)
	case strings.HasPrefix(message.Text, "/debug"):
		b.logger.Info("Info message")
		response = b.handleDebug(message)
	case strings.HasPrefix(message.Text, "/stats"):
		b.logger.Info("Info message")
		response = b.handleStats(message)
	case strings.HasPrefix(message.Text, "srv_"):
		b.logger.Info("Info message")
		response = "❌ Please use /add command instead.\nExample: /add srv_your_key_here"
	default:
		b.logger.Info("Operation completed")
		response = "❓ Unknown command. Use /help to see available commands."
	}

	b.logger.Info("Отправка ответа пользователю")
	b.sendMessage(message.Chat.ID, response)
	return nil
}

// handleStart handles the /start command
func (b *Bot) handleStart(message *tgbotapi.Message) string {
	// Register user if not exists
	if err := b.registerUser(message.From); err != nil {
		b.logger.Error("Error occurred", err)
		return "❌ Error occurred during registration. Please try again."
	}

	return `👋 Welcome to ServerEye!

To connect your server, use the /add command with the secret key you received during agent installation.

Example: /add srv_a1b2c3d4e5f6g7h8 MyServer

Available commands:
/add <key> [name] - Add server with optional name
/temp - Get CPU temperature
/memory - Get memory usage
/disk - Get disk usage
/uptime - Get system uptime
/processes - Get top processes
/containers - List Docker containers
/start_container <id> - Start container
/stop_container <id> - Stop container
/restart_container <id> - Restart container
/status - Get server status
/servers - List your servers
/help - Show this help`
}

// handleHelp handles the /help command
func (b *Bot) handleHelp(message *tgbotapi.Message) string {
	return `🤖 ServerEye Bot Commands:

📊 **Monitoring:**
/temp - Get CPU temperature
/memory - Get memory usage  
/disk - Get disk usage
/uptime - Get system uptime
/processes - List running processes

🐳 **Docker Management:**
/containers - List Docker containers
/start_container <id> - Start a container
/stop_container <id> - Stop a container
/restart_container <id> - Restart a container

⚙️ **Server Management:**
/servers - List your servers
/status - Get server status
/rename_server <#> <name> - Rename server
/remove_server <#> - Remove server
/add <key> [name] - Add new server

🔍 **Debug:**
/debug - Show connection status

💡 **Multiple Servers:**
If you have multiple servers, select from buttons that appear when you use commands.

🔗 **Connect Server:**
Use /add command: /add srv_your_key [name]`
}
