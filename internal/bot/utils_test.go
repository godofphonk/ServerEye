package bot

import (
	"testing"
)

func TestGetServerFromCommand(t *testing.T) {
	// Create a minimal bot instance for testing
	bot := &Bot{}

	tests := []struct {
		name       string
		command    string
		servers    []string
		wantServer string
		wantErr    bool
	}{
		{
			name:       "single server - no number",
			command:    "/temp",
			servers:    []string{"srv_123"},
			wantServer: "srv_123",
			wantErr:    false,
		},
		{
			name:       "multiple servers - no number",
			command:    "/temp",
			servers:    []string{"srv_123", "srv_456"},
			wantServer: "",
			wantErr:    true,
		},
		{
			name:       "select first server",
			command:    "/temp 1",
			servers:    []string{"srv_123", "srv_456"},
			wantServer: "srv_123",
			wantErr:    false,
		},
		{
			name:       "select second server",
			command:    "/temp 2",
			servers:    []string{"srv_123", "srv_456"},
			wantServer: "srv_456",
			wantErr:    false,
		},
		{
			name:       "invalid server number - non-numeric",
			command:    "/temp abc",
			servers:    []string{"srv_123", "srv_456"},
			wantServer: "",
			wantErr:    true,
		},
		{
			name:       "invalid server number - zero",
			command:    "/temp 0",
			servers:    []string{"srv_123", "srv_456"},
			wantServer: "",
			wantErr:    true,
		},
		{
			name:       "invalid server number - negative",
			command:    "/temp -1",
			servers:    []string{"srv_123", "srv_456"},
			wantServer: "",
			wantErr:    true,
		},
		{
			name:       "invalid server number - too high",
			command:    "/temp 5",
			servers:    []string{"srv_123", "srv_456"},
			wantServer: "",
			wantErr:    true,
		},
		{
			name:       "command with extra spaces",
			command:    "/temp   1  ",
			servers:    []string{"srv_123", "srv_456"},
			wantServer: "srv_123",
			wantErr:    false,
		},
		{
			name:       "empty servers list",
			command:    "/temp",
			servers:    []string{},
			wantServer: "",
			wantErr:    true, // Should return error for empty servers list
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotServer, err := bot.getServerFromCommand(tt.command, tt.servers)

			if (err != nil) != tt.wantErr {
				t.Errorf("getServerFromCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotServer != tt.wantServer {
				t.Errorf("getServerFromCommand() = %v, want %v", gotServer, tt.wantServer)
			}
		})
	}
}

func TestGetServerFromCommand_EdgeCases(t *testing.T) {
	bot := &Bot{}

	t.Run("three servers - select middle", func(t *testing.T) {
		servers := []string{"srv_111", "srv_222", "srv_333"}
		got, err := bot.getServerFromCommand("/temp 2", servers)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if got != "srv_222" {
			t.Errorf("got %v, want srv_222", got)
		}
	})

	t.Run("boundary - exactly max servers", func(t *testing.T) {
		servers := []string{"srv_1", "srv_2", "srv_3"}
		got, err := bot.getServerFromCommand("/temp 3", servers)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if got != "srv_3" {
			t.Errorf("got %v, want srv_3", got)
		}
	})

	t.Run("boundary - one past max", func(t *testing.T) {
		servers := []string{"srv_1", "srv_2", "srv_3"}
		_, err := bot.getServerFromCommand("/temp 4", servers)

		if err == nil {
			t.Error("expected error for server number out of range")
		}
	})
}
