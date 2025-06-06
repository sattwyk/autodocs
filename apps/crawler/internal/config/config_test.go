package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantErr  bool
		errMsg   string
		validate func(*testing.T, *Config)
	}{
		{
			name: "valid config with github token",
			envVars: map[string]string{
				"GITHUB_TOKEN": "test-token",
				"PORT":         "9090",
				"HOST":         "127.0.0.1",
				"MAX_WORKERS":  "25",
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "test-token", cfg.GitHubToken)
				assert.Equal(t, "9090", cfg.Port)
				assert.Equal(t, "127.0.0.1", cfg.Host)
				assert.Equal(t, 25, cfg.MaxWorkers)
			},
		},
		{
			name: "valid config with github app",
			envVars: map[string]string{
				"GITHUB_APP_ID":     "123456",
				"GITHUB_APP_KEY":    "-----BEGIN RSA PRIVATE KEY-----\\ntest\\n-----END RSA PRIVATE KEY-----",
				"GITHUB_INSTALL_ID": "789012",
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "123456", cfg.GitHubAppID)
				assert.Equal(t, "789012", cfg.GitHubInstallID)
			},
		},
		{
			name: "custom allowed extensions",
			envVars: map[string]string{
				"GITHUB_TOKEN":       "test-token",
				"ALLOWED_EXTENSIONS": ".go,.js,.py",
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				expected := []string{".go", ".js", ".py"}
				assert.Equal(t, expected, cfg.AllowedExtensions)
			},
		},
		{
			name: "missing authentication",
			envVars: map[string]string{
				"PORT": "8080",
			},
			wantErr: true,
			errMsg:  "either GITHUB_TOKEN or GitHub App credentials",
		},
		{
			name: "invalid max workers - zero",
			envVars: map[string]string{
				"GITHUB_TOKEN": "test-token",
				"MAX_WORKERS":  "0",
			},
			wantErr: true,
			errMsg:  "MAX_WORKERS must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := Load()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func clearEnv() {
	envVars := []string{
		"PORT", "HOST", "GITHUB_BASE_URL", "GITHUB_TOKEN", "GITHUB_APP_ID",
		"GITHUB_APP_KEY", "GITHUB_INSTALL_ID", "MAX_WORKERS", "API_RATE_LIMIT_THRESHOLD",
		"FETCH_TIMEOUT_MS", "RETRY_MAX_ATTEMPTS", "RETRY_BACKOFF_MS_BASE",
		"MAX_FILE_SIZE", "MAX_CONCURRENT_FETCHES", "ALLOWED_EXTENSIONS",
		"ENABLE_BINARY_DETECTION", "LOG_LEVEL", "METRICS_PATH", "ENVIRONMENT",
	}

	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

func TestConfigDefaults(t *testing.T) {
	clearEnv()
	os.Setenv("GITHUB_TOKEN", "test-token")

	cfg, err := Load()
	require.NoError(t, err)

	// Test all default values
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "0.0.0.0", cfg.Host)
	assert.Equal(t, "https://api.github.com", cfg.GitHubBaseURL)
	assert.Equal(t, 50, cfg.MaxWorkers)
	assert.Equal(t, 100, cfg.APIRateLimitThreshold)
	assert.Equal(t, 30000, cfg.FetchTimeoutMS)
	assert.Equal(t, 3, cfg.RetryMaxAttempts)
	assert.Equal(t, 1000, cfg.RetryBackoffBaseMS)
	assert.Equal(t, int64(10*1024*1024), cfg.MaxFileSize)
	assert.Equal(t, 100, cfg.MaxConcurrentFetches)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "/metrics", cfg.MetricsPath)
	assert.Equal(t, "development", cfg.Environment)
	assert.True(t, cfg.EnableBinaryDetection)
	assert.NotEmpty(t, cfg.AllowedExtensions)
}

func TestConfigHelperMethods(t *testing.T) {
	clearEnv()
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("FETCH_TIMEOUT_MS", "5000")
	os.Setenv("RETRY_BACKOFF_MS_BASE", "2000")
	os.Setenv("ENVIRONMENT", "production")

	cfg, err := Load()
	require.NoError(t, err)

	// Test GetFetchTimeout
	expectedTimeout := 5 * time.Second
	assert.Equal(t, expectedTimeout, cfg.GetFetchTimeout())

	// Test GetRetryBackoffBase
	expectedBackoff := 2 * time.Second
	assert.Equal(t, expectedBackoff, cfg.GetRetryBackoffBase())

	// Test IsProduction
	assert.True(t, cfg.IsProduction())

	// Test HasGitHubApp
	assert.False(t, cfg.HasGitHubApp())
}

func TestHasGitHubApp(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name: "all github app credentials present",
			envVars: map[string]string{
				"GITHUB_APP_ID":     "123456",
				"GITHUB_APP_KEY":    "private-key",
				"GITHUB_INSTALL_ID": "789012",
			},
			expected: true,
		},
		{
			name: "missing app id",
			envVars: map[string]string{
				"GITHUB_APP_KEY":    "private-key",
				"GITHUB_INSTALL_ID": "789012",
			},
			expected: false,
		},
		{
			name:     "no github app credentials",
			envVars:  map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()

			// Set minimum required auth to make validation pass
			if !tt.expected {
				os.Setenv("GITHUB_TOKEN", "test-token")
			}

			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := Load()
			require.NoError(t, err)

			assert.Equal(t, tt.expected, cfg.HasGitHubApp())
		})
	}
}
