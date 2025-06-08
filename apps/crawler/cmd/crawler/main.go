package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/sattwyk/autodocs/apps/crawler/internal/config"
	"github.com/sattwyk/autodocs/apps/crawler/internal/github"
	"github.com/sattwyk/autodocs/apps/crawler/internal/metrics"
	"github.com/sattwyk/autodocs/apps/crawler/internal/model"
	"github.com/sattwyk/autodocs/apps/crawler/internal/worker"
)

// Server represents the HTTP server for the crawler service
type Server struct {
	config       *config.Config
	metrics      *metrics.Metrics
	githubClient *github.Client
	workerPool   *worker.Pool
	httpServer   *http.Server
}

// NewServer creates a new server instance
func NewServer() (*Server, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize metrics
	m := metrics.New()

	// Initialize GitHub client
	ghClient, err := github.NewClient(cfg, m)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Initialize worker pool
	pool := worker.NewPool(cfg, m, ghClient)

	server := &Server{
		config:       cfg,
		metrics:      m,
		githubClient: ghClient,
		workerPool:   pool,
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	server.setupRoutes(mux)

	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler:      server.loggingMiddleware(server.metricsMiddleware(mux)),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server, nil
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/invoke", s.handleInvoke)
	mux.Handle(s.config.MetricsPath, promhttp.Handler())
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	// Start worker pool
	if err := s.workerPool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	log.Printf("Starting crawler service on %s", s.httpServer.Addr)

	// Start HTTP server in goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down crawler service...")

	// Stop worker pool
	if err := s.workerPool.Stop(); err != nil {
		log.Printf("Error stopping worker pool: %v", err)
	}

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	log.Println("Crawler service stopped")
	return nil
}

// handleRoot handles the root endpoint
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"service": "crawler",
		"status":  "running",
		"version": "1.0.0",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	status := "healthy"
	if !s.workerPool.IsRunning() {
		status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	response := model.HealthResponse{
		Status:    status,
		Service:   "crawler",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// handleInvoke handles the main crawl endpoint
func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse request
	var req model.CrawlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.RepoURL == "" {
		http.Error(w, "repo_url is required", http.StatusBadRequest)
		return
	}

	// Set default ref
	if req.Ref == "" {
		req.Ref = "main"
	}

	// Parse repository URL
	owner, repo, err := github.ParseRepositoryURL(req.RepoURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid repository URL: %v", err), http.StatusBadRequest)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	log.Printf("Starting crawl request for %s/%s", owner, repo)

	// Perform crawl
	response, err := s.workerPool.CrawlRepository(ctx, owner, repo, req.Ref, req.PathFilter)
	if err != nil {
		log.Printf("Crawl failed for %s/%s: %v", owner, repo, err)

		// Return structured error response
		errorResponse := &model.CrawlResponse{
			TotalFiles:     0,
			ProcessedFiles: 0,
			SkippedFiles:   0,
			Errors: []model.CrawlError{
				{
					FilePath: "",
					Error:    err.Error(),
					Type:     "crawl_failed",
				},
			},
			RepoInfo: model.RepositoryInfo{
				Owner: owner,
				Name:  repo,
				Ref:   req.Ref,
			},
		}

		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
			log.Printf("Failed to encode error response: %v", err)
		}
		return
	}

	log.Printf("Crawl completed for %s/%s: %d files processed, %d errors",
		owner, repo, response.ProcessedFiles, len(response.Errors))

	// Return success response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(wrapper, r)

		// Log request
		duration := time.Since(start)
		log.Printf("%s %s %d %v %s",
			r.Method, r.URL.Path, wrapper.statusCode, duration, r.RemoteAddr)
	})
}

// metricsMiddleware records metrics for HTTP requests
func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(wrapper, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		s.metrics.RecordHTTPRequest(r.Method, r.URL.Path, fmt.Sprintf("%d", wrapper.statusCode))
		s.metrics.RecordHTTPDuration(r.Method, r.URL.Path, duration)
	})
}

// responseWrapper wraps http.ResponseWriter to capture status code
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// main is the entry point
func main() {
	// Create server
	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Println("Crawler service started successfully")
	<-c

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(shutdownCtx); err != nil {
		log.Fatalf("Failed to shutdown server gracefully: %v", err)
	}
}
