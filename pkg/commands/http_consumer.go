package commands

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/godofphonk/ServerEye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

type HTTPCommandConsumer struct {
	apiURL       string
	apiKey       string
	serverKey    string
	pollInterval time.Duration
	client       *http.Client
	handler      CommandHandler
	logger       *logrus.Logger
}

type CommandHandler interface {
	HandleCommand(ctx context.Context, cmd *protocol.Message) (*protocol.Message, error)
}

type Command struct {
	ID        string                 `json:"id"`
	Command   string                 `json:"command"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type CommandResponse struct {
	RequestID string      `json:"request_id"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
}

type HTTPConsumerConfig struct {
	APIURL       string
	APIKey       string
	ServerKey    string
	PollInterval time.Duration
}

func NewHTTPCommandConsumer(cfg HTTPConsumerConfig, handler CommandHandler, logger *logrus.Logger) *HTTPCommandConsumer {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 5 * time.Second
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout:   15 * time.Second,
				KeepAlive: 60 * time.Second,
			}
			return dialer.DialContext(ctx, "tcp4", addr)
		},
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   5,
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &HTTPCommandConsumer{
		apiURL:       cfg.APIURL,
		apiKey:       cfg.APIKey,
		serverKey:    cfg.ServerKey,
		pollInterval: cfg.PollInterval,
		client: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
		handler: handler,
		logger:  logger,
	}
}

func (c *HTTPCommandConsumer) Start(ctx context.Context) error {
	c.logger.WithFields(logrus.Fields{
		"api_url":       c.apiURL,
		"server_key":    c.serverKey,
		"poll_interval": c.pollInterval,
	}).Info("Starting HTTP command consumer")

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	var consecutiveErrors int
	const maxErrors = 5

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("HTTP command consumer stopped")
			return nil
		case <-ticker.C:
			if err := c.pollCommands(ctx); err != nil {
				consecutiveErrors++
				c.logger.WithError(err).WithField("consecutive_errors", consecutiveErrors).Info("Failed to poll commands")

				if consecutiveErrors >= maxErrors {
					backoff := time.Duration(consecutiveErrors) * c.pollInterval
					if backoff > 30*time.Second {
						backoff = 30 * time.Second
					}
					c.logger.WithField("backoff", backoff).Warn("Too many errors, backing off")
					time.Sleep(backoff)
				}
			} else {
				consecutiveErrors = 0
			}
		}
	}
}

func (c *HTTPCommandConsumer) pollCommands(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1/commands/%s", c.apiURL, c.serverKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Commands []Command `json:"commands"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Commands) == 0 {
		return nil
	}

	for _, cmd := range result.Commands {
		go c.processCommand(ctx, cmd)
	}

	return nil
}

func (c *HTTPCommandConsumer) processCommand(ctx context.Context, cmd Command) {
	msgType := c.mapCommandToType(cmd.Command)

	protoMsg := &protocol.Message{
		ID:        cmd.ID,
		Type:      msgType,
		Timestamp: cmd.Timestamp,
		ServerKey: c.serverKey,
		Payload:   cmd.Params,
	}

	protoResp, err := c.handler.HandleCommand(ctx, protoMsg)

	var response *CommandResponse
	if err != nil {
		c.logger.WithError(err).WithField("command_id", cmd.ID).Error("Failed to handle command")
		response = &CommandResponse{
			RequestID: cmd.ID,
			Success:   false,
			Error:     err.Error(),
		}
	} else if protoResp != nil {
		response = &CommandResponse{
			RequestID: cmd.ID,
			Success:   protoResp.Type != protocol.TypeErrorResponse,
			Data:      protoResp.Payload,
		}
		if protoResp.Type == protocol.TypeErrorResponse {
			if errPayload, ok := protoResp.Payload.(map[string]string); ok {
				response.Error = errPayload["error"]
			}
		}
	}

	if response != nil {
		if err := c.sendResponse(ctx, response); err != nil {
			c.logger.WithError(err).Error("Failed to send command response")
		}
	}
}

func (c *HTTPCommandConsumer) mapCommandToType(cmd string) protocol.MessageType {
	switch cmd {
	case "get_temperature", "temp":
		return protocol.TypeGetCPUTemp
	case "get_memory", "memory":
		return protocol.TypeGetMemoryInfo
	case "get_disk", "disk":
		return protocol.TypeGetDiskInfo
	case "get_containers", "containers":
		return protocol.TypeGetContainers
	case "get_status", "status":
		return protocol.TypeGetSystemInfo
	case "get_uptime", "uptime":
		return protocol.TypeGetUptime
	case "get_processes", "processes":
		return protocol.TypeGetProcesses
	default:
		return protocol.MessageType(cmd)
	}
}

func (c *HTTPCommandConsumer) sendResponse(ctx context.Context, response *CommandResponse) error {
	url := fmt.Sprintf("%s/v1/commands/%s/response", c.apiURL, c.serverKey)

	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.WithField("request_id", response.RequestID).Info("Command response sent successfully")
	return nil
}

func (c *HTTPCommandConsumer) Close() error {
	c.logger.Info("Closing HTTP command consumer")
	return nil
}
