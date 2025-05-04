//go:build integration
// +build integration

package config

import (
	"os"
	"testing"
)

// TestRealConfigFile tests loading the actual config.yml from project root
// This is an integration test and only runs with the 'integration' build tag:
// go test -tags=integration ./pkg/config
func TestRealConfigFile(t *testing.T) {
	// Save the original env values
	origPort := os.Getenv("APP_APP_PORT")
	defer func() {
		// Restore env
		if origPort != "" {
			os.Setenv("APP_APP_PORT", origPort)
		} else {
			os.Unsetenv("APP_APP_PORT")
		}
	}()

	// Reset singleton instance for testing
	instance = nil

	// Set a test env var
	os.Setenv("APP_APP_PORT", "9999")

	// Try to load the actual config from project root
	cfg, err := Get()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Make sure the configuration loaded and env var override worked
	if cfg.App.Port != 9999 {
		t.Errorf("Environment override failed: expected port 9999, got %d", cfg.App.Port)
	}

	// Validate loaded config
	if err := cfg.Validate(); err != nil {
		t.Errorf("Config validation failed: %v", err)
	}

	// Test some expected values
	if cfg.App.Name != "large-file-uploads" {
		t.Errorf("Incorrect app name: got %s, want large-file-uploads", cfg.App.Name)
	}
}
