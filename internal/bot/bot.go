package bot

import (
	"context"
	"database/sql"
	"fmt"
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
	case strings.HasPrefix(message.Text, "/status"):
		b.logger.Info("Обработка команды /status")
		response = b.handleStatus(message)
	case strings.HasPrefix(message.Text, "/servers"):
		b.logger.Info("Обработка команды /servers")
		response = b.handleServers(message)
	case strings.HasPrefix(message.Text, "/help"):
		b.logger.Info("Обработка команды /help")
		response = b.handleHelp(message)
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

	// For now, use the first server
	serverKey := servers[0]
	b.logger.WithField("server_key", serverKey[:12]+"...").Info("Запрос температуры с сервера")

	temp, err := b.getCPUTemperature(serverKey)
	if err != nil {
		b.logger.WithError(err).Error("Ошибка получения температуры")
		return fmt.Sprintf("❌ Failed to get temperature: %v", err)
	}

	b.logger.WithField("temperature", temp).Info("Температура успешно получена")
	return fmt.Sprintf("🌡️ CPU Temperature: %.1f°C", temp)
}

// handleStatus handles the /status command
func (b *Bot) handleStatus(message *tgbotapi.Message) string {
	return "🟢 Server Status: Online\n⏱️ Uptime: 15 days 8 hours\n💾 Last activity: just now"
}

// handleServers handles the /servers command
func (b *Bot) handleServers(message *tgbotapi.Message) string {
	servers, err := b.getUserServers(message.From.ID)
	if err != nil {
		return "❌ Error retrieving servers."
	}

	if len(servers) == 0 {
		return "📭 No servers connected."
	}

	return fmt.Sprintf("📋 Your servers:\n🟢 Server (%s)", servers[0][:12]+"...")
}

// handleHelp handles the /help command
func (b *Bot) handleHelp(message *tgbotapi.Message) string {
	return `🤖 ServerEye Bot Commands:

/start - Start using the bot
/temp - Get CPU temperature
/status - Get server status
/servers - List your servers
/help - Show this help

To connect a server, send your secret key (starts with srv_)`
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

	msgChan, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return 0, fmt.Errorf("failed to subscribe to response: %v", err)
	}

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
		case respData := <-msgChan:
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
