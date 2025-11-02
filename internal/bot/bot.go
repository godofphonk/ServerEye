package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/redis"
	"github.com/servereye/servereye/pkg/redis/streams"
	"github.com/sirupsen/logrus"
)

// Bot represents the Telegram bot instance with dependency injection
type Bot struct {
	// Configuration
	config *config.BotConfig

	// Dependencies (interfaces for better testability)
	logger      Logger
	telegramAPI TelegramAPI
	redisClient RedisClient
	database    Database
	agentClient AgentClient
	validator   Validator
	metrics     Metrics

	// Direct database access for internal methods
	db *sql.DB

	// Concrete Redis client for Streams (not in interface yet)
	redisRawClient interface{}

	// Streams client for new architecture
	streamsClient interface{}

	// Context management
	ctx    context.Context
	cancel context.CancelFunc

	// Graceful shutdown
	wg       sync.WaitGroup
	shutdown chan struct{}
}

// BotOptions contains options for creating a new bot instance
type BotOptions struct {
	Config      *config.BotConfig
	Logger      Logger
	TelegramAPI TelegramAPI
	RedisClient RedisClient
	Database    Database
	AgentClient AgentClient
	Validator   Validator
	Metrics     Metrics
}

// New creates a new bot instance with dependency injection
func New(opts BotOptions) (*Bot, error) {
	if opts.Config == nil {
		return nil, NewValidationError("config is required", nil)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bot := &Bot{
		config:      opts.Config,
		logger:      opts.Logger,
		telegramAPI: opts.TelegramAPI,
		redisClient: opts.RedisClient,
		database:    opts.Database,
		agentClient: opts.AgentClient,
		validator:   opts.Validator,
		metrics:     opts.Metrics,
		ctx:         ctx,
		cancel:      cancel,
		shutdown:    make(chan struct{}),
	}

	// Set defaults if not provided
	if bot.logger == nil {
		logrusLogger := logrus.New()
		logrusLogger.SetLevel(logrus.InfoLevel)
		bot.logger = NewStructuredLogger(logrusLogger)
	}

	if bot.validator == nil {
		bot.validator = NewInputValidator()
	}

	if bot.metrics == nil {
		bot.metrics = NewInMemoryMetrics()
	}

	return bot, nil
}

// NewFromConfig creates a bot instance from configuration (legacy constructor)
func NewFromConfig(cfg *config.BotConfig, logger *logrus.Logger) (*Bot, error) {
	// Initialize Telegram bot
	tgBot, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return nil, NewTelegramError("failed to create Telegram bot", err)
	}

	logger.WithField("username", tgBot.Self.UserName).Info("Telegram bot authorized")

	// Initialize Redis client
	redisClient, err := redis.NewClient(redis.Config{
		Address:  cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}, logger)
	if err != nil {
		return nil, NewRedisError("failed to create Redis client", err)
	}

	// Initialize database connection
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return nil, NewDatabaseError("failed to connect to database", err)
	}

	if err := db.Ping(); err != nil {
		return nil, NewDatabaseError("failed to ping database", err)
	}

	logger.Info("Database connection established")

	// Create a temporary bot instance for adapters
	tempBot := &Bot{}

	// Create adapters
	dbAdapter := NewDatabaseAdapter(db, tempBot)
	redisAdapter := NewRedisAdapter(redisClient)
	agentAdapter := NewAgentClientAdapter(tempBot)

	// Create bot with real implementations
	bot, err := New(BotOptions{
		Config:      cfg,
		Logger:      NewStructuredLogger(logger),
		TelegramAPI: tgBot,
		RedisClient: redisAdapter,
		Database:    dbAdapter,
		AgentClient: agentAdapter,
		Validator:   NewInputValidator(),
		Metrics:     NewInMemoryMetrics(),
	})

	if err != nil {
		return nil, err
	}

	// Update adapter references and set direct DB access
	dbAdapter.bot = bot
	agentAdapter.bot = bot
	bot.db = db
	bot.redisRawClient = redisClient // Store raw client for Streams

	// Initialize Streams client
	streamsConfig := &streams.Config{
		Addr:            cfg.Redis.Address,
		Password:        cfg.Redis.Password,
		DB:              cfg.Redis.DB,
		MaxRetries:      3,
		BlockDuration:   5 * time.Second,
		BatchSize:       10,
		StreamMaxLength: 1000,
	}

	streamsClient, err := streams.NewClient(streamsConfig, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to create Streams client, will use Pub/Sub")
	} else {
		bot.streamsClient = streamsClient
		logger.Info("Redis Streams client initialized")
	}

	return bot, nil
}

// Start starts the bot with graceful shutdown handling
func (b *Bot) Start() error {
	b.logger.Info("Starting ServerEye Telegram bot")

	// Initialize database schema if database is available
	if b.database != nil {
		if err := b.database.InitSchema(); err != nil {
			return NewDatabaseError("failed to initialize database schema", err)
		}
	}

	// Setup graceful shutdown
	b.setupGracefulShutdown()

	// Start HTTP server for agent API
	go func() {
		b.logger.Info("About to start HTTP server goroutine...")
		b.startHTTPServer()
	}()

	// Start Telegram updates handler
	if err := b.startTelegramHandler(); err != nil {
		return NewTelegramError("failed to start Telegram handler", err)
	}

	b.logger.Info("ServerEye Telegram bot started successfully")

	// Wait for shutdown signal
	<-b.shutdown

	return nil
}

// startTelegramHandler starts the Telegram updates handler
func (b *Bot) startTelegramHandler() error {
	if b.telegramAPI == nil {
		return NewTelegramError("Telegram API not initialized", nil)
	}

	// Configure updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.telegramAPI.GetUpdatesChan(u)

	// Start handling updates in a separate goroutine
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.handleUpdates(updates)
	}()

	b.logger.Info("Telegram updates handler started")
	return nil
}

// setupGracefulShutdown sets up graceful shutdown handling
func (b *Bot) setupGracefulShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		b.logger.Info("Received shutdown signal", StringField("signal", sig.String()))
		b.Stop()
	}()
}

// Stop gracefully stops the bot
func (b *Bot) Stop() error {
	b.logger.Info("Initiating graceful shutdown")

	// Cancel context to stop all operations
	b.cancel()

	// Stop receiving Telegram updates
	if b.telegramAPI != nil {
		b.telegramAPI.StopReceivingUpdates()
	}

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		b.logger.Info("All goroutines stopped gracefully")
	case <-time.After(30 * time.Second):
		b.logger.Warn("Timeout waiting for goroutines to stop")
	}

	// Close connections
	if b.redisClient != nil {
		if err := b.redisClient.Close(); err != nil {
			b.logger.Error("Error closing Redis connection", err)
		}
	}

	if b.database != nil {
		if err := b.database.Close(); err != nil {
			b.logger.Error("Error closing database connection", err)
		}
	}

	// Signal shutdown complete
	close(b.shutdown)

	b.logger.Info("Bot stopped successfully")
	return nil
}

// handleUpdates processes incoming Telegram updates with error handling and metrics
func (b *Bot) handleUpdates(updates tgbotapi.UpdatesChannel) {
	b.logger.Info("Starting Telegram updates processing")

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				b.logger.Info("Updates channel closed, stopping handler")
				return
			}

			// Process update with timeout and error handling
			ctx, cancel := context.WithTimeout(b.ctx, 30*time.Second)
			b.processUpdate(ctx, update)
			cancel()

		case <-b.ctx.Done():
			b.logger.Info("Context cancelled, stopping updates handler")
			return
		}
	}
}

// processUpdate processes a single update with error recovery
func (b *Bot) processUpdate(ctx context.Context, update tgbotapi.Update) {
	// Recover from panics to prevent bot crash
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("Panic recovered in update processing",
				fmt.Errorf("panic: %v", r),
				StringField("update_id", fmt.Sprintf("%d", update.UpdateID)))

			if b.metrics != nil {
				b.metrics.IncrementError("PANIC_RECOVERED")
			}
		}
	}()

	// Process different update types
	switch {
	case update.Message != nil:
		b.processMessage(ctx, update.Message)
	case update.CallbackQuery != nil:
		b.processCallbackQuery(ctx, update.CallbackQuery)
	default:
		b.logger.Debug("Received update without message or callback query",
			IntField("update_id", update.UpdateID))
	}
}

// processMessage processes a message update
func (b *Bot) processMessage(ctx context.Context, message *tgbotapi.Message) {
	start := time.Now()

	// Log message details
	b.logger.Info("Processing message",
		Int64Field("user_id", message.From.ID),
		StringField("username", message.From.UserName),
		StringField("text", message.Text),
		IntField("message_id", message.MessageID))

	// Validate and sanitize input
	if b.validator != nil && message.Text != "" {
		if validator, ok := b.validator.(*InputValidator); ok {
			message.Text = validator.SanitizeInput(message.Text)
		}
	}

	// Handle message with error handling
	err := b.handleMessage(message)

	// Record metrics
	if b.metrics != nil {
		duration := time.Since(start).Seconds()
		b.metrics.RecordLatency("message_processing", duration)

		if err != nil {
			var botErr *BotError
			if errors.As(err, &botErr) {
				b.metrics.IncrementError(botErr.Code)
			} else {
				b.metrics.IncrementError("UNKNOWN_ERROR")
			}
		}
	}

	if err != nil {
		b.logger.Error("Error processing message", err,
			Int64Field("user_id", message.From.ID),
			StringField("text", message.Text))
	}
}

// processCallbackQuery processes a callback query update
func (b *Bot) processCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery) {
	start := time.Now()

	// Log callback details
	b.logger.Info("Processing callback query",
		Int64Field("user_id", query.From.ID),
		StringField("username", query.From.UserName),
		StringField("data", query.Data),
		StringField("query_id", query.ID))

	// Handle callback with error handling
	err := b.handleCallbackQuery(query)

	// Record metrics
	if b.metrics != nil {
		duration := time.Since(start).Seconds()
		b.metrics.RecordLatency("callback_processing", duration)

		if err != nil {
			var botErr *BotError
			if errors.As(err, &botErr) {
				b.metrics.IncrementError(botErr.Code)
			} else {
				b.metrics.IncrementError("UNKNOWN_ERROR")
			}
		}
	}

	if err != nil {
		b.logger.Error("Error processing callback query", err,
			Int64Field("user_id", query.From.ID),
			StringField("data", query.Data))
	}
}

// handleMessage processes a single message
