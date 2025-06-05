package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all the Prometheus metrics for the crawler service
type Metrics struct {
	// Request metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Crawler metrics
	FilesRequestedTotal *prometheus.CounterVec
	FilesProcessedTotal *prometheus.CounterVec
	ErrorsTotal         *prometheus.CounterVec
	ConcurrencyInUse    prometheus.Gauge

	// GitHub API metrics
	GitHubAPICallsTotal  *prometheus.CounterVec
	GitHubRateLimitUsed  prometheus.Gauge
	GitHubRateLimitLimit prometheus.Gauge

	// Worker pool metrics
	WorkerPoolSize prometheus.Gauge
	QueueDepth     prometheus.Gauge
	TaskDuration   *prometheus.HistogramVec

	// Resource metrics
	FileSizeBytes *prometheus.HistogramVec
}

// New creates and registers all Prometheus metrics
func New() *Metrics {
	return &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "crawler_http_requests_total",
				Help: "Total number of HTTP requests received",
			},
			[]string{"method", "path", "status"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "crawler_http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),

		FilesRequestedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "crawler_files_requested_total",
				Help: "Total number of files requested for crawling",
			},
			[]string{"repo_owner", "repo_name"},
		),

		FilesProcessedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "crawler_files_processed_total",
				Help: "Total number of files successfully processed",
			},
			[]string{"repo_owner", "repo_name", "status"},
		),

		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "crawler_errors_total",
				Help: "Total number of errors encountered",
			},
			[]string{"type", "repo_owner", "repo_name"},
		),

		ConcurrencyInUse: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "crawler_concurrency_in_use",
				Help: "Number of concurrent operations currently in progress",
			},
		),

		GitHubAPICallsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "crawler_github_api_calls_total",
				Help: "Total number of GitHub API calls made",
			},
			[]string{"endpoint", "status"},
		),

		GitHubRateLimitUsed: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "crawler_github_rate_limit_used",
				Help: "Number of GitHub API rate limit requests used",
			},
		),

		GitHubRateLimitLimit: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "crawler_github_rate_limit_limit",
				Help: "GitHub API rate limit maximum",
			},
		),

		WorkerPoolSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "crawler_worker_pool_size",
				Help: "Current size of the worker pool",
			},
		),

		QueueDepth: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "crawler_queue_depth",
				Help: "Current depth of the task queue",
			},
		),

		TaskDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "crawler_task_duration_seconds",
				Help:    "Duration of individual tasks in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"task_type"},
		),

		FileSizeBytes: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "crawler_file_size_bytes",
				Help:    "Size of processed files in bytes",
				Buckets: []float64{1024, 10240, 102400, 1048576, 10485760, 104857600}, // 1KB to 100MB
			},
			[]string{"repo_owner", "repo_name"},
		),
	}
}

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(method, path, status string) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
}

// RecordHTTPDuration records the duration of an HTTP request
func (m *Metrics) RecordHTTPDuration(method, path string, duration float64) {
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// RecordFileRequested records a file request
func (m *Metrics) RecordFileRequested(repoOwner, repoName string) {
	m.FilesRequestedTotal.WithLabelValues(repoOwner, repoName).Inc()
}

// RecordFileProcessed records a processed file
func (m *Metrics) RecordFileProcessed(repoOwner, repoName, status string) {
	m.FilesProcessedTotal.WithLabelValues(repoOwner, repoName, status).Inc()
}

// RecordError records an error
func (m *Metrics) RecordError(errorType, repoOwner, repoName string) {
	m.ErrorsTotal.WithLabelValues(errorType, repoOwner, repoName).Inc()
}

// SetConcurrency sets the current concurrency level
func (m *Metrics) SetConcurrency(count float64) {
	m.ConcurrencyInUse.Set(count)
}

// RecordGitHubAPICall records a GitHub API call
func (m *Metrics) RecordGitHubAPICall(endpoint, status string) {
	m.GitHubAPICallsTotal.WithLabelValues(endpoint, status).Inc()
}

// UpdateGitHubRateLimit updates the GitHub rate limit metrics
func (m *Metrics) UpdateGitHubRateLimit(used, limit int) {
	m.GitHubRateLimitUsed.Set(float64(limit - used))
	m.GitHubRateLimitLimit.Set(float64(limit))
}

// SetWorkerPoolSize sets the worker pool size
func (m *Metrics) SetWorkerPoolSize(size float64) {
	m.WorkerPoolSize.Set(size)
}

// SetQueueDepth sets the queue depth
func (m *Metrics) SetQueueDepth(depth float64) {
	m.QueueDepth.Set(depth)
}

// RecordTaskDuration records the duration of a task
func (m *Metrics) RecordTaskDuration(taskType string, duration float64) {
	m.TaskDuration.WithLabelValues(taskType).Observe(duration)
}

// RecordFileSize records the size of a processed file
func (m *Metrics) RecordFileSize(repoOwner, repoName string, sizeBytes float64) {
	m.FileSizeBytes.WithLabelValues(repoOwner, repoName).Observe(sizeBytes)
}
