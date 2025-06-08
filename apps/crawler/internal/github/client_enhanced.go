package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"golang.org/x/time/rate"

	"github.com/sattwyk/autodocs/apps/crawler/internal/config"
	"github.com/sattwyk/autodocs/apps/crawler/internal/metrics"
)

// StreamingClient extends the basic Client with streaming capabilities
type StreamingClient struct {
	*Client
	adaptiveLimiter *rate.Limiter
}

// NewStreamingClient creates a new streaming-capable GitHub client
func NewStreamingClient(cfg *config.Config, m *metrics.Metrics) (*StreamingClient, error) {
	baseClient, err := NewClient(cfg, m)
	if err != nil {
		return nil, err
	}

	initialRate := float64(cfg.APIRateLimitThreshold) / 3600.0

	return &StreamingClient{
		Client:          baseClient,
		adaptiveLimiter: rate.NewLimiter(rate.Limit(initialRate), 1),
	}, nil
}

// GetFileContentStream fetches file content as a stream
func (sc *StreamingClient) GetFileContentStream(ctx context.Context, owner, repo, path, ref string, handler func(io.Reader) error) error {
	if err := sc.adaptiveLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait failed: %w", err)
	}

	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, ref, path)

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	sc.setHeaders(req)

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	sc.updateAdaptiveRateLimit(resp)

	if resp.StatusCode == http.StatusOK {
		return handler(resp.Body)
	}

	return sc.getFileContentViaAPIStream(ctx, owner, repo, path, ref, handler)
}

// getFileContentViaAPIStream fetches content via API with streaming
func (sc *StreamingClient) getFileContentViaAPIStream(ctx context.Context, owner, repo, path, ref string, handler func(io.Reader) error) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", sc.baseURL, owner, repo, path, ref)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	sc.setHeaders(req)

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	sc.updateAdaptiveRateLimit(resp)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	decoder := base64.NewDecoder(base64.StdEncoding, resp.Body)
	return handler(decoder)
}

// updateAdaptiveRateLimit adjusts rate limit based on GitHub headers
func (sc *StreamingClient) updateAdaptiveRateLimit(resp *http.Response) {
	limitStr := resp.Header.Get("X-RateLimit-Limit")
	remainingStr := resp.Header.Get("X-RateLimit-Remaining")

	if limitStr != "" && remainingStr != "" {
		limit, _ := strconv.Atoi(limitStr)
		remaining, _ := strconv.Atoi(remainingStr)

		if limit > 0 {
			usagePercent := float64(limit-remaining) / float64(limit)

			currentLimit := sc.adaptiveLimiter.Limit()
			newLimit := currentLimit

			if usagePercent > 0.8 {
				newLimit = currentLimit * 0.5
			} else if usagePercent < 0.3 {
				newLimit = currentLimit * 1.2
			}

			// Enforce bounds
			if newLimit < 0.5 {
				newLimit = 0.5
			} else if newLimit > 50 {
				newLimit = 50
			}

			sc.adaptiveLimiter.SetLimit(rate.Limit(newLimit))
		}
	}

	sc.updateRateLimitMetrics(resp)
}
