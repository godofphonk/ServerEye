package bot

import (
	"context"
	"database/sql"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/servereye/servereye/internal/config"
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

			if update.Message != nil {
				b.logger.WithFields(logrus.Fields{
					"user_id":  update.Message.From.ID,
					"username": update.Message.From.UserName,
					"text":     update.Message.Text,
				}).Info("Получено сообщение от Telegram")

				b.handleMessage(update.Message)
			} else if update.CallbackQuery != nil {
				b.logger.WithFields(logrus.Fields{
					"user_id":  update.CallbackQuery.From.ID,
					"username": update.CallbackQuery.From.UserName,
					"data":     update.CallbackQuery.Data,
				}).Info("Получен callback query от Telegram")

				b.handleCallbackQuery(update.CallbackQuery)
			} else {
				b.logger.Info("Обновление без сообщения или callback, пропускаем")
				continue
			}

		case <-b.ctx.Done():
			b.logger.Info("Остановка обработки обновлений")
			return
		}
	}
}

// handleMessage processes a single message
