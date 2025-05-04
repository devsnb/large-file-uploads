# Large File Uploads Service

A robust & reliable service for handling large file uploads with resumable capabilities using the [tus protocol](https://tus.io/). This Go-based application provides a secure, scalable solution with multiple storage backend options.

## Features

- **Resumable File Uploads**: Implements the tus protocol for reliable uploads that can resume after network interruptions
- **Multiple Storage Backends**:
  - MinIO/S3-compatible storage (default)
  - Azure Blob Storage
  - Easily extensible for additional providers
- **Docker-Ready**: Complete Docker and Docker Compose setup for easy deployment
- **Configurable**: YAML configuration with environment variable overrides
- **Production Logging**: Structured JSON logging with customizable log levels
- **CORS Support**: Configurable Cross-Origin Resource Sharing
- **Developer Friendly**: Includes Just commands for common operations

> **Note:** Currently, only the MinIO/S3 storage backend has been thoroughly tested and confirmed working. Azure Blob Storage integration is implemented but not tested at all.

## Project Structure

```
.
├── cmd
│   └── server             # Main application entry point
├── pkg
│   ├── auth               # Authentication middleware and JWT verification
│   ├── config             # Configuration loading and management
│   ├── handler            # HTTP handlers and tus integration
│   └── storage            # Storage backend implementations
│       ├── azure.go       # Azure Blob Storage implementation
│       ├── factory.go     # Storage factory for creating backends
│       ├── minio.go       # MinIO/S3 implementation
│       └── storage.go     # Storage interfaces and abstractions
├── Dockerfile             # Container definition for the server
├── docker-compose.yml     # Multi-container Docker setup
├── config.yml             # Application configuration
├── Justfile               # Command runner for common operations
├── go.mod                 # Go module definition
└── README.md              # This documentation
```

## Requirements

- Go 1.24 or higher
- Docker and Docker Compose (for containerized deployment)
- Just command runner (optional, recommended for convenience)
- MinIO, Azure Storage, or S3-compatible storage

## Architecture

The application follows a modular architecture with clear separation of concerns:

1. **Command Layer** (`cmd/server/`): Entry point that configures and starts the HTTP server
2. **Configuration** (`pkg/config/`): Handles loading settings from YAML and environment variables
3. **Storage Abstraction** (`pkg/storage/`): Interface and implementations for different storage backends
4. **File Upload Handling**: Uses the tus protocol for resumable uploads

> **Note:** While authentication middleware is included in the codebase, it's currently not enabled by default. The service is designed to allow adding an authentication layer on top of the API endpoints as needed.

### Storage Backend System

The application uses a pluggable storage system:

- **Storage Interface**: Common interface that all backends implement
- **Factory Pattern**: Creates the appropriate storage backend based on configuration
- **Supported Backends**:
  - **MinIO/S3**: Uses AWS SDK for S3-compatible storage
  - **Azure Blob Storage**: Integrated with Azure Storage SDK

## Configuration

Configuration is managed through a YAML file (`config.yml`) with environment variable overrides.

### Configuration Structure

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
  type: 'minio' # local, s3, azure, minio

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

  # Azure Blob storage configuration
  azure:
    accountName: ''
    accountKey: ''
    containerName: 'uploads'

  # MinIO configuration
  minio:
    endpoint: 'localhost:9000'
    accessKey: 'minioadmin'
    secretKey: 'minioadmin'
    ssl: false
    bucket: 'uploads'
# Logging and CORS settings...
```

### Environment Variables

Configuration can be overridden with environment variables using the `APP_` prefix:

```bash
# Core application settings
export APP_APP_PORT=9000
export APP_APP_DEBUG=true

# Storage settings
export APP_STORAGE_TYPE=minio
export APP_MINIO_ENDPOINT=localhost:9000
export APP_MINIO_BUCKET=uploads
```

## Running the Application

The easiest way to run the application is using the Just command runner:

```bash
# Start the application with MinIO (default)
just start

# Stop the application
just stop
```

That's it! The `just start` command will:

- Build and start all necessary containers
- Set up MinIO storage
- Create required buckets
- Start the server on port 8080

And `just stop` will gracefully shut down all containers.

## Understanding the tus Protocol

### Why tus?

Traditional file uploads have several limitations that become critical when dealing with large files:

1. **No Resume Capability**: If a connection drops during a large file upload, the entire transfer fails and must be restarted from the beginning.
2. **Timeout Issues**: Web servers and browsers often have timeout limits that can interrupt long uploads.
3. **Network Reliability**: Mobile connections and unstable networks make uploading large files particularly challenging.
4. **User Experience**: Starting from scratch after a failed upload frustrates users and wastes bandwidth.

The tus protocol (Transloadit Upload Server) solves these problems by implementing a standardized, resumable file upload protocol.

### How tus Works

The tus protocol works through a series of HTTP requests with special headers:

1. **Creation**: The client initiates an upload by sending a `POST` request with an `Upload-Length` header specifying the total file size.

```
POST /files/ HTTP/1.1
Host: example.com
Tus-Resumable: 1.0.0
Upload-Length: 100000
```

2. **Server Response**: The server responds with a URL where the upload can continue.

```
HTTP/1.1 201 Created
Location: https://example.com/files/24e533e02ec3bc40c387f1a0e460e216
Tus-Resumable: 1.0.0
```

3. **Chunked Upload**: The client uploads the file in chunks using `PATCH` requests, specifying the current offset.

```
PATCH /files/24e533e02ec3bc40c387f1a0e460e216 HTTP/1.1
Host: example.com
Tus-Resumable: 1.0.0
Upload-Offset: 0
Content-Type: application/offset+octet-stream
Content-Length: 5

hello
```

4. **Progress Tracking**: Each successful chunk upload returns the new offset, allowing the client to keep track of progress.

```
HTTP/1.1 204 No Content
Tus-Resumable: 1.0.0
Upload-Offset: 5
```

5. **Resume After Interruption**: If the connection drops, the client can send a `HEAD` request to find out where to resume.

```
HEAD /files/24e533e02ec3bc40c387f1a0e460e216 HTTP/1.1
Host: example.com
Tus-Resumable: 1.0.0
```

6. **Server Provides Current Offset**: The server tells the client where to resume from.

```
HTTP/1.1 200 OK
Tus-Resumable: 1.0.0
Upload-Offset: 5
Upload-Length: 100000
```

This implementation allows for:

- Uploading files of any size
- Resuming uploads after network interruptions
- Progress tracking
- Metadata attachment to uploads
- Cross-platform compatibility

### Key Benefits of tus in This Service

- **Reliability**: Uploads always complete, even over unreliable connections
- **Performance**: Only the missing chunks need to be uploaded after an interruption
- **Scalability**: Works well with very large files (multiple GB)
- **User Experience**: Provides progress information and resumability
- **Standardization**: Follows an open protocol with clients available for multiple platforms

## Upload Endpoints

The tus upload endpoints are available at `/files/` by default:

#### Creating an Upload

```bash
curl -X POST \
  -H "Tus-Resumable: 1.0.0" \
  -H "Upload-Length: 100000" \
  -H "Upload-Metadata: filename d215ZXNvbWUucG5n" \
  http://localhost:8080/files/
```

This returns a Location header with the upload URL.

#### Uploading Data

```bash
curl -X PATCH \
  -H "Tus-Resumable: 1.0.0" \
  -H "Upload-Offset: 0" \
  -H "Content-Type: application/offset+octet-stream" \
  --data-binary @myfile.jpg \
  http://localhost:8080/files/<upload-id>
```

#### Checking Upload Status

```bash
curl \
  -H "Tus-Resumable: 1.0.0" \
  -I \
  http://localhost:8080/files/<upload-id>
```

### Client Libraries

The tus protocol has client libraries available for various platforms:

- JavaScript: [tus-js-client](https://github.com/tus/tus-js-client)
- Android: [tus-android-client](https://github.com/tus/tus-android-client)
- iOS: [TUSKit](https://github.com/tus/TUSKit)
- Python: [tus-py-client](https://github.com/tus/tus-py-client)

## Development

### Local Development Setup

1. Clone the repository:

   ```bash
   git clone https://github.com/devsnb/large-file-uploads.git
   cd large-file-uploads
   ```

2. Install dependencies:

   ```bash
   go mod download
   ```

3. Start the services:

   ```bash
   just start
   ```

4. Run the server locally:
   ```bash
   go run cmd/server/main.go
   ```

## Testing
This application comes with a test-client at `/test-client`. This is build with the official tus client for javascript and can be used for file upload testing. You can try uploading multiple big files with this client which lets you test upload progress & resumability.