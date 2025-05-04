package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setup creates a temporary YAML configuration file for testing
func setup(t *testing.T) (string, func()) {
	t.Helper()

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a test config file
	configPath := filepath.Join(tmpDir, "config.yml")
	content := []byte(`
app:
  name: "test-app"
  environment: "testing"
  port: 9090
  debug: true
  timeout: 30

storage:
  type: "local"
  local:
    rootDir: "./test-uploads"
    tempDir: "./test-temp"
  s3:
    region: "us-east-1"
    bucket: "default-bucket"
    accessKey: "default-key"
    secretKey: "default-secret"

logging:
  level: "debug"
  format: "text"

cors:
  allowedOrigins:
    - "http://localhost:3000"
  allowedMethods:
    - "GET"
    - "POST"
  allowedHeaders:
    - "Content-Type"
  maxAge: 3600
`)

	if err := os.WriteFile(configPath, content, 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return configPath, cleanup
}

func TestLoadConfig(t *testing.T) {
	configPath, cleanup := setup(t)
	defer cleanup()

	// Reset singleton instance for testing
	instance = nil

	// Test loading with explicit path
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.App.Name != "test-app" {
		t.Errorf("Expected app name 'test-app', got '%s'", cfg.App.Name)
	}
	if cfg.App.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.App.Port)
	}
	if !cfg.App.Debug {
		t.Error("Expected debug to be true")
	}
	if cfg.Storage.Type != "local" {
		t.Errorf("Expected storage type 'local', got '%s'", cfg.Storage.Type)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected logging level 'debug', got '%s'", cfg.Logging.Level)
	}
	if len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("CORS origins not loaded correctly: %v", cfg.CORS.AllowedOrigins)
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	// Create a simple test config directly instead of loading from file
	testConfig := &Config{
		App: AppConfig{
			Name:        "test-app",
			Environment: "testing",
			Port:        9090,
			Debug:       true,
		},
		Storage: StorageConfig{
			Type: "local",
			S3: S3Storage{
				Bucket: "default-bucket",
			},
		},
		Logging: LoggingConfig{
			Level: "debug",
		},
	}

	// Apply environment overrides manually
	os.Setenv("APP_APP_PORT", "8888")
	os.Setenv("APP_STORAGE_TYPE", "s3")
	os.Setenv("APP_LOGGING_LEVEL", "error")
	defer func() {
		os.Unsetenv("APP_APP_PORT")
		os.Unsetenv("APP_STORAGE_TYPE")
		os.Unsetenv("APP_LOGGING_LEVEL")
	}()

	// Call the override function directly
	applyEnvironmentOverrides(testConfig)

	// Verify environment variable overrides
	if testConfig.App.Port != 8888 {
		t.Errorf("Environment override failed: expected port 8888, got %d", testConfig.App.Port)
	}
	if testConfig.Storage.Type != "s3" {
		t.Errorf("Environment override failed: expected storage type 's3', got '%s'", testConfig.Storage.Type)
	}
	if testConfig.Logging.Level != "error" {
		t.Errorf("Environment override failed: expected logging level 'error', got '%s'", testConfig.Logging.Level)
	}
}

func TestGetConfig(t *testing.T) {
	configPath, cleanup := setup(t)
	defer cleanup()

	// Reset singleton instance for testing
	instance = nil

	// First, we manually create and set the singleton
	loadedCfg, err := loadFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	// Set the singleton instance manually
	instance = loadedCfg

	// Get should return the same instance
	cfg, err := Get()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	// Test that we got the correct config instance
	if cfg.App.Name != "test-app" {
		t.Errorf("Expected app name 'test-app', got '%s'", cfg.App.Name)
	}
}

// TestGetWithoutLoad tests the auto-loading feature of Get()
// Note: This test is skipped by default as it would require
// placing a config file in the default location
func TestGetWithoutLoad(t *testing.T) {
	// Skip this test by default since it would require creating a config file
	// in the default location which could interfere with real application configs
	t.Skip("Skipping test that requires default config file")

	// The implementation would look something like:
	//
	// // Reset singleton instance for testing
	// instance = nil
	//
	// // Get should automatically load from default path
	// cfg, err := Get()
	// if err != nil {
	//    t.Fatalf("Failed to get config: %v", err)
	// }
	//
	// // Verify config was loaded correctly
	// if cfg.App.Name == "" {
	//    t.Error("Expected app name to be loaded")
	// }
}

func TestValidate(t *testing.T) {
	// Create a config with missing required fields
	invalidConfig := &Config{
		App: AppConfig{
			Name:        "test-app",
			Environment: "development",
			Port:        0, // Invalid port
		},
		Storage: StorageConfig{
			Type: "s3",
			S3:   S3Storage{
				// Missing bucket
			},
		},
	}

	if err := invalidConfig.Validate(); err == nil {
		t.Error("Expected validation error for invalid port, got nil")
	}

	// Fix port but keep invalid storage config
	invalidConfig.App.Port = 8080
	if err := invalidConfig.Validate(); err == nil {
		t.Error("Expected validation error for missing S3 bucket, got nil")
	}

	// Fix storage config
	invalidConfig.Storage.S3.Bucket = "test-bucket"
	if err := invalidConfig.Validate(); err != nil {
		t.Errorf("Expected no validation error, got: %v", err)
	}
}

func TestEnvHelpers(t *testing.T) {
	// Test EnvString
	os.Setenv("TEST_STRING", "test-value")
	defer os.Unsetenv("TEST_STRING")
	if val := EnvString("TEST_STRING", "default"); val != "test-value" {
		t.Errorf("EnvString failed: got %s, want test-value", val)
	}
	if val := EnvString("NONEXISTENT", "default"); val != "default" {
		t.Errorf("EnvString failed on default: got %s, want default", val)
	}

	// Test EnvBool
	os.Setenv("TEST_BOOL_TRUE", "true")
	os.Setenv("TEST_BOOL_YES", "yes")
	os.Setenv("TEST_BOOL_1", "1")
	os.Setenv("TEST_BOOL_FALSE", "false")
	defer func() {
		os.Unsetenv("TEST_BOOL_TRUE")
		os.Unsetenv("TEST_BOOL_YES")
		os.Unsetenv("TEST_BOOL_1")
		os.Unsetenv("TEST_BOOL_FALSE")
	}()

	if !EnvBool("TEST_BOOL_TRUE", false) {
		t.Error("EnvBool failed for 'true'")
	}
	if !EnvBool("TEST_BOOL_YES", false) {
		t.Error("EnvBool failed for 'yes'")
	}
	if !EnvBool("TEST_BOOL_1", false) {
		t.Error("EnvBool failed for '1'")
	}
	if EnvBool("TEST_BOOL_FALSE", true) {
		t.Error("EnvBool failed for 'false'")
	}
	if !EnvBool("NONEXISTENT", true) {
		t.Error("EnvBool failed with default true")
	}

	// Test EnvInt
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")
	if val := EnvInt("TEST_INT", 0); val != 42 {
		t.Errorf("EnvInt failed: got %d, want 42", val)
	}
	if val := EnvInt("NONEXISTENT", 99); val != 99 {
		t.Errorf("EnvInt failed on default: got %d, want 99", val)
	}

	// Test FormatKey
	if key := FormatKey("APP", "storage.type"); key != "APP_STORAGE_TYPE" {
		t.Errorf("FormatKey failed: got %s, want APP_STORAGE_TYPE", key)
	}
}
