package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the crawler service
type Config struct {
	// Server settings
	Port string
	Host string

	// GitHub settings
	GitHubBaseURL   string
	GitHubToken     string // Personal Access Token
	GitHubAppID     string // GitHub App ID
	GitHubAppKey    string // GitHub App private key
	GitHubInstallID string // GitHub App installation ID

	// Worker pool settings
	MaxWorkers int

	// Rate limiting
	APIRateLimitThreshold int

	// Timeouts and retries
	FetchTimeoutMS     int
	RetryMaxAttempts   int
	RetryBackoffBaseMS int

	// Resource limits
	MaxFileSize          int64 // in bytes
	MaxConcurrentFetches int

	// Enhanced resource management
	MemoryLimitPercent    float64 // Percentage of system memory to use (0-1.0)
	EnableMemoryMonitor   bool    // Enable memory pressure monitoring
	BackpressureThreshold float64 // Queue depth percentage to trigger backpressure
	TaskBufferSize        int     // Size of buffer for paused tasks

	// Adaptive rate limiting
	EnableAdaptiveRateLimit bool    // Enable adaptive rate limiting
	RateLimitMinRate        float64 // Minimum requests per second
	RateLimitMaxRate        float64 // Maximum requests per second
	RateLimitAdjustFactor   float64 // Rate adjustment factor

	// File filtering
	AllowedExtensions     []string // allowed file extensions
	EnableBinaryDetection bool     // enable binary file detection

	// Observability
	LogLevel    string
	MetricsPath string

	// Development
	Environment string
}

// Load creates a new Config by reading from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore errors if file doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		// Default values
		Port:                    getEnvOrDefault("PORT", "8080"),
		Host:                    getEnvOrDefault("HOST", "0.0.0.0"),
		GitHubBaseURL:           getEnvOrDefault("GITHUB_BASE_URL", "https://api.github.com"),
		MaxWorkers:              getEnvAsIntOrDefault("MAX_WORKERS", 50),
		APIRateLimitThreshold:   getEnvAsIntOrDefault("API_RATE_LIMIT_THRESHOLD", 100),
		FetchTimeoutMS:          getEnvAsIntOrDefault("FETCH_TIMEOUT_MS", 30000),
		RetryMaxAttempts:        getEnvAsIntOrDefault("RETRY_MAX_ATTEMPTS", 3),
		RetryBackoffBaseMS:      getEnvAsIntOrDefault("RETRY_BACKOFF_MS_BASE", 1000),
		MaxFileSize:             getEnvAsInt64OrDefault("MAX_FILE_SIZE", 10*1024*1024), // 10MB
		MaxConcurrentFetches:    getEnvAsIntOrDefault("MAX_CONCURRENT_FETCHES", 100),
		MemoryLimitPercent:      getEnvAsFloatOrDefault("MEMORY_LIMIT_PERCENT", 0.8),
		EnableMemoryMonitor:     getEnvAsBoolOrDefault("ENABLE_MEMORY_MONITOR", true),
		BackpressureThreshold:   getEnvAsFloatOrDefault("BACKPRESSURE_THRESHOLD", 0.8),
		TaskBufferSize:          getEnvAsIntOrDefault("TASK_BUFFER_SIZE", 1000),
		EnableAdaptiveRateLimit: getEnvAsBoolOrDefault("ENABLE_ADAPTIVE_RATE_LIMIT", true),
		RateLimitMinRate:        getEnvAsFloatOrDefault("RATE_LIMIT_MIN_RATE", 1.0),
		RateLimitMaxRate:        getEnvAsFloatOrDefault("RATE_LIMIT_MAX_RATE", 50.0),
		RateLimitAdjustFactor:   getEnvAsFloatOrDefault("RATE_LIMIT_ADJUST_FACTOR", 0.1),
		LogLevel:                getEnvOrDefault("LOG_LEVEL", "info"),
		MetricsPath:             getEnvOrDefault("METRICS_PATH", "/metrics"),
		Environment:             getEnvOrDefault("ENVIRONMENT", "development"),
		EnableBinaryDetection:   getEnvAsBoolOrDefault("ENABLE_BINARY_DETECTION", true),
	}

	// Load allowed extensions
	allowedExtensionsStr := getEnvOrDefault("ALLOWED_EXTENSIONS",
		".go,.js,.ts,.jsx,.tsx,.py,.java,.cpp,.c,.h,.hpp,.cs,.rb,.php,.rs,.swift,.kt,.scala,.sh,.bash,.zsh,.fish,.ps1,.bat,.cmd,.yaml,.yml,.json,.xml,.toml,.ini,.cfg,.conf,.md,.rst,.txt,.sql,.r,.m,.pl,.lua,.vim,.el,.clj,.hs,.fs,.ml,.pas,.ada,.cob,.f90,.pro,.asm,.s,.lisp,.scm,.tcl,.awk,.sed,.dockerfile,.makefile,.cmake,.gradle,.maven,.sbt,.cabal,.stack,.cargo,.gemfile,.requirements,.setup,.pipfile,.poetry,.pom,.build,.project,.solution")

	if allowedExtensionsStr != "" {
		extensions := strings.Split(allowedExtensionsStr, ",")
		for i, ext := range extensions {
			extensions[i] = strings.TrimSpace(strings.ToLower(ext))
			// Ensure extensions start with dot
			if !strings.HasPrefix(extensions[i], ".") {
				extensions[i] = "." + extensions[i]
			}
		}
		cfg.AllowedExtensions = extensions
	}

	// Required environment variables
	cfg.GitHubToken = os.Getenv("GITHUB_TOKEN")
	cfg.GitHubAppID = os.Getenv("GITHUB_APP_ID")
	cfg.GitHubAppKey = os.Getenv("GITHUB_APP_KEY")
	cfg.GitHubInstallID = os.Getenv("GITHUB_INSTALL_ID")

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check authentication - either PAT or GitHub App must be configured
	if c.GitHubToken == "" && (c.GitHubAppID == "" || c.GitHubAppKey == "" || c.GitHubInstallID == "") {
		return fmt.Errorf("either GITHUB_TOKEN or GitHub App credentials (GITHUB_APP_ID, GITHUB_APP_KEY, GITHUB_INSTALL_ID) must be provided")
	}

	// Validate worker pool settings
	if c.MaxWorkers <= 0 {
		return fmt.Errorf("MAX_WORKERS must be greater than 0")
	}

	if c.MaxWorkers > 1000 {
		return fmt.Errorf("MAX_WORKERS should not exceed 1000 for resource efficiency")
	}

	// Validate timeouts
	if c.FetchTimeoutMS <= 0 {
		return fmt.Errorf("FETCH_TIMEOUT_MS must be greater than 0")
	}

	// Validate retry settings
	if c.RetryMaxAttempts < 0 {
		return fmt.Errorf("RETRY_MAX_ATTEMPTS must be non-negative")
	}

	if c.RetryBackoffBaseMS <= 0 {
		return fmt.Errorf("RETRY_BACKOFF_MS_BASE must be greater than 0")
	}

	// Validate file size limits
	if c.MaxFileSize <= 0 {
		return fmt.Errorf("MAX_FILE_SIZE must be greater than 0")
	}

	// Validate concurrent fetches
	if c.MaxConcurrentFetches <= 0 {
		return fmt.Errorf("MAX_CONCURRENT_FETCHES must be greater than 0")
	}

	return nil
}

// GetFetchTimeout returns the fetch timeout as a duration
func (c *Config) GetFetchTimeout() time.Duration {
	return time.Duration(c.FetchTimeoutMS) * time.Millisecond
}

// GetRetryBackoffBase returns the retry backoff base as a duration
func (c *Config) GetRetryBackoffBase() time.Duration {
	return time.Duration(c.RetryBackoffBaseMS) * time.Millisecond
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// HasGitHubApp returns true if GitHub App credentials are configured
func (c *Config) HasGitHubApp() bool {
	return c.GitHubAppID != "" && c.GitHubAppKey != "" && c.GitHubInstallID != ""
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64OrDefault(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
