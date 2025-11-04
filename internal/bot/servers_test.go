package bot

import (
	"testing"
)

func TestServerInfo_Struct(t *testing.T) {
	server := ServerInfo{
		SecretKey: "test-key-123",
		Name:      "Production Server",
	}

	if server.SecretKey != "test-key-123" {
		t.Errorf("SecretKey = %v, want test-key-123", server.SecretKey)
	}

	if server.Name != "Production Server" {
		t.Errorf("Name = %v, want Production Server", server.Name)
	}
}

func TestServerInfo_Empty(t *testing.T) {
	var server ServerInfo

	if server.SecretKey != "" {
		t.Errorf("Empty ServerInfo should have empty SecretKey")
	}

	if server.Name != "" {
		t.Errorf("Empty ServerInfo should have empty Name")
	}
}

func TestServersList(t *testing.T) {
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
		{SecretKey: "key2", Name: "Server 2"},
		{SecretKey: "key3", Name: "Server 3"},
	}

	if len(servers) != 3 {
		t.Errorf("Expected 3 servers, got %d", len(servers))
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

func TestServerInfo_UniqueKeys(t *testing.T) {
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
		{SecretKey: "key2", Name: "Server 2"},
		{SecretKey: "key1", Name: "Server 3"}, // Duplicate key
	}

	keys := make(map[string]bool)
	duplicates := 0

	for _, server := range servers {
		if keys[server.SecretKey] {
			duplicates++
		}
		keys[server.SecretKey] = true
	}

	if duplicates == 0 {
		t.Log("No duplicate keys detected (expected in test)")
	}
}
