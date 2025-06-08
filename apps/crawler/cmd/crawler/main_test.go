package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sattwyk/autodocs/apps/crawler/internal/config"
	"github.com/sattwyk/autodocs/apps/crawler/internal/github"
	"github.com/sattwyk/autodocs/apps/crawler/internal/metrics"
	"github.com/sattwyk/autodocs/apps/crawler/internal/model"
	"github.com/sattwyk/autodocs/apps/crawler/internal/worker"
)

// newTestServer creates a server instance for testing
func newTestServer(t *testing.T) *Server {
	// Set required environment variables
	os.Setenv("GITHUB_TOKEN", "test-token")
	t.Cleanup(func() {
		os.Unsetenv("GITHUB_TOKEN")
	})

	// Load configuration
	cfg, err := config.Load()
	require.NoError(t, err)

	// Initialize metrics with testing registry
	m := metrics.NewForTesting()

	// Initialize GitHub client
	ghClient, err := github.NewClient(cfg, m)
	require.NoError(t, err)

	// Initialize worker pool
	pool := worker.NewPool(cfg, m, ghClient)

	server := &Server{
		config:       cfg,
		metrics:      m,
		githubClient: ghClient,
		workerPool:   pool,
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	server.setupRoutes(mux)

	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler:      server.loggingMiddleware(server.metricsMiddleware(mux)),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server
}

func TestNewServer(t *testing.T) {
	server := newTestServer(t)
	assert.NotNil(t, server)
	assert.NotNil(t, server.config)
	assert.NotNil(t, server.metrics)
	assert.NotNil(t, server.githubClient)
	assert.NotNil(t, server.workerPool)
	assert.NotNil(t, server.httpServer)
}

func TestNewServerMissingAuth(t *testing.T) {
	// Clear any existing auth environment variables
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("GITHUB_APP_ID")
	os.Unsetenv("GITHUB_APP_KEY")
	os.Unsetenv("GITHUB_INSTALL_ID")

	server, err := NewServer()
	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "config validation failed")
}

func TestHandleRoot(t *testing.T) {
	server := newTestServer(t)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   map[string]string
	}{
		{
			name:           "GET request",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedBody: map[string]string{
				"service": "crawler",
				"status":  "running",
				"version": "1.0.0",
			},
		},
		{
			name:           "POST request (not allowed)",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "PUT request (not allowed)",
			method:         "PUT",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			w := httptest.NewRecorder()

			server.handleRoot(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestHandleHealth(t *testing.T) {
	server := newTestServer(t)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		setupWorkers   bool
	}{
		{
			name:           "GET request with workers running",
			method:         "GET",
			expectedStatus: http.StatusOK,
			setupWorkers:   true,
		},
		{
			name:           "GET request with workers not running",
			method:         "GET",
			expectedStatus: http.StatusServiceUnavailable,
			setupWorkers:   false,
		},
		{
			name:           "POST request (not allowed)",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
			setupWorkers:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupWorkers {
				ctx := context.Background()
				err := server.workerPool.Start(ctx)
				require.NoError(t, err)
				defer func() {
					if err := server.workerPool.Stop(); err != nil {
						t.Errorf("Failed to stop worker pool: %v", err)
					}
				}()
			}

			req := httptest.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			server.handleHealth(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.method == "GET" {
				var response model.HealthResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "crawler", response.Service)
				assert.Equal(t, "1.0.0", response.Version)
				assert.NotZero(t, response.Timestamp)

				if tt.setupWorkers {
					assert.Equal(t, "healthy", response.Status)
				} else {
					assert.Equal(t, "unhealthy", response.Status)
				}
			}
		})
	}
}

func TestHandleInvoke(t *testing.T) {
	server := newTestServer(t)

	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "GET request (not allowed)",
			method:         "GET",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid JSON body",
			method:         "POST",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:   "missing repo_url",
			method: "POST",
			body: map[string]interface{}{
				"ref": "main",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "repo_url is required",
		},
		{
			name:   "invalid repo URL",
			method: "POST",
			body: map[string]interface{}{
				"repo_url": "not-a-valid-url",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid repository URL",
		},
		{
			name:   "valid request with invalid repo (will fail during crawl)",
			method: "POST",
			body: map[string]interface{}{
				"repo_url": "https://github.com/nonexistent/repo",
				"ref":      "main",
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					reqBody = []byte(str)
				} else {
					reqBody, _ = json.Marshal(tt.body)
				}
			}

			req := httptest.NewRequest(tt.method, "/invoke", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleInvoke(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				responseBody := w.Body.String()
				assert.Contains(t, responseBody, tt.expectedError)
			}

			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusInternalServerError {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestLoggingMiddleware(t *testing.T) {
	server := newTestServer(t)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("test response")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	})

	// Wrap with logging middleware
	wrappedHandler := server.loggingMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// This should not panic and should call the wrapped handler
	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestMetricsMiddleware(t *testing.T) {
	server := newTestServer(t)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("test response")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	})

	// Wrap with metrics middleware
	wrappedHandler := server.metricsMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// This should not panic and should call the wrapped handler
	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestResponseWrapper(t *testing.T) {
	w := httptest.NewRecorder()
	wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

	// Test default status code
	assert.Equal(t, http.StatusOK, wrapper.statusCode)

	// Test WriteHeader
	wrapper.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, wrapper.statusCode)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test Write
	_, err := wrapper.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, "test", w.Body.String())
}

func TestServerStartStop(t *testing.T) {
	// Set random port for testing
	os.Setenv("PORT", "0")
	t.Cleanup(func() {
		os.Unsetenv("PORT")
	})

	server := newTestServer(t)

	ctx := context.Background()

	// Test start
	err := server.Start(ctx)
	assert.NoError(t, err)

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(stopCtx)
	assert.NoError(t, err)
}

func TestSetupRoutes(t *testing.T) {
	server := newTestServer(t)

	// Start worker pool for health check to pass
	ctx := context.Background()
	err := server.workerPool.Start(ctx)
	require.NoError(t, err)
	defer func() {
		if err := server.workerPool.Stop(); err != nil {
			t.Errorf("Failed to stop worker pool: %v", err)
		}
	}()

	mux := http.NewServeMux()
	server.setupRoutes(mux)

	// Test that routes are properly set up by making requests
	tests := []struct {
		path           string
		expectedStatus int
	}{
		{"/", http.StatusOK},
		{"/health", http.StatusOK},
		{"/metrics", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
