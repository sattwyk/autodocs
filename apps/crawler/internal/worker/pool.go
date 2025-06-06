package worker

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sattwyk/autodocs/apps/crawler/internal/config"
	"github.com/sattwyk/autodocs/apps/crawler/internal/github"
	"github.com/sattwyk/autodocs/apps/crawler/internal/metrics"
	"github.com/sattwyk/autodocs/apps/crawler/internal/model"
)

// Pool represents a worker pool for processing crawl tasks
type Pool struct {
	config       *config.Config
	metrics      *metrics.Metrics
	githubClient *github.Client

	// Channels
	taskChan   chan model.WorkerTask
	resultChan chan model.FileResult

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// State
	activeWorkers int
	mu            sync.RWMutex
}

// NewPool creates a new worker pool
func NewPool(cfg *config.Config, m *metrics.Metrics, ghClient *github.Client) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &Pool{
		config:       cfg,
		metrics:      m,
		githubClient: ghClient,
		taskChan:     make(chan model.WorkerTask, cfg.MaxConcurrentFetches),
		resultChan:   make(chan model.FileResult, cfg.MaxConcurrentFetches),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Set initial metrics
	m.SetWorkerPoolSize(float64(cfg.MaxWorkers))

	return pool
}

// Start starts the worker pool
func (p *Pool) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.activeWorkers > 0 {
		return fmt.Errorf("worker pool is already running")
	}

	// Start workers
	for i := range p.config.MaxWorkers {
		p.wg.Add(1)
		go p.worker(i)
		p.activeWorkers++
	}

	log.Printf("Started %d workers", p.activeWorkers)
	p.metrics.SetWorkerPoolSize(float64(p.activeWorkers))

	return nil
}

// Stop stops the worker pool gracefully
func (p *Pool) Stop() error {
	p.cancel()

	// Close task channel
	close(p.taskChan)

	// Wait for all workers to finish
	p.wg.Wait()

	// Close result channel
	close(p.resultChan)

	p.mu.Lock()
	p.activeWorkers = 0
	p.mu.Unlock()

	p.metrics.SetWorkerPoolSize(0)
	log.Printf("Worker pool stopped")

	return nil
}

// SubmitTask submits a task to the worker pool
func (p *Pool) SubmitTask(task model.WorkerTask) error {
	select {
	case p.taskChan <- task:
		p.metrics.SetQueueDepth(float64(len(p.taskChan)))
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	default:
		return fmt.Errorf("task queue is full")
	}
}

// GetResultChannel returns the result channel
func (p *Pool) GetResultChannel() <-chan model.FileResult {
	return p.resultChan
}

// GetQueueDepth returns the current queue depth
func (p *Pool) GetQueueDepth() int {
	return len(p.taskChan)
}

// IsRunning returns true if the worker pool is running
func (p *Pool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeWorkers > 0
}

// worker is the main worker routine
func (p *Pool) worker(workerID int) {
	defer p.wg.Done()

	log.Printf("Worker %d started", workerID)

	for {
		select {
		case task, ok := <-p.taskChan:
			if !ok {
				log.Printf("Worker %d: task channel closed, shutting down", workerID)
				return
			}

			// Update queue depth metric
			p.metrics.SetQueueDepth(float64(len(p.taskChan)))

			// Process the task
			result := p.processTask(workerID, task)

			// Send result
			select {
			case p.resultChan <- result:
				// Result sent successfully
			case <-p.ctx.Done():
				log.Printf("Worker %d: context cancelled while sending result", workerID)
				return
			}

		case <-p.ctx.Done():
			log.Printf("Worker %d: context cancelled, shutting down", workerID)
			return
		}
	}
}

// processTask processes a single task
func (p *Pool) processTask(workerID int, task model.WorkerTask) model.FileResult {
	startTime := time.Now()

	result := model.FileResult{
		Path:      task.Path,
		SHA:       task.SHA,
		Size:      task.Size,
		FetchedAt: startTime,
	}

	// Record concurrency
	p.metrics.SetConcurrency(float64(len(p.taskChan)))

	// Use repository information from the task
	owner, repo, ref := task.Owner, task.Repo, task.Ref

	// Check file size limit
	if int64(task.Size) > p.config.MaxFileSize {
		result.Error = fmt.Errorf("file size %d exceeds limit %d", task.Size, p.config.MaxFileSize)
		p.metrics.RecordError("file_too_large", owner, repo)
		return result
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(p.ctx, p.config.GetFetchTimeout())
	defer cancel()

	// Fetch file content using the correct ref
	content, err := p.githubClient.GetFileContent(ctx, owner, repo, task.Path, ref)
	if err != nil {
		result.Error = err
		p.metrics.RecordError("fetch_failed", owner, repo)
		p.metrics.RecordFileProcessed(owner, repo, "failed")
		log.Printf("Worker %d: failed to fetch %s: %v", workerID, task.Path, err)
		return result
	}

	// Binary detection
	if p.config.EnableBinaryDetection && p.IsBinaryContent(content) {
		result.Error = fmt.Errorf("skipping binary file")
		p.metrics.RecordError("binary_file_skipped", owner, repo)
		p.metrics.RecordFileProcessed(owner, repo, "skipped_binary")
		log.Printf("Worker %d: skipped binary file %s", workerID, task.Path)
		return result
	}

	// UTF-8 validation
	if !utf8.Valid(content) {
		result.Error = fmt.Errorf("file content is not valid UTF-8")
		p.metrics.RecordError("invalid_utf8", owner, repo)
		p.metrics.RecordFileProcessed(owner, repo, "skipped_invalid_encoding")
		log.Printf("Worker %d: skipped non-UTF-8 file %s", workerID, task.Path)
		return result
	}

	result.Content = content
	result.Size = len(content)
	p.metrics.RecordFileProcessed(owner, repo, "success")
	p.metrics.RecordFileSize(owner, repo, float64(len(content)))
	log.Printf("Worker %d: successfully fetched %s (%d bytes)", workerID, task.Path, len(content))

	// Record task duration
	duration := time.Since(startTime).Seconds()
	p.metrics.RecordTaskDuration("file_fetch", duration)

	return result
}

// CrawlRepository crawls an entire repository
func (p *Pool) CrawlRepository(ctx context.Context, owner, repo, ref string, pathFilter []string) (*model.CrawlResponse, error) {
	startTime := time.Now()

	log.Printf("Starting crawl of %s/%s at ref %s", owner, repo, ref)

	// Get repository tree
	tree, err := p.githubClient.GetRepositoryTree(ctx, owner, repo, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository tree: %w", err)
	}

	log.Printf("Retrieved tree with %d entries", len(tree.Tree))

	// Filter files
	var filesToProcess []model.TreeEntry
	for _, entry := range tree.Tree {
		if entry.Type == "blob" && p.shouldProcessFile(entry.Path, pathFilter) {
			filesToProcess = append(filesToProcess, entry)
		}
	}

	log.Printf("Processing %d files after filtering", len(filesToProcess))

	// Submit tasks with repository context
	for _, file := range filesToProcess {
		task := model.WorkerTask{
			Path:  file.Path,
			SHA:   file.SHA,
			Size:  file.Size,
			Owner: owner, // Pass repository owner
			Repo:  repo,  // Pass repository name
			Ref:   ref,   // Pass the correct ref
		}

		if err := p.SubmitTask(task); err != nil {
			log.Printf("Failed to submit task for %s: %v", file.Path, err)
			continue
		}

		p.metrics.RecordFileRequested(owner, repo)
	}

	// Collect results
	var (
		processedFiles = 0
		skippedFiles   = 0
		errors         []model.CrawlError
		mu             sync.Mutex
		fileResults    []model.FileResult
	)

	// Create a done channel to signal completion
	done := make(chan struct{})
	go func() {
		defer close(done)

		for range filesToProcess {
			select {
			case result := <-p.resultChan:
				mu.Lock()
				if result.Error != nil {
					skippedFiles++
					errors = append(errors, model.CrawlError{
						FilePath: result.Path,
						Error:    result.Error.Error(),
						Type:     "fetch_error",
					})
				} else {
					processedFiles++
				}
				fileResults = append(fileResults, result)
				mu.Unlock()

			case <-ctx.Done():
				log.Printf("Context cancelled while waiting for results")
				return
			}
		}
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		log.Printf("Crawl completed: %d processed, %d skipped, %d errors",
			processedFiles, skippedFiles, len(errors))
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Build response
	response := &model.CrawlResponse{
		TotalFiles:     len(filesToProcess),
		ProcessedFiles: processedFiles,
		SkippedFiles:   skippedFiles,
		Errors:         errors,
		RootTreeSHA:    tree.SHA,
		Duration:       time.Since(startTime).String(),
		RepoInfo: model.RepositoryInfo{
			Owner: owner,
			Name:  repo,
			Ref:   ref,
		},
		Files: fileResults,
	}

	return response, nil
}

// shouldProcessFile determines if a file should be processed based on path filters and file extensions
func (p *Pool) shouldProcessFile(path string, pathFilter []string) bool {
	// Check path filters first (existing logic)
	if len(pathFilter) > 0 {
		matchesFilter := false
		for _, filter := range pathFilter {
			if len(path) >= len(filter) && path[:len(filter)] == filter {
				matchesFilter = true
				break
			}
		}
		if !matchesFilter {
			return false
		}
	}

	// Check file extension
	if len(p.config.AllowedExtensions) > 0 {
		return p.IsAllowedFileType(path)
	}

	return true
}

// IsAllowedFileType checks if the file extension is in the allowed list
func (p *Pool) IsAllowedFileType(path string) bool {
	if len(p.config.AllowedExtensions) == 0 {
		return true // No restrictions if no extensions configured
	}

	ext := strings.ToLower(filepath.Ext(path))
	filename := strings.ToLower(filepath.Base(path))

	// Check extension
	for _, allowedExt := range p.config.AllowedExtensions {
		if ext == allowedExt {
			return true
		}
	}

	// Check special filenames (dockerfile, makefile, etc.)
	specialFiles := []string{
		"dockerfile", "makefile", "rakefile", "gemfile", "guardfile",
		"capfile", "berksfile", "cheffile", "vagrantfile", "fastfile",
		"appfile", "deliverfile", "matchfile", "gymfile", "scanfile",
		"snapfile", "podfile", "cartfile", "brewfile", "requirements.txt",
		"setup.py", "setup.cfg", "pyproject.toml", "pipfile", "poetry.lock",
		"package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
		"composer.json", "composer.lock", "go.mod", "go.sum", "cargo.toml",
		"cargo.lock", "build.gradle", "pom.xml", "build.sbt", "mix.exs",
		"deps.edn", "project.clj", "stack.yaml", "cabal.project",
	}

	for _, special := range specialFiles {
		if filename == special {
			return true
		}
	}

	return false
}

// IsBinaryContent detects if content is binary by checking for null bytes and non-printable characters
func (p *Pool) IsBinaryContent(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check first 8KB for binary indicators
	checkSize := 8192
	if len(content) < checkSize {
		checkSize = len(content)
	}

	sample := content[:checkSize]

	// Check for null bytes (strong binary indicator)
	for _, b := range sample {
		if b == 0 {
			return true
		}
	}

	// Check ratio of non-printable characters
	nonPrintable := 0
	for _, b := range sample {
		// Consider bytes outside ASCII printable range (32-126) and common whitespace (9, 10, 13)
		if b < 9 || (b > 13 && b < 32) || b > 126 {
			nonPrintable++
		}
	}

	// If more than 30% non-printable, consider it binary
	return float64(nonPrintable)/float64(len(sample)) > 0.30
}
