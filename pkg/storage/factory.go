package storage

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// Factory creates storage implementations based on configuration
type Factory struct {
	registry *Registry
}

// NewFactory creates a new storage factory with all supported providers
func NewFactory() *Factory {
	registry := NewRegistry()

	// Register all supported providers
	registry.Register(MinIO, NewMinIOStorage())
	registry.Register(Azure, NewAzureStorage())

	return &Factory{
		registry: registry,
	}
}

// CreateFromEnv creates a storage implementation based on environment variables
func (f *Factory) CreateFromEnv(ctx context.Context) (Storage, error) {
	// Determine storage type from environment
	storageType := os.Getenv("STORAGE_TYPE")
	if storageType == "" {
		storageType = string(MinIO) // Default to MinIO
	}

	provider := Provider(strings.ToLower(storageType))

	// Create configuration based on the provider
	cfg := &Config{
		Provider:   provider,
		Properties: make(map[string]interface{}),
	}

	// Load provider-specific configuration from environment variables
	switch provider {
	case MinIO:
		cfg.Properties["endpoint"] = getEnv("MINIO_ENDPOINT", "localhost:9000")
		cfg.Properties["bucket"] = getEnv("MINIO_BUCKET", "uploads")
		cfg.Properties["region"] = getEnv("MINIO_REGION", "us-east-1")
		cfg.Properties["accessKey"] = getEnv("MINIO_ACCESS_KEY", "minioadmin")
		cfg.Properties["secretKey"] = getEnv("MINIO_SECRET_KEY", "minioadmin")
		cfg.Properties["useSSL"] = getEnvBool("MINIO_USE_SSL", false)
		cfg.Properties["pathStyle"] = true
		cfg.Properties["disableSSL"] = !getEnvBool("MINIO_USE_SSL", false)

	case Azure:
		cfg.Properties["accountName"] = getEnv("AZURE_STORAGE_ACCOUNT", "")
		cfg.Properties["accountKey"] = getEnv("AZURE_STORAGE_KEY", "")
		cfg.Properties["containerName"] = getEnv("AZURE_STORAGE_CONTAINER", "uploads")
		cfg.Properties["endpoint"] = getEnv("AZURE_STORAGE_ENDPOINT", "")
		cfg.Properties["blobAccessTier"] = getEnv("AZURE_BLOB_ACCESS_TIER", "")
		cfg.Properties["containerAccessType"] = getEnv("AZURE_CONTAINER_ACCESS_TYPE", "private")

	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", provider)
	}

	// Initialize the storage provider
	return f.registry.NewStorageFromConfig(ctx, cfg)
}

// CreateFromConfig creates a storage implementation based on explicit configuration
func (f *Factory) CreateFromConfig(ctx context.Context, cfg *Config) (Storage, error) {
	return f.registry.NewStorageFromConfig(ctx, cfg)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvBool gets a boolean environment variable or returns a default value
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return strings.ToLower(value) == "true" ||
		strings.ToLower(value) == "yes" ||
		strings.ToLower(value) == "1" ||
		strings.ToLower(value) == "on"
}
 