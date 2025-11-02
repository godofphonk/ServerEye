package streams

import (
	"context"
	"fmt"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// BotAdapter provides high-level API for bot to send commands and receive responses
type BotAdapter struct {
	client StreamClient
	logger *logrus.Logger
}

// NewBotAdapter creates a new bot adapter
func NewBotAdapter(client StreamClient, logger *logrus.Logger) *BotAdapter {
	return &BotAdapter{
		client: client,
		logger: logger,
	}
}

// SendCommand sends a command to an agent and waits for response
func (a *BotAdapter) SendCommand(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	// Stream names
	cmdStream := fmt.Sprintf("stream:cmd:%s", serverKey)
	respStream := fmt.Sprintf("stream:resp:%s", serverKey)

	// Serialize command
	commandJSON, err := command.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %w", err)
	}

	// Add command to stream
	values := map[string]string{
		"type":       string(command.Type),
		"id":         command.ID,
		"payload":    string(commandJSON),
		"timestamp":  time.Now().Format(time.RFC3339),
		"server_key": serverKey,
	}

	_, err = a.client.AddMessage(ctx, cmdStream, values)
	if err != nil {
		return nil, fmt.Errorf("failed to add command to stream: %w", err)
	}

	a.logger.WithFields(logrus.Fields{
		"command_id":   command.ID,
		"command_type": command.Type,
		"stream":       cmdStream,
	}).Info("Command sent to stream")

	// Wait for response
	return a.waitForResponse(ctx, respStream, command.ID, timeout)
}

// waitForResponse waits for a specific response by command ID
func (a *BotAdapter) waitForResponse(ctx context.Context, respStream, commandID string, timeout time.Duration) (*protocol.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	lastID := "0" // Start from beginning
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for response")

		case <-ticker.C:
			// Read messages from response stream
			messages, err := a.client.ReadMessages(ctx, respStream, lastID, 10, 500*time.Millisecond)
			if err != nil {
				a.logger.WithError(err).Error("Failed to read response stream")
				continue
			}

			for _, msg := range messages {
				lastID = msg.ID

				// Check if this response is for our command
				if msg.Values["command_id"] != commandID {
					continue
				}

				// Parse response
				payloadJSON := msg.Values["payload"]
				response, err := protocol.FromJSON([]byte(payloadJSON))
				if err != nil {
					a.logger.WithError(err).Error("Failed to parse response")
					continue
				}

				a.logger.WithFields(logrus.Fields{
					"command_id":    commandID,
					"response_type": response.Type,
					"message_id":    msg.ID,
				}).Info("Response received from stream")

				return response, nil
			}
		}
	}
}

// GetStreamStats returns statistics about command/response streams
func (a *BotAdapter) GetStreamStats(ctx context.Context, serverKey string) (map[string]int64, error) {
	cmdStream := fmt.Sprintf("stream:cmd:%s", serverKey)
	respStream := fmt.Sprintf("stream:resp:%s", serverKey)

	cmdLen, err := a.client.GetStreamLength(ctx, cmdStream)
	if err != nil {
		return nil, err
	}

	respLen, err := a.client.GetStreamLength(ctx, respStream)
	if err != nil {
		return nil, err
	}

	return map[string]int64{
		"commands":  cmdLen,
		"responses": respLen,
	}, nil
}
