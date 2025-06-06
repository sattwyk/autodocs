package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrawlRequestJSON(t *testing.T) {
	request := CrawlRequest{
		RepoURL:    "https://github.com/owner/repo",
		Ref:        "main",
		PathFilter: []string{"src/"},
	}

	// Test marshaling
	data, err := json.Marshal(request)
	require.NoError(t, err)

	// Test unmarshaling
	var unmarshaled CrawlRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, request, unmarshaled)
}

func TestCrawlResponseJSON(t *testing.T) {
	response := CrawlResponse{
		TotalFiles:     10,
		ProcessedFiles: 8,
		SkippedFiles:   2,
		RootTreeSHA:    "abc123",
		Duration:       "5.2s",
		RepoInfo: RepositoryInfo{
			Owner: "owner",
			Name:  "repo",
			Ref:   "main",
		},
	}

	// Test marshaling
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Test unmarshaling
	var unmarshaled CrawlResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, response.TotalFiles, unmarshaled.TotalFiles)
}

func TestTreeEntryJSON(t *testing.T) {
	entry := TreeEntry{
		Path: "src/main.go",
		Mode: "100644",
		Type: "blob",
		SHA:  "abc123def456",
		Size: 1024,
	}

	// Test marshaling
	data, err := json.Marshal(entry)
	require.NoError(t, err)

	// Test unmarshaling
	var unmarshaled TreeEntry
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, entry, unmarshaled)
}

func TestHealthResponseJSON(t *testing.T) {
	now := time.Now()
	healthResponse := HealthResponse{
		Status:    "healthy",
		Service:   "crawler",
		Timestamp: now,
		Version:   "1.0.0",
	}

	// Test marshaling
	data, err := json.Marshal(healthResponse)
	require.NoError(t, err)

	// Test unmarshaling
	var unmarshaled HealthResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, healthResponse.Status, unmarshaled.Status)
}
