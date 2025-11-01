package bot

import (
	"testing"
)

func TestValidateServerKey(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid key",
			key:     "srv_0123456789abcdef0123456789abcdef",
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
		{
			name:    "missing prefix",
			key:     "0123456789abcdef0123456789abcdef",
			wantErr: true,
		},
		{
			name:    "wrong prefix",
			key:     "key_0123456789abcdef0123456789abcdef",
			wantErr: true,
		},
		{
			name:    "too short",
			key:     "srv_abc123",
			wantErr: true,
		},
		{
			name:    "too long",
			key:     "srv_0123456789abcdef0123456789abcdef123",
			wantErr: true,
		},
		{
			name:    "uppercase letters",
			key:     "srv_0123456789ABCDEF0123456789ABCDEF",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			key:     "srv_0123456789abcdef0123456789abcdeg",
			wantErr: true,
		},
		{
			name:    "sql injection attempt",
			key:     "srv_'; DROP TABLE users; --",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateServerKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServerKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateServerName(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		srvName string
		wantErr bool
	}{
		{
			name:    "valid name",
			srvName: "MyServer",
			wantErr: false,
		},
		{
			name:    "valid name with spaces",
			srvName: "My Production Server",
			wantErr: false,
		},
		{
			name:    "valid name with numbers",
			srvName: "Server123",
			wantErr: false,
		},
		{
			name:    "empty name",
			srvName: "",
			wantErr: true,
		},
		{
			name:    "too long",
			srvName: "This is a very long server name that exceeds the maximum allowed length of 50 characters",
			wantErr: true,
		},
		{
			name:    "xss attempt",
			srvName: "<script>alert('xss')</script>",
			wantErr: true,
		},
		{
			name:    "sql injection attempt",
			srvName: "Server'; DROP TABLE--",
			wantErr: true,
		},
		{
			name:    "command injection attempt",
			srvName: "Server; rm -rf /",
			wantErr: true,
		},
		{
			name:    "dangerous quote",
			srvName: "Server\"Test",
			wantErr: true,
		},
		{
			name:    "ampersand",
			srvName: "Server&Test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateServerName(tt.srvName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServerName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateContainerID(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "valid container ID",
			id:      "my-container",
			wantErr: false,
		},
		{
			name:    "valid with underscores",
			id:      "my_container_123",
			wantErr: false,
		},
		{
			name:    "valid with dots",
			id:      "my.container.app",
			wantErr: false,
		},
		{
			name:    "too short",
			id:      "ab",
			wantErr: true,
		},
		{
			name:    "too long",
			id:      "this-is-a-very-long-container-id-that-exceeds-the-maximum-allowed-length-of-64-characters",
			wantErr: true,
		},
		{
			name:    "protected servereye-bot",
			id:      "servereye-bot-final",
			wantErr: true,
		},
		{
			name:    "protected redis",
			id:      "redis-cache",
			wantErr: true,
		},
		{
			name:    "protected postgres",
			id:      "postgres-db",
			wantErr: true,
		},
		{
			name:    "protected db",
			id:      "my-db-container",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			id:      "container/with/slashes",
			wantErr: true,
		},
		{
			name:    "invalid start",
			id:      "-invalid-start",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateContainerID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContainerID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "clean input",
			input: "Normal text",
			want:  "Normal text",
		},
		{
			name:  "with leading/trailing spaces",
			input: "  Text with spaces  ",
			want:  "Text with spaces",
		},
		{
			name:  "with null bytes",
			input: "Text\x00with\x00nulls",
			want:  "Textwithnulls",
		},
		{
			name:  "with control characters",
			input: "Text\x01\x02\x03control",
			want:  "Textcontrol",
		},
		{
			name:  "with newlines",
			input: "Line1\nLine2\rLine3",
			want:  "Line1Line2Line3",
		},
		{
			name:  "with tabs",
			input: "Text\twith\ttabs",
			want:  "Textwithtabs",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   \t\n   ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.SanitizeInput(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeInput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewInputValidator(t *testing.T) {
	validator := NewInputValidator()

	if validator == nil {
		t.Fatal("NewInputValidator() returned nil")
	}

	if validator.serverKeyRegex == nil {
		t.Error("serverKeyRegex is nil")
	}

	if validator.containerIDRegex == nil {
		t.Error("containerIDRegex is nil")
	}

	if len(validator.protectedContainers) == 0 {
		t.Error("protectedContainers is empty")
	}
}
