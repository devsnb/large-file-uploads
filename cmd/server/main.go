package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lmittmann/tint"
	"github.com/tus/tusd/v2/pkg/handler"

	"github.com/devsnb/large-file-uploads/pkg/config"
	"github.com/devsnb/large-file-uploads/pkg/storage"
)

func main() {
	cfg, err := config.Load("config.yml")
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Setup logging
	logLevel := slog.LevelInfo
	if cfg.App.Debug {
		logLevel = slog.LevelDebug
	}

	// Custom log handler to filter excessive error messages
	logHandler := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      logLevel,
		TimeFormat: time.DateTime,
	})

	// Set up the logger with our custom handler
	slog.SetDefault(slog.New(logHandler))

	// Log basic configuration information
	slog.Info("Configuration loaded successfully",
		"path", "config.yml",
		"environment", cfg.App.Environment)

	// Determine storage provider from environment or config
	storageProvider := string(storage.MinIO)
	if cfg.Storage.Type != "" {
		storageProvider = cfg.Storage.Type
		slog.Info("Using storage provider from config", "provider", storageProvider)
	} else if os.Getenv("STORAGE_TYPE") != "" {
		storageProvider = os.Getenv("STORAGE_TYPE")
		slog.Info("Using storage provider from environment", "provider", storageProvider)
	} else {
		slog.Info("No storage provider specified, defaulting to MinIO")
	}

	// Create storage factory and initialize storage backend
	factory := storage.NewFactory()
	store, err := factory.CreateFromEnv(context.Background())
	if err != nil {
		slog.Error("Failed to create storage", "error", err)
		os.Exit(1)
	}

	slog.Info("Storage backend initialized successfully", "provider", store.GetProvider())

	// Get the tus handler
	tusHandler, err := store.GetHandler("/files/")
	if err != nil {
		slog.Error("Failed to create tus handler", "error", err)
		os.Exit(1)
	}

	// Add hooks for logging
	tusHandler.CompleteUploads = make(chan handler.HookEvent)
	go func() {
		for event := range tusHandler.CompleteUploads {
			slog.Info("Upload completed",
				"id", event.Upload.ID,
				"size", event.Upload.Size,
				"offset", event.Upload.Offset,
				"metadata", event.Upload.MetaData)
		}
	}()

	// Set up Gin router
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New() // Use New() instead of Default() to avoid using the default logger

	// Add our custom request logger middleware
	r.Use(requestLoggerMiddleware())

	// Add recovery middleware to handle panics
	r.Use(gin.Recovery())

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{
			"Authorization",
			"Content-Type",
			"Tus-Resumable",
			"Upload-Length",
			"Upload-Metadata",
			"Upload-Offset",
			"Content-Length",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Location",
			"Tus-Resumable",
			"Upload-Length",
			"Upload-Offset",
			"Upload-Metadata",
			"Content-Type",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"storage": string(store.GetProvider()),
		})
	})

	// Define routes with middleware
	tusGroup := r.Group("/files")

	// Temporarily disable authentication for testing
	// TODO: Re-enable and ensure auth.JWTMiddleware is defined and exported
	// tusGroup.Use(auth.JWTMiddleware())

	// Handle all TUS protocol methods using the simplified StripPrefix approach
	// This uses gin.WrapH to directly wrap the HTTP handler with a StripPrefix handler
	// which is the method from the working code
	tusGroup.Any("/*any", gin.WrapH(http.StripPrefix("/files/", tusHandler)))

	// Determine port from config or environment
	port := "8080"
	if cfg.App.Port != 0 {
		port = fmt.Sprintf("%d", cfg.App.Port)
	} else if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	// Start server
	slog.Info(fmt.Sprintf("Server starting on port %s", port))
	err = r.Run(":" + port)
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

// requestLoggerMiddleware returns a gin middleware for logging HTTP requests and responses
func requestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Get request headers
		headers := map[string]string{}
		for k, v := range c.Request.Header {
			// Skip sensitive headers
			if strings.ToLower(k) == "authorization" {
				headers[k] = "REDACTED"
				continue
			}
			headers[k] = strings.Join(v, ",")
		}

		// Log request
		slog.Info("Request received",
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
			"headers", fmt.Sprintf("%v", headers),
		)

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Get response status
		statusCode := c.Writer.Status()
		statusClass := statusCode / 100

		// Log level based on status code
		var logFn func(msg string, args ...any)
		switch statusClass {
		case 5: // 5xx
			logFn = slog.Error
		case 4: // 4xx
			// Filter common errors that we don't want to spam logs with
			if strings.Contains(c.Errors.String(), "feature not supported") {
				logFn = slog.Debug // Downgrade to debug level
			} else {
				logFn = slog.Warn
			}
		default: // 2xx, 3xx
			logFn = slog.Info
		}

		// Log response
		logFn("Request completed",
			"method", c.Request.Method,
			"path", path,
			"status", statusCode,
			"duration_ms", duration.Milliseconds(),
			"content_length", c.Writer.Size(),
			"errors", c.Errors.String(),
		)
	}
}
