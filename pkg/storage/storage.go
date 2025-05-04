// Package storage provides interfaces and implementations for various storage backends
package storage

import (
	"context"
	"errors"
	"fmt"

	tusd "github.com/tus/tusd/v2/pkg/handler"
)

// Common errors returned by storage operations
var (
	ErrStorageNotConfigured = errors.New("storage not properly configured")
	ErrInvalidConfig        = errors.New("invalid configuration")
	ErrStorageUnavailable   = errors.New("storage unavailable")
)

// Provider identifies supported storage providers
type Provider string

const (
	// MinIO represents S3-compatible storage (MinIO, AWS S3, etc.)
	MinIO Provider = "minio"

	// Azure represents Azure Blob Storage
	Azure Provider = "azure"

	// Disk represents local disk storage
	Disk Provider = "disk"

	// Memory represents in-memory storage (for testing)
	Memory Provider = "memory"
)

// Config represents the abstract configuration for any storage provider
type Config struct {
	// Provider specifies which storage backend to use
	Provider Provider

	// Additional provider-specific configuration is stored in Properties
	Properties map[string]interface{}
}

// Storage is the interface that all storage backend implementations must satisfy
type Storage interface {
	// Initialize sets up the storage backend with the provided configuration
	Initialize(ctx context.Context, cfg *Config) error

	// GetHandler returns a tusd handler configured with this storage backend
	GetHandler(basePath string) (*tusd.Handler, error)

	// GetProvider returns the provider type for this storage implementation
	GetProvider() Provider

	// GetStoreComposer returns the tusd StoreComposer for this storage backend
	GetStoreComposer() *tusd.StoreComposer
}

// Registry keeps track of all storage implementations
type Registry struct {
	providers map[Provider]Storage
}

// NewRegistry creates a new storage registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[Provider]Storage),
	}
}

// Register adds a storage implementation to the registry
func (r *Registry) Register(provider Provider, storage Storage) {
	r.providers[provider] = storage
}

// Get returns a storage implementation for the specified provider
func (r *Registry) Get(provider Provider) (Storage, error) {
	if storage, ok := r.providers[provider]; ok {
		return storage, nil
	}
	return nil, fmt.Errorf("storage provider %s not found", provider)
}

// NewStorageFromConfig creates and initializes a storage backend from the provided configuration
func (r *Registry) NewStorageFromConfig(ctx context.Context, cfg *Config) (Storage, error) {
	storage, err := r.Get(cfg.Provider)
	if err != nil {
		return nil, err
	}

	if err := storage.Initialize(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return storage, nil
}
 