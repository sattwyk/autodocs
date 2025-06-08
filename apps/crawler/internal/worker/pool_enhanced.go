package worker

import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sattwyk/autodocs/apps/crawler/internal/config"
	"github.com/sattwyk/autodocs/apps/crawler/internal/github"
	"github.com/sattwyk/autodocs/apps/crawler/internal/metrics"
	"github.com/sattwyk/autodocs/apps/crawler/internal/model"
)

// EnhancedPool represents an improved worker pool with better handling for large repos
type EnhancedPool struct {
	*Pool // Embed the original pool

	// Memory management
	memoryLimit      int64 // in bytes
	currentMemoryUse int64 // atomic counter for memory usage
	memoryPressure   bool  // flag to indicate memory pressure

	// Adaptive rate limiting
	rateLimitWindow   *RateLimitWindow
	adaptiveRateLimit *AdaptiveRateLimiter

	// Task management
	pauseChan          chan struct{}      // Channel to pause/resume workers
	isPaused           atomic.Bool        // Atomic flag for pause state
	backpressureLimit  int                // Threshold for applying backpressure
	droppedTasksBuffer []model.WorkerTask // Buffer to hold tasks during pause
	bufferMu           sync.Mutex

	// Monitoring
	memoryMonitorStop chan struct{}
	memoryMonitorWg   sync.WaitGroup
}

// RateLimitWindow tracks API rate limit usage over time
type RateLimitWindow struct {
	mu         sync.RWMutex
	requests   []time.Time
	windowSize time.Duration
	limit      int
	remaining  int
	resetTime  time.Time
}

// AdaptiveRateLimiter adjusts rate based on GitHub's response headers
type AdaptiveRateLimiter struct {
	mu                sync.RWMutex
	currentRate       float64
	minRate           float64
	maxRate           float64
	adjustmentFactor  float64
	lastAdjustment    time.Time
	backoffMultiplier float64
}

// NewEnhancedPool creates a new enhanced worker pool
func NewEnhancedPool(cfg *config.Config, m *metrics.Metrics, ghClient *github.Client) *EnhancedPool {
	basePool := NewPool(cfg, m, ghClient)

	// Calculate memory limit (use 80% of available memory)
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryLimit := int64(float64(memStats.Sys) * 0.8)

	enhancedPool := &EnhancedPool{
		Pool:               basePool,
		memoryLimit:        memoryLimit,
		pauseChan:          make(chan struct{}),
		backpressureLimit:  cfg.MaxConcurrentFetches * 2,
		droppedTasksBuffer: make([]model.WorkerTask, 0, 1000),
		memoryMonitorStop:  make(chan struct{}),
		rateLimitWindow: &RateLimitWindow{
			requests:   make([]time.Time, 0, cfg.APIRateLimitThreshold),
			windowSize: time.Hour,
			limit:      cfg.APIRateLimitThreshold,
			remaining:  cfg.APIRateLimitThreshold,
		},
		adaptiveRateLimit: &AdaptiveRateLimiter{
			currentRate:       float64(cfg.APIRateLimitThreshold) / 3600.0, // per second
			minRate:           1.0,                                         // minimum 1 request per second
			maxRate:           100.0,                                       // maximum 100 requests per second
			adjustmentFactor:  0.1,
			backoffMultiplier: 0.5,
		},
	}

	return enhancedPool
}

// Start starts the enhanced worker pool with memory monitoring
func (ep *EnhancedPool) Start(ctx context.Context) error {
	if err := ep.Pool.Start(ctx); err != nil {
		return err
	}

	// Start memory monitor
	ep.memoryMonitorWg.Add(1)
	go ep.monitorMemory()

	return nil
}

// Stop stops the enhanced worker pool and all monitoring
func (ep *EnhancedPool) Stop() error {
	// Stop memory monitor
	close(ep.memoryMonitorStop)
	ep.memoryMonitorWg.Wait()

	// Flush any buffered tasks
	ep.flushBufferedTasks()

	return ep.Pool.Stop()
}

// SubmitTaskWithBackpressure submits a task with backpressure handling
func (ep *EnhancedPool) SubmitTaskWithBackpressure(ctx context.Context, task model.WorkerTask) error {
	// Check if we're paused
	if ep.isPaused.Load() {
		// Buffer the task instead of dropping it
		ep.bufferMu.Lock()
		ep.droppedTasksBuffer = append(ep.droppedTasksBuffer, task)
		ep.bufferMu.Unlock()

		ep.metrics.RecordError("task_buffered", task.Owner, task.Repo)

		// Wait for unpause or context cancellation
		select {
		case <-ep.pauseChan:
			// Resumed, try to submit again
			return ep.SubmitTaskWithBackpressure(ctx, task)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Check memory pressure
	if ep.memoryPressure {
		// Apply exponential backoff when under memory pressure
		backoffDuration := time.Duration(float64(time.Second) * (1.0 + float64(ep.GetQueueDepth())/float64(ep.config.MaxConcurrentFetches)))
		select {
		case <-time.After(backoffDuration):
			// Continue after backoff
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Check queue depth for backpressure
	queueDepth := ep.GetQueueDepth()
	if queueDepth >= ep.backpressureLimit {
		// Pause submission until queue drains
		ep.pauseWorkers()

		// Wait for queue to drain below threshold
		for ep.GetQueueDepth() >= ep.backpressureLimit/2 {
			select {
			case <-time.After(100 * time.Millisecond):
				// Check again
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		ep.resumeWorkers()
	}

	// Try to submit with blocking behavior instead of failing
	for {
		select {
		case ep.taskChan <- task:
			ep.metrics.SetQueueDepth(float64(len(ep.taskChan)))
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Retry submission
			continue
		}
	}
}

// monitorMemory monitors memory usage and applies pressure when needed
func (ep *EnhancedPool) monitorMemory() {
	defer ep.memoryMonitorWg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			currentUse := int64(memStats.Alloc)
			atomic.StoreInt64(&ep.currentMemoryUse, currentUse)

			// Update metrics
			// Memory usage metrics could be added to the metrics package if needed

			// Check if we're approaching memory limit
			if float64(currentUse) > float64(ep.memoryLimit)*0.9 {
				if !ep.memoryPressure {
					ep.memoryPressure = true
					ep.pauseWorkers()

					// Force garbage collection
					runtime.GC()

					// Log warning
					log.Printf("Memory pressure detected: %d MB / %d MB", currentUse/1024/1024, ep.memoryLimit/1024/1024)
				}
			} else if float64(currentUse) < float64(ep.memoryLimit)*0.7 {
				if ep.memoryPressure {
					ep.memoryPressure = false
					ep.resumeWorkers()
					log.Printf("Memory pressure relieved: %d MB / %d MB", currentUse/1024/1024, ep.memoryLimit/1024/1024)
				}
			}

		case <-ep.memoryMonitorStop:
			return
		}
	}
}

// pauseWorkers pauses all workers temporarily
func (ep *EnhancedPool) pauseWorkers() {
	if ep.isPaused.CompareAndSwap(false, true) {
		log.Printf("Pausing workers due to resource constraints")
		// Event metrics could be added to the metrics package if needed
	}
}

// resumeWorkers resumes paused workers
func (ep *EnhancedPool) resumeWorkers() {
	if ep.isPaused.CompareAndSwap(true, false) {
		close(ep.pauseChan)
		ep.pauseChan = make(chan struct{}) // Reset for next pause

		// Process buffered tasks
		go ep.flushBufferedTasks()

		log.Printf("Resuming workers")
		// Event metrics could be added to the metrics package if needed
	}
}

// flushBufferedTasks processes tasks that were buffered during pause
func (ep *EnhancedPool) flushBufferedTasks() {
	ep.bufferMu.Lock()
	tasks := make([]model.WorkerTask, len(ep.droppedTasksBuffer))
	copy(tasks, ep.droppedTasksBuffer)
	ep.droppedTasksBuffer = ep.droppedTasksBuffer[:0]
	ep.bufferMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for _, task := range tasks {
		if err := ep.SubmitTaskWithBackpressure(ctx, task); err != nil {
			log.Printf("Failed to resubmit buffered task %s: %v", task.Path, err)
		}
	}
}

// UpdateRateLimitFromHeaders updates rate limiting based on GitHub response headers
func (ep *EnhancedPool) UpdateRateLimitFromHeaders(remaining, limit int, resetTime time.Time) {
	ep.rateLimitWindow.mu.Lock()
	defer ep.rateLimitWindow.mu.Unlock()

	ep.rateLimitWindow.remaining = remaining
	ep.rateLimitWindow.limit = limit
	ep.rateLimitWindow.resetTime = resetTime

	// Update adaptive rate limiter
	ep.adaptiveRateLimit.mu.Lock()
	defer ep.adaptiveRateLimit.mu.Unlock()

	// Calculate usage percentage
	usagePercent := float64(limit-remaining) / float64(limit)

	// Adjust rate based on usage
	if usagePercent > 0.8 {
		// Slow down when approaching limit
		ep.adaptiveRateLimit.currentRate *= ep.adaptiveRateLimit.backoffMultiplier
	} else if usagePercent < 0.5 && time.Since(ep.adaptiveRateLimit.lastAdjustment) > 5*time.Minute {
		// Speed up if we have plenty of headroom
		ep.adaptiveRateLimit.currentRate *= (1.0 + ep.adaptiveRateLimit.adjustmentFactor)
	}

	// Enforce min/max bounds
	if ep.adaptiveRateLimit.currentRate < ep.adaptiveRateLimit.minRate {
		ep.adaptiveRateLimit.currentRate = ep.adaptiveRateLimit.minRate
	} else if ep.adaptiveRateLimit.currentRate > ep.adaptiveRateLimit.maxRate {
		ep.adaptiveRateLimit.currentRate = ep.adaptiveRateLimit.maxRate
	}

	ep.adaptiveRateLimit.lastAdjustment = time.Now()

	// Update metrics
	// Adaptive rate limit metrics could be added to the metrics package if needed
}

// GetCurrentRateLimit returns the current adaptive rate limit
func (ep *EnhancedPool) GetCurrentRateLimit() float64 {
	ep.adaptiveRateLimit.mu.RLock()
	defer ep.adaptiveRateLimit.mu.RUnlock()
	return ep.adaptiveRateLimit.currentRate
}

// GetMemoryUsage returns current memory usage information
func (ep *EnhancedPool) GetMemoryUsage() (current, limit int64, pressure bool) {
	return atomic.LoadInt64(&ep.currentMemoryUse), ep.memoryLimit, ep.memoryPressure
}
