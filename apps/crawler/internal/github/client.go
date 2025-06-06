package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"

	"github.com/sattwyk/autodocs/apps/crawler/internal/config"
	"github.com/sattwyk/autodocs/apps/crawler/internal/metrics"
	"github.com/sattwyk/autodocs/apps/crawler/internal/model"
)

// Client represents a GitHub API client
type Client struct {
	baseURL     string
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	metrics     *metrics.Metrics
	config      *config.Config
	token       string
}

// NewClient creates a new GitHub API client
func NewClient(cfg *config.Config, m *metrics.Metrics) (*Client, error) {
	client := &Client{
		baseURL:     cfg.GitHubBaseURL,
		httpClient:  &http.Client{Timeout: cfg.GetFetchTimeout()},
		rateLimiter: rate.NewLimiter(rate.Limit(cfg.APIRateLimitThreshold), cfg.APIRateLimitThreshold),
		metrics:     m,
		config:      cfg,
	}

	// Set up authentication
	if err := client.setupAuth(); err != nil {
		return nil, fmt.Errorf("failed to setup authentication: %w", err)
	}

	return client, nil
}

// setupAuth configures authentication for the GitHub client
func (c *Client) setupAuth() error {
	if c.config.GitHubToken != "" {
		// Use Personal Access Token
		c.token = c.config.GitHubToken
		return nil
	}

	if c.config.HasGitHubApp() {
		// Use GitHub App authentication
		token, err := c.generateInstallationToken()
		if err != nil {
			return fmt.Errorf("failed to generate installation token: %w", err)
		}
		c.token = token
		return nil
	}

	return fmt.Errorf("no authentication method configured")
}

// generateInstallationToken generates a GitHub App installation token
func (c *Client) generateInstallationToken() (string, error) {
	// Generate JWT for GitHub App
	jwtToken, err := c.generateAppJWT()
	if err != nil {
		return "", fmt.Errorf("failed to generate app JWT: %w", err)
	}

	// Get installation token
	url := fmt.Sprintf("%s/app/installations/%s/access_tokens", c.baseURL, c.config.GitHubInstallID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get installation token: %s", string(body))
	}

	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.Token, nil
}

// generateAppJWT generates a JWT for GitHub App authentication
func (c *Client) generateAppJWT() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(),
		"iss": c.config.GitHubAppID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Parse the private key
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(c.config.GitHubAppKey))
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	return token.SignedString(key)
}

// GetRepositoryTree fetches the Git tree for a repository
func (c *Client) GetRepositoryTree(ctx context.Context, owner, repo, ref string) (*model.GitHubTreeResponse, error) {
	// Wait for rate limit
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/git/trees/%s?recursive=1", c.baseURL, owner, repo, ref)

	var treeResp *model.GitHubTreeResponse
	err := c.makeRequestWithRetry(ctx, "GET", url, nil, func(resp *http.Response) error {
		c.metrics.RecordGitHubAPICall("get_tree", strconv.Itoa(resp.StatusCode))

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		return json.NewDecoder(resp.Body).Decode(&treeResp)
	})

	if err != nil {
		c.metrics.RecordError("api_error", owner, repo)
		return nil, fmt.Errorf("failed to get repository tree: %w", err)
	}

	return treeResp, nil
}

// GetFileContent fetches the content of a specific file
func (c *Client) GetFileContent(ctx context.Context, owner, repo, path, ref string) ([]byte, error) {
	// Wait for rate limit
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Try raw content first (more efficient)
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, ref, path)

	var content []byte
	err := c.makeRequestWithRetry(ctx, "GET", rawURL, nil, func(resp *http.Response) error {
		c.metrics.RecordGitHubAPICall("get_raw_content", strconv.Itoa(resp.StatusCode))

		if resp.StatusCode == http.StatusOK {
			var err error
			content, err = io.ReadAll(resp.Body)
			return err
		}

		// If raw content fails, try API endpoint
		return c.getFileContentViaAPI(ctx, owner, repo, path, ref, &content)
	})

	if err != nil {
		c.metrics.RecordError("api_error", owner, repo)
		return nil, fmt.Errorf("failed to get file content for %s: %w", path, err)
	}

	return content, nil
}

// getFileContentViaAPI fetches file content via the GitHub API
func (c *Client) getFileContentViaAPI(ctx context.Context, owner, repo, path, ref string, content *[]byte) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", c.baseURL, owner, repo, path, ref)

	return c.makeRequestWithRetry(ctx, "GET", url, nil, func(resp *http.Response) error {
		c.metrics.RecordGitHubAPICall("get_content", strconv.Itoa(resp.StatusCode))

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		var contentResp model.GitHubContentResponse
		if err := json.NewDecoder(resp.Body).Decode(&contentResp); err != nil {
			return fmt.Errorf("failed to decode content response: %w", err)
		}

		if contentResp.Encoding == "base64" {
			decoded, err := base64.StdEncoding.DecodeString(contentResp.Content)
			if err != nil {
				return fmt.Errorf("failed to decode base64 content: %w", err)
			}
			*content = decoded
		} else {
			*content = []byte(contentResp.Content)
		}

		return nil
	})
}

// makeRequestWithRetry makes an HTTP request with retry logic
func (c *Client) makeRequestWithRetry(ctx context.Context, method, url string, body io.Reader, handler func(*http.Response) error) error {
	var lastErr error
	backoff := c.config.GetRetryBackoffBase()

	for attempt := 0; attempt <= c.config.RetryMaxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2 // Exponential backoff
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		c.setHeaders(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Update rate limit metrics
		c.updateRateLimitMetrics(resp)

		err = handler(resp)
		resp.Body.Close()

		if err == nil {
			return nil
		}

		// Check if we should retry
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			lastErr = err
			continue
		}

		// Don't retry for client errors
		return err
	}

	return fmt.Errorf("max retries exceeded, last error: %w", lastErr)
}

// setHeaders sets the required headers for GitHub API requests
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "autodocs-crawler/1.0")
}

// updateRateLimitMetrics updates rate limit metrics from response headers
func (c *Client) updateRateLimitMetrics(resp *http.Response) {
	if limitStr := resp.Header.Get("X-RateLimit-Limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			if remainingStr := resp.Header.Get("X-RateLimit-Remaining"); remainingStr != "" {
				if remaining, err := strconv.Atoi(remainingStr); err == nil {
					c.metrics.UpdateGitHubRateLimit(limit-remaining, limit)
				}
			}
		}
	}
}

// ParseRepositoryURL parses a GitHub repository URL and extracts owner and repo name
func ParseRepositoryURL(repoURL string) (owner, repo string, err error) {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid repository URL: %w", err)
	}

	// Handle different URL formats
	path := strings.Trim(parsed.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository URL format, expected owner/repo")
	}

	return parts[0], parts[1], nil
}
