// Package config provides functionality for loading and accessing
// application configuration from config.yml and environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Constants for configuration paths and environment variables
const (
	DefaultConfigPath = "config.yml"
	EnvPrefix         = "APP_"
)

// Config represents the application configuration structure
type Config struct {
	App     AppConfig     `yaml:"app"`
	Storage StorageConfig `yaml:"storage"`
	Logging LoggingConfig `yaml:"logging"`
	CORS    CORSConfig    `yaml:"cors"`
}

// AppConfig contains general application settings
type AppConfig struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	Port        int    `yaml:"port"`
	Debug       bool   `yaml:"debug"`
	Timeout     int    `yaml:"timeout"`
}

// StorageConfig contains settings for various storage backends
type StorageConfig struct {
	Type  string       `yaml:"type"`
	Local LocalStorage `yaml:"local"`
	S3    S3Storage    `yaml:"s3"`
	Azure AzureStorage `yaml:"azure"`
	Minio MinioStorage `yaml:"minio"`
}

// LocalStorage configuration
type LocalStorage struct {
	RootDir string `yaml:"rootDir"`
	TempDir string `yaml:"tempDir"`
}

// S3Storage configuration
type S3Storage struct {
	Region    string `yaml:"region"`
	Bucket    string `yaml:"bucket"`
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	Endpoint  string `yaml:"endpoint"`
}

// AzureStorage configuration
type AzureStorage struct {
	AccountName   string `yaml:"accountName"`
	AccountKey    string `yaml:"accountKey"`
	ContainerName string `yaml:"containerName"`
}

// MinioStorage configuration
type MinioStorage struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	SSL       bool   `yaml:"ssl"`
	Bucket    string `yaml:"bucket"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowedOrigins"`
	AllowedMethods []string `yaml:"allowedMethods"`
	AllowedHeaders []string `yaml:"allowedHeaders"`
	MaxAge         int      `yaml:"maxAge"`
}

var (
	instance *Config
	once     sync.Once
)

// Load reads configuration from the specified file path or the default path
// if not provided. It also applies environment variable overrides.
func Load(configPath string) (*Config, error) {
	var loadErr error

	once.Do(func() {
		if configPath == "" {
			configPath = DefaultConfigPath
		}

		// Load config from YAML file
		cfg, err := loadFromFile(configPath)
		if err != nil {
			loadErr = fmt.Errorf("failed to load config from file: %w", err)
			return
		}

		// Override with environment variables
		applyEnvironmentOverrides(cfg)

		instance = cfg
		slog.Info("configuration loaded successfully",
			"path", configPath,
			"environment", cfg.App.Environment)
	})

	if loadErr != nil {
		return nil, loadErr
	}

	return instance, nil
}

// Get returns the singleton configuration instance.
// It loads the configuration from the default path if not already loaded.
func Get() (*Config, error) {
	if instance == nil {
		return Load("")
	}
	return instance, nil
}

// loadFromFile reads and parses the YAML configuration file
func loadFromFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %w", err)
	}
	defer file.Close()

	config := &Config{}
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("could not decode config file: %w", err)
	}

	return config, nil
}

// applyEnvironmentOverrides overrides configuration values from environment variables
func applyEnvironmentOverrides(cfg *Config) {
	// Get all environment variables
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, EnvPrefix) {
			continue
		}

		// Split key and value
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimPrefix(parts[0], EnvPrefix)
		value := parts[1]

		// Apply overrides based on key patterns
		applyEnvOverride(cfg, key, value)
	}
}

// applyEnvOverride applies a single environment variable override to the config
func applyEnvOverride(cfg *Config, key, value string) {
	// Convert APP_STORAGE_TYPE to storage.type in the config
	key = strings.ToLower(key)

	// Apply based on specific keys
	// This is a simple implementation that could be extended for more complex cases
	switch {
	case key == "app_port":
		var port int
		if _, err := fmt.Sscanf(value, "%d", &port); err == nil {
			cfg.App.Port = port
		}
	case key == "app_debug":
		cfg.App.Debug = strings.ToLower(value) == "true"
	case key == "app_environment":
		cfg.App.Environment = value
	case key == "storage_type":
		cfg.Storage.Type = value
	case key == "s3_accesskey":
		cfg.Storage.S3.AccessKey = value
	case key == "s3_secretkey":
		cfg.Storage.S3.SecretKey = value
	case key == "s3_bucket":
		cfg.Storage.S3.Bucket = value
	case key == "s3_region":
		cfg.Storage.S3.Region = value
	case key == "azure_accountkey":
		cfg.Storage.Azure.AccountKey = value
	case key == "azure_accountname":
		cfg.Storage.Azure.AccountName = value
	case key == "azure_containername":
		cfg.Storage.Azure.ContainerName = value
	case key == "minio_accesskey":
		cfg.Storage.Minio.AccessKey = value
	case key == "minio_secretkey":
		cfg.Storage.Minio.SecretKey = value
	case key == "minio_bucket":
		cfg.Storage.Minio.Bucket = value
	case key == "logging_level":
		cfg.Logging.Level = value
	}
}

// Validate performs validation on the configuration values
func (c *Config) Validate() error {
	// Basic validation
	if c.App.Port <= 0 {
		return fmt.Errorf("invalid port: %d", c.App.Port)
	}

	// Validate storage configuration based on type
	switch c.Storage.Type {
	case "local":
		if c.Storage.Local.RootDir == "" {
			return fmt.Errorf("local storage requires rootDir to be set")
		}
		// Create dirs if they don't exist
		if err := os.MkdirAll(c.Storage.Local.RootDir, 0755); err != nil {
			return fmt.Errorf("failed to create rootDir: %w", err)
		}
		if c.Storage.Local.TempDir != "" {
			if err := os.MkdirAll(c.Storage.Local.TempDir, 0755); err != nil {
				return fmt.Errorf("failed to create tempDir: %w", err)
			}
		}
	case "s3":
		if c.Storage.S3.Bucket == "" {
			return fmt.Errorf("s3 storage requires bucket to be set")
		}
		// Credentials can be loaded from environment or instance profile
	case "azure":
		if c.Storage.Azure.ContainerName == "" {
			return fmt.Errorf("azure storage requires containerName to be set")
		}
	case "minio":
		if c.Storage.Minio.Endpoint == "" || c.Storage.Minio.Bucket == "" {
			return fmt.Errorf("minio storage requires endpoint and bucket to be set")
		}
	default:
		return fmt.Errorf("unsupported storage type: %s", c.Storage.Type)
	}

	return nil
}

// GetStoragePath returns an absolute path by joining the provided path
// with the root storage directory for local storage
func (c *Config) GetStoragePath(path string) string {
	if c.Storage.Type != "local" {
		return path
	}
	return filepath.Join(c.Storage.Local.RootDir, path)
}

// IsDevelopment returns true if the application is running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// IsProduction returns true if the application is running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}
