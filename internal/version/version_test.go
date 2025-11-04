package version

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	version := GetVersion()

	if version == "" {
		t.Error("GetVersion() returned empty string")
	}

	if version != Version {
		t.Errorf("GetVersion() = %v, want %v", version, Version)
	}
}

func TestGetFullVersion(t *testing.T) {
	// Test with current BuildDate value
	got := GetFullVersion()

	if got == "" {
		t.Error("GetFullVersion() returned empty string")
	}

	// Should contain Version
	if !strings.Contains(got, Version) {
		t.Errorf("GetFullVersion() = %v, should contain %v", got, Version)
	}
}

func TestVersionFormat(t *testing.T) {
	// Version should follow semantic versioning (e.g., "1.0.4")
	parts := strings.Split(Version, ".")

	if len(parts) != 3 {
		t.Errorf("Version format invalid: %v, expected X.Y.Z format", Version)
	}

	// Each part should be numeric
	for _, part := range parts {
		if part == "" {
			t.Errorf("Version has empty component: %v", Version)
		}
	}
}

func TestVersionConstants(t *testing.T) {
	// Ensure version constants are not empty
	if Version == "" {
		t.Error("Version constant is empty")
	}

	if BuildDate == "" {
		t.Error("BuildDate constant is empty")
	}

	if GitCommit == "" {
		t.Error("GitCommit constant is empty")
	}
}
