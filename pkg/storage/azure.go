package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tus/tusd/v2/pkg/azurestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	"github.com/tus/tusd/v2/pkg/memorylocker"
)

// AzureConfig holds configuration specific to Azure Blob Storage
type AzureConfig struct {
	AccountName         string `json:"accountName"`
	AccountKey          string `json:"accountKey"`
	ContainerName       string `json:"containerName"`
	Endpoint            string `json:"endpoint"` // Optional, used for Azurite testing
	BlobAccessTier      string `json:"blobAccessTier"`
	ContainerAccessType string `json:"containerAccessType"`
}

// AzureStorage implements Storage interface for Azure Blob Storage
type AzureStorage struct {
	config      AzureConfig
	service     azurestore.AzService
	composer    *tusd.StoreComposer
	initialized bool
}

// NewAzureStorage creates a new Azure Blob Storage instance
func NewAzureStorage() *AzureStorage {
	return &AzureStorage{
		composer:    tusd.NewStoreComposer(),
		initialized: false,
	}
}

// Initialize sets up the Azure Blob Storage service and configures the storage
func (s *AzureStorage) Initialize(ctx context.Context, cfg *Config) error {
	// Default values
	azureCfg := AzureConfig{
		ContainerName:       "uploads",
		ContainerAccessType: "private",
	}

	// Override with provided configuration if any
	if cfg.Properties != nil {
		if accountName, ok := cfg.Properties["accountName"].(string); ok && accountName != "" {
			azureCfg.AccountName = accountName
		}

		if accountKey, ok := cfg.Properties["accountKey"].(string); ok && accountKey != "" {
			azureCfg.AccountKey = accountKey
		}

		if containerName, ok := cfg.Properties["containerName"].(string); ok && containerName != "" {
			azureCfg.ContainerName = containerName
		}

		if endpoint, ok := cfg.Properties["endpoint"].(string); ok && endpoint != "" {
			azureCfg.Endpoint = endpoint
		}

		if blobAccessTier, ok := cfg.Properties["blobAccessTier"].(string); ok && blobAccessTier != "" {
			azureCfg.BlobAccessTier = blobAccessTier
		}

		if containerAccessType, ok := cfg.Properties["containerAccessType"].(string); ok && containerAccessType != "" {
			azureCfg.ContainerAccessType = containerAccessType
		}
	}

	// Validate required Azure configuration
	if azureCfg.AccountName == "" {
		return fmt.Errorf("azure account name is required: %w", ErrInvalidConfig)
	}

	if azureCfg.AccountKey == "" {
		return fmt.Errorf("azure account key is required: %w", ErrInvalidConfig)
	}

	// Store the configuration
	s.config = azureCfg

	// Create Azure configuration for tusd
	azConfig := azurestore.AzConfig{
		AccountName:         azureCfg.AccountName,
		AccountKey:          azureCfg.AccountKey,
		ContainerName:       azureCfg.ContainerName,
		BlobAccessTier:      azureCfg.BlobAccessTier,
		ContainerAccessType: azureCfg.ContainerAccessType,
	}

	// If custom endpoint is provided, use it (useful for Azurite emulation)
	if azureCfg.Endpoint != "" {
		azConfig.Endpoint = azureCfg.Endpoint
		slog.Info("Using custom Azure endpoint", "endpoint", azureCfg.Endpoint)
	}

	// Log the configuration details
	slog.Info("Setting up Azure Blob Storage",
		"account", azureCfg.AccountName,
		"container", azureCfg.ContainerName,
		"customEndpoint", azureCfg.Endpoint != "",
	)

	// Create Azure service
	service, err := azurestore.NewAzureService(&azConfig)
	if err != nil {
		return fmt.Errorf("error creating Azure service: %w", err)
	}

	// Create Azure store for tusd
	store := azurestore.New(service)

	// Create in-memory locker
	locker := memorylocker.New()

	// Configure composer with explicit support for creation
	s.composer = tusd.NewStoreComposer()

	// Enable all required extensions for proper file upload
	locker.UseIn(s.composer) // For file locking
	store.UseIn(s.composer)  // For data storage

	// Extra debug logging
	slog.Debug("Azure store configured",
		"provider", "Azure",
		"container", azureCfg.ContainerName)

	// Store the service reference
	s.service = service
	s.initialized = true

	return nil
}

// GetHandler returns a configured tusd handler for Azure Blob Storage
func (s *AzureStorage) GetHandler(basePath string) (*tusd.Handler, error) {
	if !s.initialized {
		return nil, ErrStorageNotConfigured
	}

	config := tusd.Config{
		BasePath:              basePath,
		StoreComposer:         s.composer,
		NotifyCompleteUploads: true,
		DisableDownload:       false,
	}

	slog.Debug("Creating TUS handler for Azure",
		"basePath", basePath,
		"disableDownload", config.DisableDownload)

	handler, err := tusd.NewHandler(config)
	if err != nil {
		return nil, fmt.Errorf("error creating handler: %w", err)
	}

	return handler, nil
}

// GetProvider returns the storage provider type
func (s *AzureStorage) GetProvider() Provider {
	return Azure
}

// GetStoreComposer returns the tusd store composer
func (s *AzureStorage) GetStoreComposer() *tusd.StoreComposer {
	return s.composer
}
