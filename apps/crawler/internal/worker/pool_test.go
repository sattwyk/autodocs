package worker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sattwyk/autodocs/apps/crawler/internal/config"
	"github.com/sattwyk/autodocs/apps/crawler/internal/github"
	"github.com/sattwyk/autodocs/apps/crawler/internal/metrics"
	"github.com/sattwyk/autodocs/apps/crawler/internal/model"
)

func TestNewPool(t *testing.T) {
	cfg := &config.Config{
		MaxWorkers:           10,
		MaxConcurrentFetches: 50,
	}
	m := metrics.NewForTesting()
	ghClient := &github.Client{}

	pool := NewPool(cfg, m, ghClient)

	assert.NotNil(t, pool)
	assert.Equal(t, cfg, pool.config)
	assert.Equal(t, m, pool.metrics)
	assert.Equal(t, ghClient, pool.githubClient)
	assert.NotNil(t, pool.taskChan)
	assert.NotNil(t, pool.resultChan)
	assert.Equal(t, 0, pool.activeWorkers)
}

func TestPoolStartStop(t *testing.T) {
	cfg := &config.Config{
		MaxWorkers:           2,
		MaxConcurrentFetches: 10,
	}
	m := metrics.NewForTesting()
	ghClient := &github.Client{}

	pool := NewPool(cfg, m, ghClient)

	// Test start
	ctx := context.Background()
	err := pool.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, pool.IsRunning())
	assert.Equal(t, 2, pool.activeWorkers)

	// Test double start should fail
	err = pool.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stop
	err = pool.Stop()
	assert.NoError(t, err)
	assert.False(t, pool.IsRunning())
	assert.Equal(t, 0, pool.activeWorkers)
}

func TestSubmitTask(t *testing.T) {
	cfg := &config.Config{
		MaxWorkers:           1,
		MaxConcurrentFetches: 2,
	}
	m := metrics.NewForTesting()
	ghClient := &github.Client{}

	pool := NewPool(cfg, m, ghClient)

	task := model.WorkerTask{
		Path:  "test.go",
		SHA:   "abc123",
		Size:  100,
		Owner: "owner",
		Repo:  "repo",
		Ref:   "main",
	}

	// Submit task should work
	err := pool.SubmitTask(task)
	assert.NoError(t, err)

	// Submit another task should work
	err = pool.SubmitTask(task)
	assert.NoError(t, err)

	// Submit third task should fail (queue full)
	err = pool.SubmitTask(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue is full")
}

func TestGetQueueDepth(t *testing.T) {
	cfg := &config.Config{
		MaxWorkers:           1,
		MaxConcurrentFetches: 5,
	}
	m := metrics.NewForTesting()
	ghClient := &github.Client{}

	pool := NewPool(cfg, m, ghClient)

	assert.Equal(t, 0, pool.GetQueueDepth())

	task := model.WorkerTask{
		Path:  "test.go",
		SHA:   "abc123",
		Size:  100,
		Owner: "owner",
		Repo:  "repo",
		Ref:   "main",
	}

	err := pool.SubmitTask(task)
	assert.NoError(t, err)
	assert.Equal(t, 1, pool.GetQueueDepth())

	err = pool.SubmitTask(task)
	assert.NoError(t, err)
	assert.Equal(t, 2, pool.GetQueueDepth())
}

func TestShouldProcessFile(t *testing.T) {
	cfg := &config.Config{
		AllowedExtensions: []string{".go", ".js", ".py"},
	}
	m := metrics.NewForTesting()
	ghClient := &github.Client{}

	pool := NewPool(cfg, m, ghClient)

	tests := []struct {
		name       string
		path       string
		pathFilter []string
		expected   bool
	}{
		{
			name:     "allowed extension",
			path:     "main.go",
			expected: true,
		},
		{
			name:     "disallowed extension",
			path:     "main.txt",
			expected: false,
		},
		{
			name:       "path filter match",
			path:       "src/main.go",
			pathFilter: []string{"src/"},
			expected:   true,
		},
		{
			name:       "path filter no match",
			path:       "test/main.go",
			pathFilter: []string{"src/"},
			expected:   false,
		},
		{
			name:     "dockerfile special case",
			path:     "Dockerfile",
			expected: true,
		},
		{
			name:     "makefile special case",
			path:     "Makefile",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pool.shouldProcessFile(tt.path, tt.pathFilter)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAllowedFileType(t *testing.T) {
	cfg := &config.Config{
		AllowedExtensions: []string{".go", ".js", ".py"},
	}
	m := metrics.NewForTesting()
	ghClient := &github.Client{}

	pool := NewPool(cfg, m, ghClient)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "allowed go file",
			path:     "main.go",
			expected: true,
		},
		{
			name:     "allowed js file",
			path:     "script.js",
			expected: true,
		},
		{
			name:     "disallowed txt file",
			path:     "readme.txt",
			expected: false,
		},
		{
			name:     "dockerfile",
			path:     "Dockerfile",
			expected: true,
		},
		{
			name:     "makefile",
			path:     "Makefile",
			expected: true,
		},
		{
			name:     "package.json",
			path:     "package.json",
			expected: true,
		},
		{
			name:     "go.mod",
			path:     "go.mod",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pool.IsAllowedFileType(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAllowedFileTypeNoRestrictions(t *testing.T) {
	cfg := &config.Config{
		AllowedExtensions: []string{}, // No restrictions
	}
	m := metrics.NewForTesting()
	ghClient := &github.Client{}

	pool := NewPool(cfg, m, ghClient)

	// Should allow any file when no restrictions
	assert.True(t, pool.IsAllowedFileType("any.file"))
	assert.True(t, pool.IsAllowedFileType("no.extension"))
}

func TestIsBinaryContent(t *testing.T) {
	cfg := &config.Config{}
	m := metrics.NewForTesting()
	ghClient := &github.Client{}

	pool := NewPool(cfg, m, ghClient)

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "empty content",
			content:  []byte{},
			expected: false,
		},
		{
			name:     "text content",
			content:  []byte("Hello, World!"),
			expected: false,
		},
		{
			name:     "content with null byte",
			content:  []byte("Hello\x00World"),
			expected: true,
		},
		{
			name:     "content with many non-printable chars",
			content:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			expected: true,
		},
		{
			name:     "valid utf-8 with newlines",
			content:  []byte("package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}"),
			expected: false,
		},
		{
			name:     "content with tabs and newlines",
			content:  []byte("line1\tcolumn2\nline2\r\nline3"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pool.IsBinaryContent(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProcessTaskFileTooLarge(t *testing.T) {
	cfg := &config.Config{
		MaxFileSize: 100, // 100 bytes limit
	}
	m := metrics.NewForTesting()
	ghClient := &github.Client{}

	pool := NewPool(cfg, m, ghClient)

	task := model.WorkerTask{
		Path:  "large.go",
		SHA:   "abc123",
		Size:  200, // Exceeds limit
		Owner: "owner",
		Repo:  "repo",
		Ref:   "main",
	}

	result := pool.processTask(1, task)

	assert.Equal(t, "large.go", result.Path)
	assert.Equal(t, "abc123", result.SHA)
	assert.Equal(t, 200, result.Size)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "file size 200 exceeds limit 100")
}

func TestGetResultChannel(t *testing.T) {
	cfg := &config.Config{
		MaxWorkers:           1,
		MaxConcurrentFetches: 5,
	}
	m := metrics.NewForTesting()
	ghClient := &github.Client{}

	pool := NewPool(cfg, m, ghClient)

	resultChan := pool.GetResultChannel()
	assert.NotNil(t, resultChan)

	// Verify it's a read-only channel
	_, ok := interface{}(resultChan).(<-chan model.FileResult)
	assert.True(t, ok)
}
