package streams

import (
	"context"
	"fmt"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// CommandHandler is a function that processes a command and returns a response
type CommandHandler func(ctx context.Context, command *protocol.Message) *protocol.Message

// AgentAdapter provides high-level API for agent to consume commands and send responses
type AgentAdapter struct {
	client        StreamClient
	logger        *logrus.Logger
	serverKey     string
	consumerGroup string
	consumerName  string
}

// NewAgentAdapter creates a new agent adapter
func NewAgentAdapter(client StreamClient, serverKey, consumerGroup, consumerName string, logger *logrus.Logger) *AgentAdapter {
	return &AgentAdapter{
		client:        client,
		logger:        logger,
		serverKey:     serverKey,
		consumerGroup: consumerGroup,
		consumerName:  consumerName,
	}
}

// Initialize sets up consumer group for the agent
func (a *AgentAdapter) Initialize(ctx context.Context) error {
	cmdStream := fmt.Sprintf("stream:cmd:%s", a.serverKey)

	// Create consumer group
	err := a.client.CreateConsumerGroup(ctx, cmdStream, a.consumerGroup)
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	a.logger.WithFields(logrus.Fields{
		"stream":   cmdStream,
		"group":    a.consumerGroup,
		"consumer": a.consumerName,
	}).Info("Agent initialized with consumer group")

	return nil
}

// ProcessCommands starts processing commands from the stream
func (a *AgentAdapter) ProcessCommands(ctx context.Context, handler CommandHandler) error {
	cmdStream := fmt.Sprintf("stream:cmd:%s", a.serverKey)
	respStream := fmt.Sprintf("stream:resp:%s", a.serverKey)

	a.logger.WithField("stream", cmdStream).Info("Starting command processing")

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Command processing stopped")
			return ctx.Err()

		default:
			// Read messages from consumer group
			messages, err := a.client.ReadGroupMessages(
				ctx,
				cmdStream,
				a.consumerGroup,
				a.consumerName,
				10,                // batch size
				5*time.Second,     // block duration
			)
			if err != nil {
				a.logger.WithError(err).Error("Failed to read messages")
				time.Sleep(1 * time.Second)
				continue
			}

			// Process each message
			for _, msg := range messages {
				if err := a.processMessage(ctx, msg, handler, respStream); err != nil {
					a.logger.WithError(err).Error("Failed to process message")
					// Don't ACK - message will be redelivered
					continue
				}

				// ACK message after successful processing
				if err := a.client.AckMessage(ctx, cmdStream, a.consumerGroup, msg.ID); err != nil {
					a.logger.WithError(err).Error("Failed to ACK message")
				}
			}
		}
	}
}

// processMessage processes a single command message
func (a *AgentAdapter) processMessage(ctx context.Context, msg StreamMessage, handler CommandHandler, respStream string) error {
	start := time.Now()

	// Parse command
	payloadJSON := msg.Values["payload"]
	command, err := protocol.FromJSON([]byte(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to parse command: %w", err)
	}

	a.logger.WithFields(logrus.Fields{
		"command_id":   command.ID,
		"command_type": command.Type,
		"message_id":   msg.ID,
	}).Info("Processing command")

	// Execute handler
	response := handler(ctx, command)
	if response == nil {
		return fmt.Errorf("handler returned nil response")
	}

	// Send response
	if err := a.SendResponse(ctx, response, command.ID); err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}

	a.logger.WithFields(logrus.Fields{
		"command_id":    command.ID,
		"response_type": response.Type,
		"duration":      time.Since(start),
	}).Info("Command processed successfully")

	return nil
}

// SendResponse sends a response to the response stream
func (a *AgentAdapter) SendResponse(ctx context.Context, response *protocol.Message, commandID string) error {
	respStream := fmt.Sprintf("stream:resp:%s", a.serverKey)

	responseJSON, err := response.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize response: %w", err)
	}

	values := map[string]string{
		"type":       string(response.Type),
		"id":         response.ID,
		"command_id": commandID, // Link response to original command
		"payload":    string(responseJSON),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	_, err = a.client.AddMessage(ctx, respStream, values)
	if err != nil {
		return fmt.Errorf("failed to add response to stream: %w", err)
	}

	a.logger.WithFields(logrus.Fields{
		"command_id":    commandID,
		"response_type": response.Type,
		"stream":        respStream,
	}).Debug("Response sent to stream")

	return nil
}

// GetPendingMessages returns messages that were delivered but not ACKed
func (a *AgentAdapter) GetPendingMessages(ctx context.Context) ([]StreamMessage, error) {
	cmdStream := fmt.Sprintf("stream:cmd:%s", a.serverKey)

	// XPENDING shows pending messages in consumer group
	pending, err := a.client.(*Client).client.XPending(ctx, cmdStream, a.consumerGroup).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get pending messages: %w", err)
	}

	a.logger.WithFields(logrus.Fields{
		"count": pending.Count,
	}).Debug("Pending messages info")

	return nil, nil // Simplified for now
}
