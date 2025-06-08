package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationFullWorkflow tests the complete workflow
func TestIntegrationFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use random port for integration test
	os.Setenv("PORT", "0")
	t.Cleanup(func() {
		os.Unsetenv("PORT")
	})

	server := newTestServer(t)

	ctx := context.Background()
	err := server.Start(ctx)
	require.NoError(t, err)
	defer func() {
		if err := server.Stop(context.Background()); err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}()

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.handleHealth(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test root endpoint
	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	server.handleRoot(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestIntegrationErrorHandling tests error scenarios
func TestIntegrationErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := newTestServer(t)

	// Test invalid request
	reqBody := []byte(`{"repo_url": "invalid"}`)
	req := httptest.NewRequest("POST", "/invoke", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleInvoke(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
