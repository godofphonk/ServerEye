package bot

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// handleMessage processes a single message
func (b *Bot) handleMessage(message *tgbotapi.Message) error {
	b.legacyLogger.WithFields(logrus.Fields{
		"user_id":  message.From.ID,
		"username": message.From.UserName,
		"text":     message.Text,
	}).Info("Получено сообщение от пользователя")

	var response string

	switch {
	case strings.HasPrefix(message.Text, "/start"):
		b.legacyLogger.Info("Обработка команды /start")
		response = b.handleStart(message)
	case strings.HasPrefix(message.Text, "/temp"):
		b.legacyLogger.Info("Обработка команды /temp")
		response = b.handleTemp(message)
	case strings.HasPrefix(message.Text, "/memory"):
		b.legacyLogger.Info("Обработка команды /memory")
		response = b.handleMemory(message)
	case strings.HasPrefix(message.Text, "/disk"):
		b.legacyLogger.Info("Обработка команды /disk")
		response = b.handleDisk(message)
	case strings.HasPrefix(message.Text, "/uptime"):
		b.legacyLogger.Info("Обработка команды /uptime")
		response = b.handleUptime(message)
	case strings.HasPrefix(message.Text, "/processes"):
		b.legacyLogger.Info("Обработка команды /processes")
		response = b.handleProcesses(message)
	case strings.HasPrefix(message.Text, "/containers"):
		b.legacyLogger.Info("Обработка команды /containers")
		response = b.handleContainers(message)
	case strings.HasPrefix(message.Text, "/start_container"):
		b.legacyLogger.Info("Обработка команды /start_container")
		response = b.handleStartContainer(message)
	case strings.HasPrefix(message.Text, "/stop_container"):
		b.legacyLogger.Info("Обработка команды /stop_container")
		response = b.handleStopContainer(message)
	case strings.HasPrefix(message.Text, "/restart_container"):
		b.legacyLogger.Info("Обработка команды /restart_container")
		response = b.handleRestartContainer(message)
	case strings.HasPrefix(message.Text, "/status"):
		b.legacyLogger.Info("Обработка команды /status")
		response = b.handleStatus(message)
	case strings.HasPrefix(message.Text, "/servers"):
		b.legacyLogger.Info("Обработка команды /servers")
		response = b.handleServers(message)
	case strings.HasPrefix(message.Text, "/help"):
		b.legacyLogger.Info("Обработка команды /help")
		response = b.handleHelp(message)
	case strings.HasPrefix(message.Text, "/rename_server"):
		b.legacyLogger.Info("Обработка команды /rename_server")
		response = b.handleRenameServer(message)
	case strings.HasPrefix(message.Text, "/remove_server"):
		b.legacyLogger.Info("Обработка команды /remove_server")
		response = b.handleRemoveServer(message)
	case strings.HasPrefix(message.Text, "/add"):
		b.legacyLogger.Info("Обработка команды /add")
		response = b.handleAddServer(message)
	case strings.HasPrefix(message.Text, "srv_"):
		b.legacyLogger.Info("Обработка ключа сервера (deprecated)")
		response = "❌ Please use /add command instead.\nExample: /add srv_your_key_here"
	default:
		b.legacyLogger.WithField("text", message.Text).Info("Неизвестная команда")
		response = "❓ Unknown command. Use /help to see available commands."
	}

	b.legacyLogger.WithField("response_length", len(response)).Info("Отправка ответа пользователю")
	b.sendMessage(message.Chat.ID, response)
	return nil
}

// handleStart handles the /start command
func (b *Bot) handleStart(message *tgbotapi.Message) string {
	// Register user if not exists
	if err := b.registerUser(message.From); err != nil {
		b.legacyLogger.WithError(err).Error("Failed to register user")
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

💡 **Multiple Servers:**
If you have multiple servers, select from buttons that appear when you use commands.

🔗 **Connect Server:**
Use /add command: /add srv_your_key [name]`
}
