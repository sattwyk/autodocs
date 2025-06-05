package model

import (
	"time"
)

// CrawlRequest represents the incoming request to crawl a repository
type CrawlRequest struct {
	RepoURL    string   `json:"repo_url"`
	Ref        string   `json:"ref,omitempty"`         // branch/tag/sha, defaults to "main"
	PathFilter []string `json:"path_filter,omitempty"` // optional filter for specific paths
}

// CrawlResponse represents the response after crawling
type CrawlResponse struct {
	TotalFiles     int            `json:"total_files"`
	SkippedFiles   int            `json:"skipped_files"`
	ProcessedFiles int            `json:"processed_files"`
	Errors         []CrawlError   `json:"errors"`
	RootTreeSHA    string         `json:"root_tree_sha"`
	Duration       string         `json:"duration"`
	RepoInfo       RepositoryInfo `json:"repo_info"`
	Files          []FileResult   `json:"files,omitempty"`
}

// CrawlError represents an error that occurred during crawling
type CrawlError struct {
	FilePath string `json:"file_path"`
	Error    string `json:"error"`
	Type     string `json:"type"` // "api_error", "timeout", "permission_denied", etc.
}

// RepositoryInfo contains basic repository information
type RepositoryInfo struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
	Ref   string `json:"ref"`
}

// TreeEntry represents a file or directory in the Git tree
type TreeEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"` // "blob", "tree"
	SHA  string `json:"sha"`
	Size int    `json:"size,omitempty"`
}

// FileResult represents the result of fetching a file
type FileResult struct {
	Path      string    `json:"path"`
	Content   []byte    `json:"content,omitempty"`
	SHA       string    `json:"sha"`
	Size      int       `json:"size"`
	Error     error     `json:"error,omitempty"`
	FetchedAt time.Time `json:"fetched_at"`
}

// WorkerTask represents a task for the worker pool
type WorkerTask struct {
	Path  string
	SHA   string
	Size  int
	Owner string // Repository owner
	Repo  string // Repository name
	Ref   string // Git reference (branch/tag/sha)
}

// GitHubTreeResponse represents the GitHub API tree response
type GitHubTreeResponse struct {
	SHA       string      `json:"sha"`
	URL       string      `json:"url"`
	Tree      []TreeEntry `json:"tree"`
	Truncated bool        `json:"truncated"`
}

// GitHubContentResponse represents the GitHub API content response
type GitHubContentResponse struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	GitURL      string `json:"git_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
}

// RateLimitInfo represents GitHub API rate limit information
type RateLimitInfo struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     time.Time `json:"reset"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
}
