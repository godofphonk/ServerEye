package bot

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// serverSelection holds selected server info
type serverSelection struct {
	Key  string
	Name string
}

// selectServer validates and returns server selection
func selectServer(servers []ServerInfo, serverNum string) (*serverSelection, error) {
	num, err := strconv.Atoi(serverNum)
	if err != nil || num < 1 || num > len(servers) {
		return nil, fmt.Errorf("invalid server selection")
	}

	return &serverSelection{
		Key:  servers[num-1].SecretKey,
		Name: servers[num-1].Name,
	}, nil
}

// handleCallbackQuery processes callback queries from inline keyboards
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) error {
	// Answer the callback query
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := b.telegramAPI.Request(callback); err != nil {
		b.logger.Error("Error occurred", err)
	}

	// Check for cancel action
	if query.Data == "container_cancel" {
		editMsg := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID,
			query.Message.MessageID,
			"❌ Action cancelled",
		)
		if _, err := b.telegramAPI.Send(editMsg); err != nil {
			b.logger.Error("Error occurred", err)
		}
		return nil
	}

	// Check if it's a create template selection
	if strings.HasPrefix(query.Data, "create_template_") {
		return b.handleTemplateSelection(query)
	}

	// Check if it's a container action selection
	if strings.HasPrefix(query.Data, "container_action_") {
		return b.handleContainerActionSelection(query)
	}

	// Check if it's a container action callback
	if strings.HasPrefix(query.Data, "container_") {
		return b.handleContainerActionCallback(query)
	}

	// Parse callback data (format: "command_serverNumber")
	parts := strings.Split(query.Data, "_")
	if len(parts) != 2 {
		b.logger.Error("Operation failed", nil)
		return fmt.Errorf("invalid callback data format: %s", query.Data)
	}

	command := parts[0]
	serverNum := parts[1]

	// Get user's servers
	servers, err := b.getUserServersWithInfo(query.From.ID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return err
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
	case "update":
		response = b.executeUpdateCommand(servers, serverNum, query.Message.Chat.ID)
	default:
		response = "❌ Unknown command"
	}

	// Send response
	b.sendMessage(query.Message.Chat.ID, response)
	return nil
}
