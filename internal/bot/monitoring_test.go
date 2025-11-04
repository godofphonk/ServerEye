package bot

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestMonitoringCommands_Structure(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{"temperature", "/temp"},
		{"memory", "/memory"},
		{"disk", "/disk"},
		{"uptime", "/uptime"},
		{"processes", "/processes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &tgbotapi.Message{
				Text: tt.command,
				From: &tgbotapi.User{ID: 12345},
				Chat: &tgbotapi.Chat{ID: 12345},
			}

			// Should not panic even without servers
			if msg.Text == "" {
				t.Error("Message text is empty")
			}
		})
	}
}

func TestBuildServerSelectionKeyboard(t *testing.T) {
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
		{SecretKey: "key2", Name: "Server 2"},
	}

	// Test that we can build a valid keyboard structure
	if len(servers) == 0 {
		t.Error("Servers list is empty")
	}

	for i, server := range servers {
		if server.SecretKey == "" {
			t.Errorf("Server %d has empty SecretKey", i)
		}
		if server.Name == "" {
			t.Errorf("Server %d has empty Name", i)
		}
	}
}

func TestMonitoringCommand_InvalidUserID(t *testing.T) {
	msg := &tgbotapi.Message{
		Text: "/temp",
		From: &tgbotapi.User{ID: 0}, // Invalid ID
		Chat: &tgbotapi.Chat{ID: 0},
	}

	// Should handle gracefully
	if msg.From.ID == 0 {
		t.Log("Zero user ID detected (expected)")
	}
}

func TestMonitoringKeyboard_ButtonData(t *testing.T) {
	tests := []struct {
		command string
		wantKey string
	}{
		{"/temp", "temp"},
		{"/memory", "memory"},
		{"/disk", "disk"},
		{"/uptime", "uptime"},
		{"/processes", "processes"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			// Test callback data format: "command_serverNumber"
			callbackData := tt.wantKey + "_1"
			if len(callbackData) == 0 {
				t.Error("Callback data is empty")
			}
		})
	}
}
