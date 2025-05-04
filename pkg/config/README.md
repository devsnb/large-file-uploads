# Configuration Package

This package provides a simple, flexible configuration system for the application. It loads configuration from a YAML file and supports overriding values with environment variables.

## Features

- YAML-based configuration
- Environment variable overrides
- Singleton configuration instance
- Validation of configuration values
- Helper functions for environment variables
- Support for multiple storage backends

## Usage

### Basic Usage

```go
import "github.com/user/large-file-uploads/pkg/config"

func main() {
    // Load config from default path (config.yml)
    cfg, err := config.Get()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Validate configuration
    if err := cfg.Validate(); err != nil {
        log.Fatalf("Invalid configuration: %v", err)
    }

    // Use configuration values
    log.Printf("Starting %s server on port %d", cfg.App.Name, cfg.App.Port)
}
```

### Loading from a Specific Path

```go
// Load config from a specific path
cfg, err := config.Load("path/to/config.yml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
```

### Environment Variable Overrides

The configuration values can be overridden with environment variables using the format `APP_SECTION_KEY`. For example:

```bash
# Override app port
export APP_APP_PORT=8080

# Override storage type
export APP_STORAGE_TYPE=s3

# Override S3 credentials
export APP_S3_ACCESSKEY=your-access-key
export APP_S3_SECRETKEY=your-secret-key
```

### Configuration File Structure

```yaml
# Application Configuration
app:
  name: 'large-file-uploads'
  environment: 'development' # development, staging, production
  port: 8080
  debug: true
  timeout: 60 # seconds

# Storage Configuration
storage:
  type: 'local' # local, s3, azure, minio

  # Local storage configuration
  local:
    rootDir: './uploads'
    tempDir: './temp'

  # S3 storage configuration
  s3:
    region: 'us-east-1'
    bucket: 'my-uploads-bucket'
    accessKey: '' # Set via environment variables for security
    secretKey: '' # Set via environment variables for security
    endpoint: '' # Optional custom endpoint for S3-compatible services

# ... (other configuration sections)
```

## Helper Functions

The package provides several helper functions for working with environment variables:

```go
// Get string from environment variable or default
value := config.EnvString("ENV_KEY", "default-value")

// Get boolean from environment variable or default
enabled := config.EnvBool("FEATURE_ENABLED", false)

// Get integer from environment variable or default
port := config.EnvInt("SERVER_PORT", 8080)

// Get slice of strings from comma-separated environment variable
origins := config.EnvStringSlice("ALLOWED_ORIGINS", []string{"localhost"})
```

## Testing

The package includes comprehensive tests. Run them with:

```bash
go test -v ./pkg/config
```
