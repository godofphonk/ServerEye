package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/servereye/servereye/pkg/redis"
	"github.com/sirupsen/logrus"
	_ "github.com/lib/pq"
)

// Bot represents the Telegram bot instance
type Bot struct {
	config      *config.BotConfig
	logger      *logrus.Logger
	tgBot       *tgbotapi.BotAPI
	redisClient *redis.Client
	db          *sql.DB
	ctx         context.Context
	cancel      context.CancelFunc
}

// New creates a new bot instance
func New(cfg *config.BotConfig, logger *logrus.Logger) (*Bot, error) {
	// Initialize Telegram bot
	tgBot, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %v", err)
	}

	logger.WithField("username", tgBot.Self.UserName).Info("Telegram bot authorized")

	// Initialize Redis client
	redisClient, err := redis.NewClient(redis.Config{
		Address:  cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %v", err)
	}

	// Initialize database connection
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	logger.Info("Database connection established")

	ctx, cancel := context.WithCancel(context.Background())

	return &Bot{
		config:      cfg,
		logger:      logger,
		tgBot:       tgBot,
		redisClient: redisClient,
		db:          db,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// Start starts the bot
func (b *Bot) Start() error {
	b.logger.Info("Starting ServerEye Telegram bot")

	// Initialize database schema
	if err := b.initDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	b.logger.Info("Настройка получения обновлений от Telegram")

	// Start handling updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	b.logger.Info("Получение канала обновлений")
	updates := b.tgBot.GetUpdatesChan(u)

	b.logger.Info("Запуск обработчика обновлений в горутине")
	// Handle updates in a separate goroutine
	go b.handleUpdates(updates)

	b.logger.Info("Обработчик обновлений запущен")
	return nil
}

// Stop stops the bot
func (b *Bot) Stop() error {
	b.logger.Info("Stopping bot")
	b.cancel()
	b.tgBot.StopReceivingUpdates()
	b.redisClient.Close()
	return b.db.Close()
}

// handleUpdates processes incoming Telegram updates
func (b *Bot) handleUpdates(updates tgbotapi.UpdatesChannel) {
	b.logger.Info("Начало обработки обновлений от Telegram")

	for {
		select {
		case update := <-updates:
			b.logger.Info("Получено обновление от Telegram")

			if update.Message == nil {
				b.logger.Info("Обновление без сообщения, пропускаем")
				continue
			}

			b.logger.WithFields(logrus.Fields{
				"user_id":  update.Message.From.ID,
				"username": update.Message.From.UserName,
				"text":     update.Message.Text,
			}).Info("Получено сообщение от Telegram")

			b.handleMessage(update.Message)

		case <-b.ctx.Done():
			b.logger.Info("Остановка обработки обновлений")
			return
		}
	}
}

// handleMessage processes a single message
func (b *Bot) handleMessage(message *tgbotapi.Message) {
	b.logger.WithFields(logrus.Fields{
		"user_id":  message.From.ID,
		"username": message.From.UserName,
		"text":     message.Text,
	}).Info("Получено сообщение от пользователя")

	var response string

	switch {
	case strings.HasPrefix(message.Text, "/start"):
		b.logger.Info("Обработка команды /start")
		response = b.handleStart(message)
	case strings.HasPrefix(message.Text, "/temp"):
		b.logger.Info("Обработка команды /temp")
		response = b.handleTemp(message)
	case strings.HasPrefix(message.Text, "/memory"):
		b.logger.Info("Обработка команды /memory")
		response = b.handleMemory(message)
	case strings.HasPrefix(message.Text, "/disk"):
		b.logger.Info("Обработка команды /disk")
		response = b.handleDisk(message)
	case strings.HasPrefix(message.Text, "/uptime"):
		b.logger.Info("Обработка команды /uptime")
		response = b.handleUptime(message)
	case strings.HasPrefix(message.Text, "/processes"):
		b.logger.Info("Обработка команды /processes")
		response = b.handleProcesses(message)
	case strings.HasPrefix(message.Text, "/containers"):
		b.logger.Info("Обработка команды /containers")
		response = b.handleContainers(message)
	case strings.HasPrefix(message.Text, "/start_container"):
		b.logger.Info("Обработка команды /start_container")
		response = b.handleStartContainer(message)
	case strings.HasPrefix(message.Text, "/stop_container"):
		b.logger.Info("Обработка команды /stop_container")
		response = b.handleStopContainer(message)
	case strings.HasPrefix(message.Text, "/restart_container"):
		b.logger.Info("Обработка команды /restart_container")
		response = b.handleRestartContainer(message)
	case strings.HasPrefix(message.Text, "/status"):
		b.logger.Info("Обработка команды /status")
		response = b.handleStatus(message)
	case strings.HasPrefix(message.Text, "/servers"):
		b.logger.Info("Обработка команды /servers")
		response = b.handleServers(message)
	case strings.HasPrefix(message.Text, "/help"):
		b.logger.Info("Обработка команды /help")
		response = b.handleHelp(message)
	case strings.HasPrefix(message.Text, "/rename_server"):
		b.logger.Info("Обработка команды /rename_server")
		response = b.handleRenameServer(message)
	case strings.HasPrefix(message.Text, "/remove_server"):
		b.logger.Info("Обработка команды /remove_server")
		response = b.handleRemoveServer(message)
	case strings.HasPrefix(message.Text, "srv_"):
		b.logger.Info("Обработка ключа сервера")
		response = b.handleServerKey(message)
	default:
		b.logger.WithField("text", message.Text).Info("Неизвестная команда")
		response = "❓ Unknown command. Use /help to see available commands."
	}

	b.logger.WithField("response_length", len(response)).Info("Отправка ответа пользователю")
	b.sendMessage(message.Chat.ID, response)
}

// handleStart handles the /start command
func (b *Bot) handleStart(message *tgbotapi.Message) string {
	// Register user if not exists
	if err := b.registerUser(message.From); err != nil {
		b.logger.WithError(err).Error("Failed to register user")
		return "❌ Error occurred during registration. Please try again."
	}

	return `👋 Welcome to ServerEye!

To connect your server, send the secret key you received during agent installation.

Example: srv_a1b2c3d4e5f6g7h8

Available commands:
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

// handleTemp handles the /temp command
func (b *Bot) handleTemp(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /temp")

	servers, err := b.getUserServers(message.From.ID)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get user servers")
		return "❌ Error retrieving your servers."
	}

	b.logger.WithField("servers_count", len(servers)).Info("Найдено серверов пользователя")

	if len(servers) == 0 {
		return "📭 No servers connected. Send your server key to connect a server."
	}

	// Parse server number from command (e.g., "/temp 2")
	serverKey, err := b.getServerFromCommand(message.Text, servers)
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

// handleContainers handles the /containers command
func (b *Bot) handleContainers(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /containers")
	
	servers, err := b.getUserServers(message.From.ID)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get user servers")
		return "❌ Error retrieving your servers."
	}

	b.logger.WithField("servers_count", len(servers)).Info("Найдено серверов пользователя")
	
	if len(servers) == 0 {
		return "📭 No servers connected. Send your server key to connect a server."
	}

	// Parse server number from command
	serverKey, err := b.getServerFromCommand(message.Text, servers)
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

// handleStatus handles the /status command
func (b *Bot) handleStatus(message *tgbotapi.Message) string {
	return "🟢 Server Status: Online\n⏱️ Uptime: 15 days 8 hours\n💾 Last activity: just now"
}

// handleServers handles the /servers command
func (b *Bot) handleServers(message *tgbotapi.Message) string {
	servers, err := b.getUserServersWithInfo(message.From.ID)
	if err != nil {
		return "❌ Error retrieving servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected.\n\n💡 To connect a server:\n1. Install ServerEye agent\n2. Send your server key here"
	}

	if len(servers) == 1 {
		statusIcon := "🟢"
		if servers[0].Status == "offline" {
			statusIcon = "🔴"
		}
		return fmt.Sprintf("📋 Your servers:\n%s **%s** (%s)\n\n💡 All commands will use this server automatically.\n\n🔧 Management:\n/rename_server 1 <name> - Rename server\n/remove_server 1 - Remove server", 
			statusIcon, servers[0].Name, servers[0].SecretKey[:12]+"...")
	}

	// Multiple servers - show list with numbers
	response := "📋 Your servers:\n\n"
	for i, server := range servers {
		statusIcon := "🟢"
		if server.Status == "offline" {
			statusIcon = "🔴"
		}
		response += fmt.Sprintf("%d. %s **%s** (%s)\n", i+1, statusIcon, server.Name, server.SecretKey[:12]+"...")
	}
	response += "\n💡 Use commands with server number:\n"
	response += "Example: /temp 1 or /containers 2\n\n"
	response += "🔧 Management:\n"
	response += "/rename_server <#> <name> - Rename server\n"
	response += "/remove_server <#> - Remove server"
	
	return response
}

// getServerFromCommand parses server number from command and returns server key
func (b *Bot) getServerFromCommand(command string, servers []string) (string, error) {
	parts := strings.Fields(command)
	
	// If no server number specified, use first server
	if len(parts) == 1 {
		if len(servers) > 1 {
			return "", fmt.Errorf("❌ Multiple servers found. Please specify server number.\nExample: %s 1\n\nUse /servers to see your servers.", parts[0])
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

// handleHelp handles the /help command
func (b *Bot) handleHelp(message *tgbotapi.Message) string {
	return `🤖 ServerEye Bot Commands:

📊 **Monitoring:**
/temp [server#] - Get CPU temperature
/memory [server#] - Get memory usage  
/disk [server#] - Get disk usage
/uptime [server#] - Get system uptime
/processes [server#] - List running processes

🐳 **Docker Management:**
/containers [server#] - List Docker containers
/start_container <id> [server#] - Start a container
/stop_container <id> [server#] - Stop a container
/restart_container <id> [server#] - Restart a container

⚙️ **Server Management:**
/servers - List your servers
/status [server#] - Get server status
/rename_server <#> <name> - Rename server
/remove_server <#> - Remove server

💡 **Multiple Servers:**
If you have multiple servers, add server number:
Example: /temp 1 or /containers 2

🔗 **Connect Server:**
Send your server key (starts with srv_)`
}

// handleRenameServer handles the /rename_server command
func (b *Bot) handleRenameServer(message *tgbotapi.Message) string {
	parts := strings.Fields(message.Text)
	if len(parts) < 3 {
		return "❌ Usage: /rename_server <server#> <new_name>\nExample: /rename_server 1 MyWebServer"
	}
	
	servers, err := b.getUserServers(message.From.ID)
	if err != nil || len(servers) == 0 {
		return "❌ No servers found."
	}
	
	serverNum, err := strconv.Atoi(parts[1])
	if err != nil || serverNum < 1 || serverNum > len(servers) {
		return fmt.Sprintf("❌ Invalid server number. You have %d servers.", len(servers))
	}
	
	newName := strings.Join(parts[2:], " ")
	if len(newName) > 50 {
		return "❌ Server name too long (max 50 characters)."
	}
	
	serverKey := servers[serverNum-1]
	if err := b.renameServer(serverKey, newName); err != nil {
		return "❌ Failed to rename server."
	}
	
	return fmt.Sprintf("✅ Server renamed to: %s", newName)
}

// handleRemoveServer handles the /remove_server command
func (b *Bot) handleRemoveServer(message *tgbotapi.Message) string {
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		return "❌ Usage: /remove_server <server#>\nExample: /remove_server 1\n\n⚠️ This will permanently remove the server!"
	}
	
	servers, err := b.getUserServers(message.From.ID)
	if err != nil || len(servers) == 0 {
		return "❌ No servers found."
	}
	
	serverNum, err := strconv.Atoi(parts[1])
	if err != nil || serverNum < 1 || serverNum > len(servers) {
		return fmt.Sprintf("❌ Invalid server number. You have %d servers.", len(servers))
	}
	
	serverKey := servers[serverNum-1]
	if err := b.removeServer(message.From.ID, serverKey); err != nil {
		return "❌ Failed to remove server."
	}
	
	return "✅ Server removed successfully."
}

// handleServerKey handles server key registration
func (b *Bot) handleServerKey(message *tgbotapi.Message) string {
	serverKey := strings.TrimSpace(message.Text)

	if err := b.connectServer(message.From.ID, serverKey); err != nil {
		b.logger.WithError(err).Error("Failed to connect server")
		return "❌ Failed to connect server. Please check your key."
	}

	return "✅ Server connected successfully!\n🟢 Status: Online\n\nUse /temp to get CPU temperature."
}

// sendMessage sends a message to a chat
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.tgBot.Send(msg); err != nil {
		b.logger.WithError(err).Error("Failed to send message")
	}
}

// getCPUTemperature requests CPU temperature from agent
func (b *Bot) getCPUTemperature(serverKey string) (float64, error) {
	// Create command message
	cmd := protocol.NewMessage(protocol.TypeGetCPUTemp, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return 0, fmt.Errorf("failed to serialize command: %v", err)
	}

	// Subscribe to response channel first
	respChannel := redis.GetResponseChannel(serverKey)
	b.logger.WithField("channel", respChannel).Info("Подписались на канал Redis")

	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return 0, fmt.Errorf("failed to subscribe to response: %v", err)
	}
	defer subscription.Close()

	// Small delay to ensure subscription is active
	time.Sleep(100 * time.Millisecond)

	// Send command to agent
	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return 0, fmt.Errorf("failed to send command: %v", err)
	}

	b.logger.WithFields(logrus.Fields{
		"command_id": cmd.ID,
		"channel":    cmdChannel,
	}).Info("Команда отправлена агенту")

	// Wait for response with timeout
	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			b.logger.WithField("data", string(respData)).Debug("Получен ответ от агента")

			resp, err := protocol.FromJSON(respData)
			if err != nil {
				b.logger.WithError(err).Error("Failed to parse response")
				continue
			}

			// Check if this response is for our command
			if resp.ID != cmd.ID {
				b.logger.WithFields(logrus.Fields{
					"expected": cmd.ID,
					"received": resp.ID,
				}).Debug("Response ID mismatch, waiting for correct response")
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return 0, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeCPUTempResponse {
				// Parse temperature from payload
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					if temp, ok := payload["temperature"].(float64); ok {
						b.logger.WithField("temperature", temp).Info("Получена температура CPU")
						return temp, nil
					}
				}
				return 0, fmt.Errorf("invalid temperature data in response")
			}

			return 0, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return 0, fmt.Errorf("timeout waiting for response")
		}
	}
}

// getContainers requests Docker containers list from agent
func (b *Bot) getContainers(serverKey string) (*protocol.ContainersPayload, error) {
	// Create command message
	cmd := protocol.NewMessage(protocol.TypeGetContainers, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

	// Subscribe to response channel first
	respChannel := redis.GetResponseChannel(serverKey)
	b.logger.WithField("channel", respChannel).Info("Подписались на канал Redis")
	
	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %v", err)
	}

	// Small delay to ensure subscription is active
	time.Sleep(100 * time.Millisecond)

	// Send command to agent
	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	b.logger.WithFields(logrus.Fields{
		"command_id": cmd.ID,
		"channel": cmdChannel,
	}).Info("Команда отправлена агенту")

	// Wait for response with timeout
	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			b.logger.WithField("data", string(respData)).Debug("Получен ответ от агента")
			
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				b.logger.WithError(err).Error("Failed to parse response")
				continue
			}

			// Check if this response is for our command
			if resp.ID != cmd.ID {
				b.logger.WithFields(logrus.Fields{
					"expected": cmd.ID,
					"received": resp.ID,
				}).Debug("Response ID mismatch, waiting for correct response")
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return nil, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeContainersResponse {
				// Parse containers from payload
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					containersData, _ := json.Marshal(payload)
					var containers protocol.ContainersPayload
					if err := json.Unmarshal(containersData, &containers); err == nil {
						b.logger.WithField("containers_count", containers.Total).Info("Получен список контейнеров")
						return &containers, nil
					}
				}
				return nil, fmt.Errorf("invalid containers data in response")
			}

			return nil, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}

// formatContainers formats containers list for display
func (b *Bot) formatContainers(containers *protocol.ContainersPayload) string {
	if containers.Total == 0 {
		return "📦 No Docker containers found on the server."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("🐳 **Docker Containers (%d total):**\n\n", containers.Total))

	for i, container := range containers.Containers {
		if i >= 10 { // Limit to 10 containers to avoid message length issues
			result.WriteString(fmt.Sprintf("... and %d more containers\n", containers.Total-10))
			break
		}

		// Status emoji
		statusEmoji := "🔴" // Red for stopped
		if strings.Contains(strings.ToLower(container.State), "running") {
			statusEmoji = "🟢" // Green for running
		} else if strings.Contains(strings.ToLower(container.State), "paused") {
			statusEmoji = "🟡" // Yellow for paused
		}

		result.WriteString(fmt.Sprintf("%s **%s**\n", statusEmoji, container.Name))
		result.WriteString(fmt.Sprintf("📷 Image: `%s`\n", container.Image))
		result.WriteString(fmt.Sprintf("🔄 Status: %s\n", container.Status))
		
		if len(container.Ports) > 0 {
			result.WriteString(fmt.Sprintf("🔌 Ports: %s\n", strings.Join(container.Ports, ", ")))
		}
		
		result.WriteString("\n")
	}

	return result.String()
}

// getMemoryInfo requests memory information from agent
func (b *Bot) getMemoryInfo(serverKey string) (*protocol.MemoryInfo, error) {
	cmd := protocol.NewMessage(protocol.TypeGetMemoryInfo, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

	respChannel := redis.GetResponseChannel(serverKey)
	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %v", err)
	}
	defer subscription.Close()

	time.Sleep(100 * time.Millisecond)

	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				continue
			}

			if resp.ID != cmd.ID {
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return nil, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeMemoryInfoResponse {
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					memData, _ := json.Marshal(payload)
					var memInfo protocol.MemoryInfo
					if err := json.Unmarshal(memData, &memInfo); err == nil {
						return &memInfo, nil
					}
				}
				return nil, fmt.Errorf("invalid memory data in response")
			}

			return nil, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}

// getDiskInfo requests disk information from agent
func (b *Bot) getDiskInfo(serverKey string) (*protocol.DiskInfoPayload, error) {
	cmd := protocol.NewMessage(protocol.TypeGetDiskInfo, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

	respChannel := redis.GetResponseChannel(serverKey)
	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %v", err)
	}
	defer subscription.Close()

	time.Sleep(100 * time.Millisecond)

	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				continue
			}

			if resp.ID != cmd.ID {
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return nil, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeDiskInfoResponse {
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					diskData, _ := json.Marshal(payload)
					var diskInfo protocol.DiskInfoPayload
					if err := json.Unmarshal(diskData, &diskInfo); err == nil {
						return &diskInfo, nil
					}
				}
				return nil, fmt.Errorf("invalid disk data in response")
			}

			return nil, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}

// getUptime requests uptime information from agent
func (b *Bot) getUptime(serverKey string) (*protocol.UptimeInfo, error) {
	cmd := protocol.NewMessage(protocol.TypeGetUptime, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

	respChannel := redis.GetResponseChannel(serverKey)
	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %v", err)
	}
	defer subscription.Close()

	time.Sleep(100 * time.Millisecond)

	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				continue
			}

			if resp.ID != cmd.ID {
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return nil, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeUptimeResponse {
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					uptimeData, _ := json.Marshal(payload)
					var uptimeInfo protocol.UptimeInfo
					if err := json.Unmarshal(uptimeData, &uptimeInfo); err == nil {
						return &uptimeInfo, nil
					}
				}
				return nil, fmt.Errorf("invalid uptime data in response")
			}

			return nil, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}

// getProcesses requests processes information from agent
func (b *Bot) getProcesses(serverKey string) (*protocol.ProcessesPayload, error) {
	cmd := protocol.NewMessage(protocol.TypeGetProcesses, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

	respChannel := redis.GetResponseChannel(serverKey)
	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %v", err)
	}
	defer subscription.Close()

	time.Sleep(100 * time.Millisecond)

	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				continue
			}

			if resp.ID != cmd.ID {
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return nil, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeProcessesResponse {
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					processData, _ := json.Marshal(payload)
					var processes protocol.ProcessesPayload
					if err := json.Unmarshal(processData, &processes); err == nil {
						return &processes, nil
					}
				}
				return nil, fmt.Errorf("invalid processes data in response")
			}

			return nil, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}

// handleStartContainer handles the /start_container command
func (b *Bot) handleStartContainer(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /start_container")
	
	// Парсим команду
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		return "❌ Usage: /start_container <container_id_or_name>\n\nExample: /start_container nginx"
	}
	
	containerID := parts[1]
	return b.handleContainerAction(message.From.ID, containerID, "start")
}

// handleStopContainer handles the /stop_container command
func (b *Bot) handleStopContainer(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /stop_container")
	
	// Парсим команду
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		return "❌ Usage: /stop_container <container_id_or_name>\n\nExample: /stop_container nginx"
	}
	
	containerID := parts[1]
	return b.handleContainerAction(message.From.ID, containerID, "stop")
}

// handleRestartContainer handles the /restart_container command
func (b *Bot) handleRestartContainer(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /restart_container")
	
	// Парсим команду
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		return "❌ Usage: /restart_container <container_id_or_name>\n\nExample: /restart_container nginx"
	}
	
	containerID := parts[1]
	return b.handleContainerAction(message.From.ID, containerID, "restart")
}

// handleContainerAction handles container management actions
func (b *Bot) handleContainerAction(userID int64, containerID, action string) string {
	// Валидация входных данных
	if err := b.validateContainerAction(containerID, action); err != nil {
		return fmt.Sprintf("❌ %s", err.Error())
	}
	
	// Получаем серверы пользователя
	servers, err := b.getUserServers(userID)
	if err != nil {
		b.logger.WithError(err).Error("Ошибка получения серверов пользователя")
		return "❌ Error getting your servers. Please try again."
	}
	
	if len(servers) == 0 {
		return "❌ You don't have any connected servers. Send your server key first."
	}
	
	b.logger.WithField("servers_count", len(servers)).Info("Найдено серверов пользователя")
	
	// Пока работаем только с первым сервером
	serverKey := servers[0]
	b.logger.WithField("server_key", serverKey[:12]+"...").Info("Выполнение действия над контейнером")
	
	// Определяем тип команды
	var messageType protocol.MessageType
	switch action {
	case "start":
		messageType = protocol.TypeStartContainer
	case "stop":
		messageType = protocol.TypeStopContainer
	case "restart":
		messageType = protocol.TypeRestartContainer
	default:
		return fmt.Sprintf("❌ Invalid action: %s", action)
	}
	
	// Создаем payload
	payload := protocol.ContainerActionPayload{
		ContainerID:   containerID,
		ContainerName: containerID, // Может быть именем или ID
		Action:        action,
	}
	
	response, err := b.sendContainerAction(serverKey, messageType, payload)
	if err != nil {
		b.logger.WithError(err).Error("Ошибка выполнения действия над контейнером")
		return fmt.Sprintf("❌ Failed to %s container: %v", action, err)
	}
	
	b.logger.WithField("container_id", containerID).Info("Действие над контейнером успешно выполнено")
	return b.formatContainerActionResponse(response)
}

// sendContainerAction sends container action command to agent
func (b *Bot) sendContainerAction(serverKey string, messageType protocol.MessageType, payload protocol.ContainerActionPayload) (*protocol.ContainerActionResponse, error) {
	// Подписываемся на канал ответов
	responseChannel := fmt.Sprintf("resp:%s", serverKey)
	subscription, err := b.redisClient.Subscribe(b.ctx, responseChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response channel: %w", err)
	}
	defer subscription.Close()
	
	b.logger.WithField("channel", responseChannel).Info("Подписались на канал Redis")
	
	// Отправляем команду
	message := protocol.NewMessage(messageType, payload)
	commandChannel := fmt.Sprintf("cmd:%s", serverKey)
	
	messageData, err := message.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}
	
	if err := b.redisClient.Publish(b.ctx, commandChannel, messageData); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}
	
	b.logger.WithFields(logrus.Fields{
		"channel":    commandChannel,
		"command_id": message.ID,
	}).Info("Команда отправлена агенту")
	
	// Ожидаем ответ
	ctx, cancel := context.WithTimeout(b.ctx, 30*time.Second)
	defer cancel()
	
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for agent response")
		case msgBytes := <-subscription.Channel():
			var response protocol.Message
			if err := json.Unmarshal(msgBytes, &response); err != nil {
				b.logger.WithError(err).Error("Ошибка парсинга ответа")
				continue
			}
			
			if response.ID != message.ID {
				continue // Не наш ответ
			}
			
			if response.Type == protocol.TypeErrorResponse {
				// Парсим ошибку
				if errorData, ok := response.Payload.(map[string]interface{}); ok {
					errorMsg := "unknown error"
					if msg, exists := errorData["error_message"]; exists {
						errorMsg = fmt.Sprintf("%v", msg)
					}
					return nil, fmt.Errorf("agent error: %s", errorMsg)
				}
				return nil, fmt.Errorf("agent returned error")
			}
			
			if response.Type == protocol.TypeContainerActionResponse {
				// Парсим ответ
				if payload, ok := response.Payload.(map[string]interface{}); ok {
					actionData, _ := json.Marshal(payload)
					var actionResponse protocol.ContainerActionResponse
					if err := json.Unmarshal(actionData, &actionResponse); err == nil {
						b.logger.WithField("success", actionResponse.Success).Info("Получен ответ о действии над контейнером")
						return &actionResponse, nil
					}
				}
				return nil, fmt.Errorf("invalid container action response format")
			}
		}
	}
}

// formatContainerActionResponse formats container action response for display
func (b *Bot) formatContainerActionResponse(response *protocol.ContainerActionResponse) string {
	if !response.Success {
		return fmt.Sprintf("❌ Failed to %s container **%s**:\n%s", 
			response.Action, response.ContainerName, response.Message)
	}
	
	var actionEmoji string
	switch response.Action {
	case "start":
		actionEmoji = "▶️"
	case "stop":
		actionEmoji = "⏹️"
	case "restart":
		actionEmoji = "🔄"
	default:
		actionEmoji = "⚙️"
	}
	
	result := fmt.Sprintf("%s Successfully **%sed** container **%s**", 
		actionEmoji, response.Action, response.ContainerName)
	
	if response.NewState != "" {
		var stateEmoji string
		switch response.NewState {
		case "running":
			stateEmoji = "🟢"
		case "exited":
			stateEmoji = "🔴"
		default:
			stateEmoji = "🟡"
		}
		result += fmt.Sprintf("\n%s New state: %s", stateEmoji, response.NewState)
	}
	
	return result
}

// validateContainerAction validates container action parameters
func (b *Bot) validateContainerAction(containerID, action string) error {
	// Проверяем длину ID контейнера
	if len(containerID) < 3 {
		return fmt.Errorf("Container ID/name too short (minimum 3 characters)")
	}
	
	if len(containerID) > 64 {
		return fmt.Errorf("Container ID/name too long (maximum 64 characters)")
	}
	
	// Проверяем символы в ID/имени
	for _, char := range containerID {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
			(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.') {
			return fmt.Errorf("Container ID/name contains invalid characters. Only alphanumeric, hyphens, underscores and dots allowed")
		}
	}
	
	// Проверяем действие
	validActions := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
	}
	
	if !validActions[action] {
		return fmt.Errorf("Invalid action '%s'. Allowed: start, stop, restart", action)
	}
	
	// Проверяем черный список контейнеров
	blacklist := []string{
		"servereye-bot",
		"redis",
		"postgres",
		"postgresql",
		"database",
		"db",
	}
	
	containerLower := strings.ToLower(containerID)
	for _, blocked := range blacklist {
		if strings.Contains(containerLower, blocked) {
			return fmt.Errorf("Container '%s' is protected and cannot be managed", containerID)
		}
	}
	
	return nil
}

// isSystemContainer checks if container is a system container that shouldn't be managed
func (b *Bot) isSystemContainer(containerName string) bool {
	systemContainers := []string{
		"servereye-bot",
		"deployments-servereye-bot",
		"redis",
		"deployments-redis",
		"postgres",
		"deployments-postgres",
		"postgresql",
		"database",
	}
	
	containerLower := strings.ToLower(containerName)
	for _, system := range systemContainers {
		if strings.Contains(containerLower, system) {
			return true
		}
	}
	
	return false
}

// handleMemory handles the /memory command
func (b *Bot) handleMemory(message *tgbotapi.Message) string {
	b.logger.WithField("user_id", message.From.ID).Info("Обработка команды /memory")

	servers, err := b.getUserServers(message.From.ID)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get user servers")
		return "❌ Error retrieving your servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected. Send your server key to connect a server."
	}

	// For now, use the first server
	serverKey := servers[0]
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
		return "📭 No servers connected. Send your server key to connect a server."
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
		return "📭 No servers connected. Send your server key to connect a server."
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
		return "📭 No servers connected. Send your server key to connect a server."
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
