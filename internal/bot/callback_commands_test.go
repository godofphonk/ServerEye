package bot

import (
	"strings"
	"testing"
)

func TestExecuteTemperatureCommand_InvalidServer(t *testing.T) {
	bot := &Bot{}
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
	}

	result := bot.executeTemperatureCommand(servers, "invalid")
	
	if !strings.Contains(result, "Invalid server selection") {
		t.Errorf("Expected invalid server error, got: %v", result)
	}
}

func TestExecuteContainersCommand_InvalidServer(t *testing.T) {
	bot := &Bot{}
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
	}

	result := bot.executeContainersCommand(servers, "0")
	
	if !strings.Contains(result, "Invalid server selection") {
		t.Errorf("Expected invalid server error, got: %v", result)
	}
}

func TestExecuteMemoryCommand_InvalidServer(t *testing.T) {
	bot := &Bot{}
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
	}

	result := bot.executeMemoryCommand(servers, "999")
	
	if !strings.Contains(result, "Invalid server selection") {
		t.Errorf("Expected invalid server error, got: %v", result)
	}
}

func TestExecuteDiskCommand_InvalidServer(t *testing.T) {
	bot := &Bot{}
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
	}

	result := bot.executeDiskCommand(servers, "-1")
	
	if !strings.Contains(result, "Invalid server selection") {
		t.Errorf("Expected invalid server error, got: %v", result)
	}
}

func TestExecuteUptimeCommand_InvalidServer(t *testing.T) {
	bot := &Bot{}
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
	}

	result := bot.executeUptimeCommand(servers, "abc")
	
	if !strings.Contains(result, "Invalid server selection") {
		t.Errorf("Expected invalid server error, got: %v", result)
	}
}

func TestExecuteProcessesCommand_InvalidServer(t *testing.T) {
	bot := &Bot{}
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
	}

	result := bot.executeProcessesCommand(servers, "")
	
	if !strings.Contains(result, "Invalid server selection") {
		t.Errorf("Expected invalid server error, got: %v", result)
	}
}

func TestExecuteStatusCommand_InvalidServer(t *testing.T) {
	bot := &Bot{}
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
	}

	result := bot.executeStatusCommand(servers, "10")
	
	if !strings.Contains(result, "Invalid server selection") {
		t.Errorf("Expected invalid server error, got: %v", result)
	}
}

func TestExecuteStatusCommand_ValidServer(t *testing.T) {
	bot := &Bot{}
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "TestServer"},
	}

	result := bot.executeStatusCommand(servers, "1")
	
	if !strings.Contains(result, "TestServer") {
		t.Errorf("Expected TestServer in result, got: %v", result)
	}
	
	if !strings.Contains(result, "Status: Online") {
		t.Errorf("Expected status info, got: %v", result)
	}
}

func TestExecuteUpdateCommand_InvalidServer(t *testing.T) {
	bot := &Bot{}
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
	}

	result := bot.executeUpdateCommand(servers, "2", 123456)
	
	if !strings.Contains(result, "Invalid server selection") {
		t.Errorf("Expected invalid server error, got: %v", result)
	}
}
