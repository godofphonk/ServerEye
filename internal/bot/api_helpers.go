package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
)

// sendCommandAndParse sends a command and parses the response into the provided type
// This eliminates code duplication across all API methods
func sendCommandAndParse[T any](
	b *Bot,
	serverKey string,
	commandType protocol.MessageType,
	payload interface{},
	expectedResponseType protocol.MessageType,
	timeout time.Duration,
) (*T, error) {
	cmd := protocol.NewMessage(commandType, payload)

	ctx, cancel := context.WithTimeout(b.ctx, timeout)
	defer cancel()

	resp, err := b.sendCommandViaStreams(ctx, serverKey, cmd, timeout)
	if err != nil {
		return nil, err
	}

	if resp.Type == protocol.TypeErrorResponse {
		return nil, fmt.Errorf("agent error: %v", resp.Payload)
	}

	if resp.Type != expectedResponseType {
		return nil, fmt.Errorf("unexpected response type: expected %s, got %s", expectedResponseType, resp.Type)
	}

	// Parse payload
	payloadMap, ok := resp.Payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload format")
	}

	// Marshal and unmarshal for type conversion
	data, err := json.Marshal(payloadMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}
