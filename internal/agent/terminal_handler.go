package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type TerminalMessage struct {
	Type      string `json:"type"`
	Data      string `json:"data,omitempty"`
	Cols      int    `json:"cols,omitempty"`
	Rows      int    `json:"rows,omitempty"`
	Message   string `json:"message,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	ServerID  string `json:"server_id,omitempty"`
}

type TerminalSession struct {
	SessionID string
	PTY       *os.File
	CMD       *exec.Cmd
	Cancel    context.CancelFunc
}

// startTerminalHandler запускает обработчик терминальных команд
func (a *Agent) startTerminalHandler() {
	a.logger.Info("Terminal handler started")

	sessions := make(map[string]*TerminalSession)
	var sessionsMu sync.RWMutex

	// Subscribe to terminal commands
	commandsChan := make(chan []byte, 100)
	go a.subscribeToTerminalCommands(commandsChan)

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Info("Terminal handler stopped")
			// Cleanup all sessions
			sessionsMu.Lock()
			for _, session := range sessions {
				session.Cancel()
				if session.PTY != nil {
					session.PTY.Close()
				}
			}
			sessionsMu.Unlock()
			return

		case cmdData := <-commandsChan:
			a.logger.WithField("data", string(cmdData)).Info("Processing terminal command")

			var msg TerminalMessage
			if err := json.Unmarshal(cmdData, &msg); err != nil {
				a.logger.WithError(err).Error("Failed to unmarshal terminal message")
				continue
			}

			a.logger.WithFields(logrus.Fields{
				"type":       msg.Type,
				"server_id":  msg.ServerID,
				"session_id": msg.SessionID,
			}).Info("Parsed terminal message")

			// ServerID in message is the actual server UUID from DB, not the secret key
			// We accept all messages since we're subscribed to our specific consumer group
			// No filtering needed - Kafka consumer group already filters by server

			switch msg.Type {
			case "init":
				// Initialize new terminal session
				go a.initTerminalSession(&msg, sessions, &sessionsMu)

			case "input":
				// Send input to existing session
				sessionsMu.RLock()
				session, exists := sessions[msg.SessionID]
				sessionsMu.RUnlock()

				a.logger.WithFields(logrus.Fields{
					"session_id": msg.SessionID,
					"exists":     exists,
					"data":       msg.Data,
				}).Info("Processing input for session")

				if !exists {
					a.logger.WithField("session_id", msg.SessionID).Warn("Session not found for input - ignoring old message")
					continue
				}

				if session.PTY == nil {
					a.logger.WithField("session_id", msg.SessionID).Error("Session exists but PTY is nil!")
					continue
				}

				n, err := session.PTY.Write([]byte(msg.Data))
				if err != nil {
					a.logger.WithError(err).Error("Failed to write to PTY")
				} else {
					a.logger.WithFields(logrus.Fields{
						"session_id": msg.SessionID,
						"data":       msg.Data,
						"bytes":      n,
					}).Info("Successfully wrote input to PTY")
				}

			case "resize":
				// Resize terminal
				sessionsMu.RLock()
				session, exists := sessions[msg.SessionID]
				sessionsMu.RUnlock()

				if exists && session.PTY != nil {
					ws := &pty.Winsize{
						Cols: uint16(msg.Cols),
						Rows: uint16(msg.Rows),
					}
					if err := pty.Setsize(session.PTY, ws); err != nil {
						a.logger.WithError(err).Error("Failed to resize PTY")
					}
				}
			}
		}
	}
}

func (a *Agent) initTerminalSession(msg *TerminalMessage, sessions map[string]*TerminalSession, mu *sync.RWMutex) {
	a.logger.WithFields(logrus.Fields{
		"session_id": msg.SessionID,
		"cols":       msg.Cols,
		"rows":       msg.Rows,
	}).Info("Initializing PTY terminal session")

	ctx, cancel := context.WithCancel(a.ctx)

	// Create PTY with interactive bash
	cmd := exec.CommandContext(ctx, "/bin/bash", "-i")
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"PS1=$ ",
	)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		a.logger.WithError(err).Error("Failed to start PTY")
		a.sendTerminalOutput(msg.SessionID, TerminalMessage{
			Type:      "error",
			Message:   "Failed to start terminal",
			SessionID: msg.SessionID,
		})
		cancel()
		return
	}

	a.logger.WithField("session_id", msg.SessionID).Info("PTY session created successfully")

	// Set initial size
	if msg.Cols > 0 && msg.Rows > 0 {
		ws := &pty.Winsize{
			Cols: uint16(msg.Cols),
			Rows: uint16(msg.Rows),
		}
		if err := pty.Setsize(ptmx, ws); err != nil {
			a.logger.WithError(err).Warn("Failed to set PTY size")
		}
	}

	session := &TerminalSession{
		SessionID: msg.SessionID,
		PTY:       ptmx,
		CMD:       cmd,
		Cancel:    cancel,
	}

	mu.Lock()
	sessions[msg.SessionID] = session
	mu.Unlock()

	a.logger.WithField("session_id", msg.SessionID).Info("Session registered, starting output reader")

	// Send welcome message immediately to test output path
	a.sendTerminalOutput(msg.SessionID, TerminalMessage{
		Type:      "output",
		Data:      "\r\nTerminal ready. Type your commands:\r\n",
		SessionID: msg.SessionID,
	})

	// Read output and send to Kafka
	go func() {
		defer func() {
			mu.Lock()
			delete(sessions, msg.SessionID)
			mu.Unlock()
			cancel()
			ptmx.Close()
			a.logger.WithField("session_id", msg.SessionID).Info("PTY reader goroutine exiting")
		}()

		reader := bufio.NewReader(ptmx)
		buf := make([]byte, 1024)

		a.logger.WithField("session_id", msg.SessionID).Info("PTY reader started, waiting for output...")

		for {
			n, err := reader.Read(buf)
			if err != nil {
				if err != io.EOF {
					a.logger.WithError(err).Debug("PTY read error")
				}
				break
			}

			if n > 0 {
				output := string(buf[:n])
				a.logger.WithFields(logrus.Fields{
					"session_id": msg.SessionID,
					"bytes":      n,
					"output":     output,
				}).Debug("Sending PTY output to Kafka")

				a.sendTerminalOutput(msg.SessionID, TerminalMessage{
					Type:      "output",
					Data:      output,
					SessionID: msg.SessionID,
				})
			}
		}

		// Session ended
		a.sendTerminalOutput(msg.SessionID, TerminalMessage{
			Type:      "output",
			Data:      "\r\n[Session closed]\r\n",
			SessionID: msg.SessionID,
		})
	}()
}

func (a *Agent) subscribeToTerminalCommands(ch chan<- []byte) {
	if !a.config.Kafka.Enabled {
		a.logger.Warn("Kafka not enabled, terminal commands not available")
		return
	}

	a.logger.Info("Subscribed to terminal commands")

	// Create Kafka reader for terminal commands
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        a.config.Kafka.Brokers,
		Topic:          "terminal.commands",
		GroupID:        fmt.Sprintf("agent-%s", a.config.Server.SecretKey),
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})
	defer reader.Close()

	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			msg, err := reader.FetchMessage(a.ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				a.logger.WithError(err).Debug("Failed to fetch terminal command (waiting)")
				time.Sleep(time.Second)
				continue
			}

			a.logger.WithField("message", string(msg.Value)).Info("Received terminal command from Kafka")

			// Send to channel
			ch <- msg.Value

			// Commit message
			if err := reader.CommitMessages(a.ctx, msg); err != nil {
				a.logger.WithError(err).Warn("Failed to commit terminal message")
			}
		}
	}
}

func (a *Agent) sendTerminalOutput(sessionID string, msg TerminalMessage) {
	a.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"type":       msg.Type,
		"data_len":   len(msg.Data),
	}).Info("sendTerminalOutput called")

	if !a.config.Kafka.Enabled {
		a.logger.Warn("Kafka not enabled, cannot send terminal output")
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		a.logger.WithError(err).Error("Failed to marshal terminal output")
		return
	}

	a.logger.WithField("json", string(data)).Info("Sending to Kafka terminal.output")

	// Send directly to Kafka topic "terminal.output"
	writer := &kafka.Writer{
		Addr:     kafka.TCP(a.config.Kafka.Brokers...),
		Topic:    "terminal.output",
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	err = writer.WriteMessages(a.ctx, kafka.Message{
		Key:   []byte(sessionID),
		Value: data,
	})

	if err != nil {
		a.logger.WithError(err).Error("Failed to send terminal output to Kafka")
	} else {
		a.logger.Info("Successfully sent terminal output to Kafka!")
	}
}
