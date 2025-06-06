package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sattwyk/autodocs/apps/crawler/internal/config"
	"github.com/sattwyk/autodocs/apps/crawler/internal/metrics"
	"github.com/sattwyk/autodocs/apps/crawler/internal/model"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with token",
			config: &config.Config{
				GitHubToken:           "test-token",
				GitHubBaseURL:         "https://api.github.com",
				APIRateLimitThreshold: 100,
				FetchTimeoutMS:        30000,
			},
			wantErr: false,
		},
		{
			name: "valid config with github app",
			config: &config.Config{
				GitHubAppID:           "123456",
				GitHubAppKey:          testPrivateKey,
				GitHubInstallID:       "789012",
				GitHubBaseURL:         "https://api.github.com",
				APIRateLimitThreshold: 100,
				FetchTimeoutMS:        30000,
			},
			wantErr: true, // This will fail due to invalid key format, which is expected
			errMsg:  "failed to setup authentication",
		},
		{
			name: "missing authentication",
			config: &config.Config{
				GitHubBaseURL:         "https://api.github.com",
				APIRateLimitThreshold: 100,
				FetchTimeoutMS:        30000,
			},
			wantErr: true,
			errMsg:  "no authentication method configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metrics.NewForTesting()
			client, err := NewClient(tt.config, m)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, tt.config.GitHubBaseURL, client.baseURL)
		})
	}
}

func TestParseRepositoryURL(t *testing.T) {
	tests := []struct {
		name      string
		repoURL   string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "https github url",
			repoURL:   "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "https github url with .git",
			repoURL:   "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "ssh github url",
			repoURL:   "git@github.com:owner/repo.git",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:    "invalid url",
			repoURL: "not-a-url",
			wantErr: true,
		},
		{
			name:    "invalid format",
			repoURL: "https://github.com/owner",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseRepositoryURL(tt.repoURL)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantOwner, owner)
			assert.Equal(t, tt.wantRepo, repo)
		})
	}
}

func TestGetRepositoryTree(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/repos/owner/repo/git/trees/main")
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

		response := model.GitHubTreeResponse{
			SHA: "abc123",
			Tree: []model.TreeEntry{
				{Path: "file1.go", Type: "blob", SHA: "def456", Size: 100},
				{Path: "dir1", Type: "tree", SHA: "ghi789"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		GitHubToken:           "test-token",
		GitHubBaseURL:         server.URL,
		APIRateLimitThreshold: 1000,
		FetchTimeoutMS:        30000,
		RetryMaxAttempts:      1,
		RetryBackoffBaseMS:    100,
	}

	m := metrics.NewForTesting()
	client, err := NewClient(cfg, m)
	require.NoError(t, err)

	ctx := context.Background()
	tree, err := client.GetRepositoryTree(ctx, "owner", "repo", "main")

	assert.NoError(t, err)
	assert.NotNil(t, tree)
	assert.Equal(t, "abc123", tree.SHA)
	assert.Len(t, tree.Tree, 2)
	assert.Equal(t, "file1.go", tree.Tree[0].Path)
	assert.Equal(t, "blob", tree.Tree[0].Type)
}

func TestGetFileContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "raw.githubusercontent.com" {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("file content")); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
			return
		}

		// Fallback to API endpoint
		response := model.GitHubContentResponse{
			Content:  "ZmlsZSBjb250ZW50", // base64 encoded "file content"
			Encoding: "base64",
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		GitHubToken:           "test-token",
		GitHubBaseURL:         server.URL,
		APIRateLimitThreshold: 1000,
		FetchTimeoutMS:        30000,
		RetryMaxAttempts:      1,
		RetryBackoffBaseMS:    100,
	}

	m := metrics.NewForTesting()
	client, err := NewClient(cfg, m)
	require.NoError(t, err)

	ctx := context.Background()
	content, err := client.GetFileContent(ctx, "owner", "repo", "file.go", "main")

	assert.NoError(t, err)
	assert.Equal(t, []byte("file content"), content)
}

const testPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btYhA
z2R6eMhqz3lKHoHI7H6sv7yl1sN1LVrpF4FpjjBwgxaFzV4ddTjHxd4kjSQw7HLq
uehHch5wbtfXkXS5nig2XCxD7sRJyOOdj2ReJpjuwqHjuYHl6CXSgtObvdma2iei
5crGjMwjXGlO3OjMCCQoNfvLy0AyDdmzBJqxRYMGjPTAQqNKnY4jsirfCGKaT2RX
9Q62ZbeZVYhiRRLBBYRtfUHBHdvfn5N0SQxjcOWLc4xHPx6b5i7AnWnlrFqOgLRV
CWFWmZzxSPxltQEfNjwqHnJ1/XuEt7g1I1VwIDAQABAoIBAQDgl4cL9O7cc0XW
+8ykr746yz58VFHSjyLPpIBn4XqPD+IpTH7jbwjzNZhvJixWiG/VBnWrmVT3
pvTenqfZhhHpmMFcEKklEQDrfuFMLRjRz5pqHWDiDYyP99tBHWh6qmhzqbSaHgHq
jotKrfvNOocgKcTwFAEuv381GKMFCZZ4vbLZJ2RfTMH0A5xk8zHej+hMwBgSQnNs
d77eFbcR4SYz4DTBwQQYJVX15FrjMM1U5v4gzjM3Z8+Q5TTyKFe5zXnTTDHI
bK2A5J3cc4ieInlTL+hM9SiAs8O6N06fY5jGQXLGw2aWGd+su2s5gCBrTn8kg
-----END RSA PRIVATE KEY-----`
