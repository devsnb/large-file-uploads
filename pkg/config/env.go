// Package config provides functionality for loading and accessing
// application configuration from config.yml and environment variables.
package config

import (
	"os"
	"strconv"
	"strings"
)

// EnvString retrieves an environment variable with the given key,
// or returns the default value if not set
func EnvString(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// EnvBool retrieves a boolean environment variable with the given key,
// or returns the default value if not set
func EnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		lower := strings.ToLower(value)
		if lower == "true" || lower == "1" || lower == "yes" {
			return true
		}
		if lower == "false" || lower == "0" || lower == "no" {
			return false
		}
	}
	return defaultValue
}

// EnvInt retrieves an integer environment variable with the given key,
// or returns the default value if not set or invalid
func EnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// EnvFloat retrieves a float environment variable with the given key,
// or returns the default value if not set or invalid
func EnvFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

// EnvStringSlice retrieves a comma-separated list environment variable with the given key,
// or returns the default value if not set
func EnvStringSlice(key string, defaultValue []string) []string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result
	}
	return defaultValue
}

// FormatKey formats a configuration key for environment variable lookup
func FormatKey(prefix, key string) string {
	result := strings.ReplaceAll(key, ".", "_")
	result = strings.ToUpper(result)
	if prefix != "" {
		result = prefix + "_" + result
	}
	return result
}
