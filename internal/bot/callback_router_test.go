package bot

import (
	"testing"
)

func TestSelectServer(t *testing.T) {
	servers := []ServerInfo{
		{SecretKey: "key1", Name: "Server 1"},
		{SecretKey: "key2", Name: "Server 2"},
		{SecretKey: "key3", Name: "Server 3"},
	}

	tests := []struct {
		name      string
		serverNum string
		wantKey   string
		wantName  string
		wantErr   bool
	}{
		{
			name:      "valid selection - first server",
			serverNum: "1",
			wantKey:   "key1",
			wantName:  "Server 1",
			wantErr:   false,
		},
		{
			name:      "valid selection - middle server",
			serverNum: "2",
			wantKey:   "key2",
			wantName:  "Server 2",
			wantErr:   false,
		},
		{
			name:      "valid selection - last server",
			serverNum: "3",
			wantKey:   "key3",
			wantName:  "Server 3",
			wantErr:   false,
		},
		{
			name:      "invalid - number too low",
			serverNum: "0",
			wantErr:   true,
		},
		{
			name:      "invalid - number too high",
			serverNum: "4",
			wantErr:   true,
		},
		{
			name:      "invalid - negative number",
			serverNum: "-1",
			wantErr:   true,
		},
		{
			name:      "invalid - not a number",
			serverNum: "abc",
			wantErr:   true,
		},
		{
			name:      "invalid - empty string",
			serverNum: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectServer(servers, tt.serverNum)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("selectServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Expected error, no need to check result
			}

			if got == nil {
				t.Fatal("selectServer() returned nil")
			}

			if got.Key != tt.wantKey {
				t.Errorf("Key = %v, want %v", got.Key, tt.wantKey)
			}

			if got.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", got.Name, tt.wantName)
			}
		})
	}
}

func TestSelectServer_EmptyServers(t *testing.T) {
	var emptyServers []ServerInfo

	_, err := selectServer(emptyServers, "1")
	if err == nil {
		t.Error("selectServer() with empty servers should return error")
	}
}

func TestServerSelection_Struct(t *testing.T) {
	sel := &serverSelection{
		Key:  "test-key",
		Name: "Test Server",
	}

	if sel.Key != "test-key" {
		t.Errorf("Key = %v, want test-key", sel.Key)
	}

	if sel.Name != "Test Server" {
		t.Errorf("Name = %v, want Test Server", sel.Name)
	}
}
