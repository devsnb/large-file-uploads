package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	tusd "github.com/tus/tusd/v2/pkg/handler"
	"github.com/tus/tusd/v2/pkg/memorylocker"
	"github.com/tus/tusd/v2/pkg/s3store"
)

// S3Config holds configuration specific to S3-compatible storage
type S3Config struct {
	Endpoint   string `json:"endpoint"`
	Bucket     string `json:"bucket"`
	Region     string `json:"region"`
	AccessKey  string `json:"accessKey"`
	SecretKey  string `json:"secretKey"`
	UseSSL     bool   `json:"useSSL"`
	PathStyle  bool   `json:"pathStyle"` // Use path-style URLs (required for MinIO)
	DisableSSL bool   `json:"disableSSL"`
}

// MinIOStorage implements Storage interface for S3-compatible storage providers
type MinIOStorage struct {
	config      S3Config
	s3Client    *s3.Client
	composer    *tusd.StoreComposer
	initialized bool
}

// NewMinIOStorage creates a new S3-compatible storage instance
func NewMinIOStorage() *MinIOStorage {
	return &MinIOStorage{
		composer:    tusd.NewStoreComposer(),
		initialized: false,
	}
}

// Initialize sets up the S3 client and configures the storage
func (s *MinIOStorage) Initialize(ctx context.Context, cfg *Config) error {
	// Default values
	s3Cfg := S3Config{
		Endpoint:   "localhost:9000",
		Bucket:     "uploads",
		Region:     "us-east-1",
		AccessKey:  "minioadmin",
		SecretKey:  "minioadmin",
		UseSSL:     false,
		PathStyle:  true,
		DisableSSL: true,
	}

	// Override with provided configuration if any
	if cfg.Properties != nil {
		if endpoint, ok := cfg.Properties["endpoint"].(string); ok && endpoint != "" {
			s3Cfg.Endpoint = endpoint
		}

		if bucket, ok := cfg.Properties["bucket"].(string); ok && bucket != "" {
			s3Cfg.Bucket = bucket
		}

		if region, ok := cfg.Properties["region"].(string); ok && region != "" {
			s3Cfg.Region = region
		}

		if accessKey, ok := cfg.Properties["accessKey"].(string); ok && accessKey != "" {
			s3Cfg.AccessKey = accessKey
		}

		if secretKey, ok := cfg.Properties["secretKey"].(string); ok && secretKey != "" {
			s3Cfg.SecretKey = secretKey
		}

		if useSSL, ok := cfg.Properties["useSSL"].(bool); ok {
			s3Cfg.UseSSL = useSSL
		}

		if pathStyle, ok := cfg.Properties["pathStyle"].(bool); ok {
			s3Cfg.PathStyle = pathStyle
		}

		if disableSSL, ok := cfg.Properties["disableSSL"].(bool); ok {
			s3Cfg.DisableSSL = disableSSL
		}
	}

	// Store the configuration
	s.config = s3Cfg

	slog.Info("Setting up S3-compatible storage",
		"endpoint", s3Cfg.Endpoint,
		"bucket", s3Cfg.Bucket,
		"region", s3Cfg.Region,
		"useSSL", s3Cfg.UseSSL)

	// Construct the MinIO URL with appropriate protocol
	protocol := "http"
	if s3Cfg.UseSSL {
		protocol = "https"
	}

	// Create the full URL for MinIO
	minioURL := s3Cfg.Endpoint
	if len(minioURL) < 4 || (minioURL[:4] != "http" && minioURL[:5] != "https") {
		minioURL = fmt.Sprintf("%s://%s", protocol, s3Cfg.Endpoint)
	}

	// Use a simplified resolver for MinIO
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               minioURL,
			HostnameImmutable: true,
			Source:            aws.EndpointSourceCustom,
		}, nil
	})

	// Set up AWS SDK configuration with simplified approach
	awsOpts := []func(*config.LoadOptions) error{
		config.WithRegion(s3Cfg.Region),
		config.WithEndpointResolverWithOptions(resolver),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s3Cfg.AccessKey, s3Cfg.SecretKey, ""),
		),
	}

	// Load the AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	// Create S3 client with path-style access enabled
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // Essential for MinIO
	})

	s.s3Client = s3Client

	// Verify bucket exists or create it
	_, err = s.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s3Cfg.Bucket),
	})

	if err != nil {
		slog.Info("Bucket does not exist. Creating...", "bucket", s3Cfg.Bucket)
		_, err = s.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(s3Cfg.Bucket),
		})
		if err != nil {
			return fmt.Errorf("error creating bucket: %w", err)
		}
		slog.Info("Bucket created successfully", "bucket", s3Cfg.Bucket)
	}

	// Create S3 store for tusd with the configured client
	store := s3store.New(s3Cfg.Bucket, s.s3Client)

	// Create in-memory locker
	locker := memorylocker.New()

	// Configure composer with explicit support for creation
	s.composer = tusd.NewStoreComposer()

	// Enable all required extensions for proper file upload
	locker.UseIn(s.composer) // For file locking
	store.UseIn(s.composer)  // For data storage

	// Extra debug logging
	slog.Debug("S3 store configured",
		"provider", "MinIO",
		"bucket", s3Cfg.Bucket)

	s.initialized = true

	return nil
}

// GetHandler returns a configured tusd handler for S3 storage
func (s *MinIOStorage) GetHandler(basePath string) (*tusd.Handler, error) {
	if !s.initialized {
		return nil, ErrStorageNotConfigured
	}

	config := tusd.Config{
		BasePath:              basePath,
		StoreComposer:         s.composer,
		NotifyCompleteUploads: true,
		DisableDownload:       false,
	}

	slog.Debug("Creating TUS handler",
		"basePath", basePath,
		"disableDownload", config.DisableDownload)

	handler, err := tusd.NewHandler(config)
	if err != nil {
		return nil, fmt.Errorf("error creating handler: %w", err)
	}

	return handler, nil
}

// GetProvider returns the storage provider type
func (s *MinIOStorage) GetProvider() Provider {
	return MinIO
}

// GetStoreComposer returns the tusd store composer
func (s *MinIOStorage) GetStoreComposer() *tusd.StoreComposer {
	return s.composer
}
