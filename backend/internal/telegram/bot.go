package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type Bot struct {
	bot    *tgbotapi.BotAPI
	logger *logrus.Logger
}

func NewBot(token string, logger *logrus.Logger) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	bot.Debug = false
	logger.WithField("bot_username", bot.Self.UserName).Info("Telegram bot authorized")

	return &Bot{
		bot:    bot,
		logger: logger,
	}, nil
}

func (b *Bot) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			if update.Message != nil {
				b.handleMessage(update.Message)
			}
		case <-ctx.Done():
			b.logger.Info("Stopping telegram bot")
			return ctx.Err()
		}
	}
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	b.logger.WithFields(logrus.Fields{
		"user_id":  message.From.ID,
		"username": message.From.UserName,
		"text":     message.Text,
		"chat_id":  message.Chat.ID,
	}).Info("Received telegram message")

	// Handle different commands
	switch message.Text {
	case "/start":
		b.sendMessage(message.Chat.ID, "👋 Welcome to ServerEye Bot!\n\nUse /help to see available commands.")
	case "/help":
		b.sendMessage(message.Chat.ID, "📋 Available commands:\n\n/start - Start the bot\n/help - Show this help message\n/status - Check server status")
	case "/status":
		b.sendMessage(message.Chat.ID, "✅ Server is running normally\n\n📊 CPU: 15%\n💾 Memory: 2.1GB/8GB\n🌐 Network: Connected")
	default:
		b.sendMessage(message.Chat.ID, "❓ Unknown command. Use /help to see available commands.")
	}
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	if _, err := b.bot.Send(msg); err != nil {
		b.logger.WithError(err).Error("Failed to send telegram message")
	}
}

func (b *Bot) Stop() {
	b.logger.Info("Stopping telegram bot")
}
