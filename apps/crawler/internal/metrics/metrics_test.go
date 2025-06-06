package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	m := NewForTesting()
	assert.NotNil(t, m)
	assert.NotNil(t, m.HTTPRequestsTotal)
	assert.NotNil(t, m.HTTPRequestDuration)
	assert.NotNil(t, m.FilesRequestedTotal)
	assert.NotNil(t, m.FilesProcessedTotal)
	assert.NotNil(t, m.ErrorsTotal)
	assert.NotNil(t, m.ConcurrencyInUse)
	assert.NotNil(t, m.GitHubAPICallsTotal)
	assert.NotNil(t, m.GitHubRateLimitUsed)
	assert.NotNil(t, m.GitHubRateLimitLimit)
	assert.NotNil(t, m.WorkerPoolSize)
	assert.NotNil(t, m.QueueDepth)
	assert.NotNil(t, m.TaskDuration)
	assert.NotNil(t, m.FileSizeBytes)
}

func TestRecordHTTPRequest(t *testing.T) {
	m := NewForTesting()

	m.RecordHTTPRequest("GET", "/health", "200")
	m.RecordHTTPRequest("GET", "/health", "200") // Second call with same labels
	m.RecordHTTPRequest("POST", "/invoke", "200")
	m.RecordHTTPRequest("GET", "/health", "500")

	// Test that counters were incremented
	assert.Equal(t, float64(2), testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/health", "200")))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("POST", "/invoke", "200")))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/health", "500")))
}

func TestRecordHTTPDuration(t *testing.T) {
	m := NewForTesting()

	// Test that the method doesn't panic
	m.RecordHTTPDuration("GET", "/health", 0.1)
	m.RecordHTTPDuration("GET", "/health", 0.2)
	m.RecordHTTPDuration("POST", "/invoke", 1.5)

	// Just verify the method works without error
	assert.NotNil(t, m.HTTPRequestDuration)
}

func TestRecordFileRequested(t *testing.T) {
	m := NewForTesting()

	m.RecordFileRequested("owner1", "repo1")
	m.RecordFileRequested("owner1", "repo1")
	m.RecordFileRequested("owner2", "repo2")

	assert.Equal(t, float64(2), testutil.ToFloat64(m.FilesRequestedTotal.WithLabelValues("owner1", "repo1")))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.FilesRequestedTotal.WithLabelValues("owner2", "repo2")))
}

func TestRecordFileProcessed(t *testing.T) {
	m := NewForTesting()

	m.RecordFileProcessed("owner1", "repo1", "success")
	m.RecordFileProcessed("owner1", "repo1", "failed")
	m.RecordFileProcessed("owner1", "repo1", "success")

	assert.Equal(t, float64(2), testutil.ToFloat64(m.FilesProcessedTotal.WithLabelValues("owner1", "repo1", "success")))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.FilesProcessedTotal.WithLabelValues("owner1", "repo1", "failed")))
}

func TestRecordError(t *testing.T) {
	m := NewForTesting()

	m.RecordError("api_error", "owner1", "repo1")
	m.RecordError("timeout", "owner1", "repo1")
	m.RecordError("api_error", "owner1", "repo1")

	assert.Equal(t, float64(2), testutil.ToFloat64(m.ErrorsTotal.WithLabelValues("api_error", "owner1", "repo1")))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.ErrorsTotal.WithLabelValues("timeout", "owner1", "repo1")))
}

func TestSetConcurrency(t *testing.T) {
	m := NewForTesting()

	m.SetConcurrency(5.0)
	assert.Equal(t, float64(5), testutil.ToFloat64(m.ConcurrencyInUse))

	m.SetConcurrency(10.0)
	assert.Equal(t, float64(10), testutil.ToFloat64(m.ConcurrencyInUse))
}

func TestRecordGitHubAPICall(t *testing.T) {
	m := NewForTesting()

	m.RecordGitHubAPICall("get_tree", "200")
	m.RecordGitHubAPICall("get_content", "200")
	m.RecordGitHubAPICall("get_tree", "404")

	assert.Equal(t, float64(1), testutil.ToFloat64(m.GitHubAPICallsTotal.WithLabelValues("get_tree", "200")))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.GitHubAPICallsTotal.WithLabelValues("get_content", "200")))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.GitHubAPICallsTotal.WithLabelValues("get_tree", "404")))
}

func TestUpdateGitHubRateLimit(t *testing.T) {
	m := NewForTesting()

	m.UpdateGitHubRateLimit(100, 5000)
	assert.Equal(t, float64(4900), testutil.ToFloat64(m.GitHubRateLimitUsed))
	assert.Equal(t, float64(5000), testutil.ToFloat64(m.GitHubRateLimitLimit))

	m.UpdateGitHubRateLimit(200, 5000)
	assert.Equal(t, float64(4800), testutil.ToFloat64(m.GitHubRateLimitUsed))
	assert.Equal(t, float64(5000), testutil.ToFloat64(m.GitHubRateLimitLimit))
}

func TestSetWorkerPoolSize(t *testing.T) {
	m := NewForTesting()

	m.SetWorkerPoolSize(50.0)
	assert.Equal(t, float64(50), testutil.ToFloat64(m.WorkerPoolSize))

	m.SetWorkerPoolSize(100.0)
	assert.Equal(t, float64(100), testutil.ToFloat64(m.WorkerPoolSize))
}

func TestSetQueueDepth(t *testing.T) {
	m := NewForTesting()

	m.SetQueueDepth(25.0)
	assert.Equal(t, float64(25), testutil.ToFloat64(m.QueueDepth))

	m.SetQueueDepth(0.0)
	assert.Equal(t, float64(0), testutil.ToFloat64(m.QueueDepth))
}

func TestRecordTaskDuration(t *testing.T) {
	m := NewForTesting()

	// Test that the method doesn't panic
	m.RecordTaskDuration("file_fetch", 0.5)
	m.RecordTaskDuration("file_fetch", 1.0)
	m.RecordTaskDuration("tree_fetch", 2.0)

	// Just verify the method works without error
	assert.NotNil(t, m.TaskDuration)
}

func TestRecordFileSize(t *testing.T) {
	m := NewForTesting()

	// Test that the method doesn't panic
	m.RecordFileSize("owner1", "repo1", 1024.0)
	m.RecordFileSize("owner1", "repo1", 2048.0)
	m.RecordFileSize("owner2", "repo2", 512.0)

	// Just verify the method works without error
	assert.NotNil(t, m.FileSizeBytes)
}
