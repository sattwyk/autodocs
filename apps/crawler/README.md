# GitHub Crawler Service

A production-grade Go service for crawling GitHub repositories with high concurrency, robust error handling, and comprehensive observability.

## Features

- **High Concurrency**: Configurable worker pool with thousands of files processed in parallel
- **GitHub Integration**: Supports both Personal Access Tokens (PAT) and GitHub App authentication
- **Rate Limiting**: Intelligent rate limiting to respect GitHub API limits
- **Adaptive Rate Limiting**: Automatically adjusts request rate based on GitHub's response headers
- **Error Handling**: Comprehensive retry logic with exponential backoff
- **Observability**: Prometheus metrics and structured logging
- **Resource Efficiency**: Memory-efficient streaming and configurable file size limits
- **Memory Management**: Automatic memory pressure detection and response
- **Task Pausing**: Pauses task processing instead of dropping during high load
- **Path Filtering**: Optional filtering to crawl specific directories
- **Graceful Shutdown**: Proper cleanup and resource management

## Architecture

```bash
crawler/
├── cmd/
│   └── crawler/main.go       # HTTP server and bootstrap
├── internal/
│   ├── github/               # GitHub API client with auth
│   ├── worker/               # Worker pool implementation
│   ├── config/               # Configuration management
│   ├── metrics/              # Prometheus instrumentation
│   └── model/                # Data types and models
├── go.mod
└── go.sum
```

## API Endpoints

### POST /invoke

Main crawling endpoint that accepts a repository URL and returns crawl results.

**Request:**

```json
{
  "repo_url": "https://github.com/owner/repo.git",
  "ref": "main",
  "path_filter": ["src/", "lib/"]
}
```

**Response:**

```json
{
  "total_files": 1500,
  "processed_files": 1450,
  "skipped_files": 50,
  "errors": [
    {
      "file_path": "src/large_file.bin",
      "error": "file size exceeds limit",
      "type": "file_too_large"
    }
  ],
  "root_tree_sha": "abc123...",
  "duration": "2m30s",
  "repo_info": {
    "owner": "owner",
    "name": "repo",
    "ref": "main"
  }
}
```

### GET /health

Health check endpoint.

**Response:**

```json
{
  "status": "healthy",
  "service": "crawler",
  "timestamp": "2024-01-01T12:00:00Z",
  "version": "1.0.0"
}
```

### GET /metrics

Prometheus metrics endpoint exposing:

- HTTP request metrics
- File processing counters
- GitHub API usage
- Worker pool status
- Error rates

### GET /

Service information endpoint.

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `HOST` | `0.0.0.0` | HTTP server host |
| `GITHUB_BASE_URL` | `https://api.github.com` | GitHub API base URL |
| `GITHUB_TOKEN` | - | Personal Access Token (required if no GitHub App) |
| `GITHUB_APP_ID` | - | GitHub App ID |
| `GITHUB_APP_KEY` | - | GitHub App private key (PEM format) |
| `GITHUB_INSTALL_ID` | - | GitHub App installation ID |
| `MAX_WORKERS` | `50` | Maximum number of worker goroutines |
| `API_RATE_LIMIT_THRESHOLD` | `100` | API rate limit threshold |
| `FETCH_TIMEOUT_MS` | `30000` | File fetch timeout in milliseconds |
| `RETRY_MAX_ATTEMPTS` | `3` | Maximum retry attempts |
| `RETRY_BACKOFF_MS_BASE` | `1000` | Base backoff time in milliseconds |
| `MAX_FILE_SIZE` | `10485760` | Maximum file size in bytes (10MB) |
| `MAX_CONCURRENT_FETCHES` | `100` | Maximum concurrent file fetches |
| `MEMORY_LIMIT_PERCENT` | `0.8` | Percentage of system memory to use (0.0-1.0) |
| `ENABLE_MEMORY_MONITOR` | `true` | Enable automatic memory pressure monitoring |
| `BACKPRESSURE_THRESHOLD` | `0.8` | Queue depth percentage to trigger backpressure |
| `TASK_BUFFER_SIZE` | `1000` | Size of buffer for paused tasks |
| `ENABLE_ADAPTIVE_RATE_LIMIT` | `true` | Enable adaptive rate limiting |
| `RATE_LIMIT_MIN_RATE` | `1.0` | Minimum requests per second |
| `RATE_LIMIT_MAX_RATE` | `50.0` | Maximum requests per second |
| `RATE_LIMIT_ADJUST_FACTOR` | `0.1` | Rate adjustment factor for adaptive limiting |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `METRICS_PATH` | `/metrics` | Prometheus metrics endpoint path |
| `ENVIRONMENT` | `development` | Environment (development, production) |

### Authentication

The service supports two authentication methods:

#### Personal Access Token (PAT)

```bash
export GITHUB_TOKEN="ghp_your_personal_access_token_here"
```

#### GitHub App (Recommended for production)

```bash
export GITHUB_APP_ID="123456"
export GITHUB_APP_KEY="-----BEGIN RSA PRIVATE KEY-----\n..."
export GITHUB_INSTALL_ID="12345678"
```

## Deployment

### Docker

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o crawler ./cmd/crawler

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/crawler .
EXPOSE 8080
CMD ["./crawler"]
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: crawler
spec:
  replicas: 3
  selector:
    matchLabels:
      app: crawler
  template:
    metadata:
      labels:
        app: crawler
    spec:
      containers:
      - name: crawler
        image: your-registry/crawler:latest
        ports:
        - containerPort: 8080
        env:
        - name: GITHUB_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-credentials
              key: token
        - name: MAX_WORKERS
          value: "100"
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Performance Tuning

### Worker Pool Configuration

- **Small repos** (<1000 files): `MAX_WORKERS=20`
- **Medium repos** (1000-10000 files): `MAX_WORKERS=50`
- **Large repos** (>10000 files): `MAX_WORKERS=100`

### Memory Optimization

- Set `MAX_FILE_SIZE` to prevent memory issues with large files
- Adjust `MAX_CONCURRENT_FETCHES` based on available memory
- Enable `ENABLE_MEMORY_MONITOR=true` for automatic memory pressure handling
- Tune `MEMORY_LIMIT_PERCENT` based on your system (default 80%)
- Monitor `crawler_file_size_bytes` metrics

### Rate Limiting

- Adjust `API_RATE_LIMIT_THRESHOLD` based on your GitHub plan
- Enable `ENABLE_ADAPTIVE_RATE_LIMIT=true` for automatic rate adjustment
- Configure `RATE_LIMIT_MIN_RATE` and `RATE_LIMIT_MAX_RATE` for your needs
- Monitor `crawler_github_rate_limit_*` metrics
- Use GitHub Apps for higher rate limits

### Handling Extremely Large Repositories

For repositories with hundreds of thousands of files:

1. **Enable Enhanced Features**:

   ```bash
   export ENABLE_MEMORY_MONITOR=true
   export ENABLE_ADAPTIVE_RATE_LIMIT=true
   export BACKPRESSURE_THRESHOLD=0.7
   export TASK_BUFFER_SIZE=5000
   ```

2. **Increase Worker Pool Gradually**:

   ```bash
   export MAX_WORKERS=200
   export MAX_CONCURRENT_FETCHES=300
   ```

3. **Configure Memory Limits**:

   ```bash
   export MEMORY_LIMIT_PERCENT=0.75
   export MAX_FILE_SIZE=5242880  # 5MB per file
   ```

4. **Fine-tune Rate Limiting**:

   ```bash
   export RATE_LIMIT_MIN_RATE=0.5
   export RATE_LIMIT_MAX_RATE=30.0
   export API_RATE_LIMIT_THRESHOLD=1000  # If using GitHub App
   ```

## Monitoring

### Key Metrics to Monitor

- `crawler_files_processed_total` - File processing rate
- `crawler_errors_total` - Error rate by type
- `crawler_github_rate_limit_used` - API usage
- `crawler_concurrency_in_use` - Active workers
- `crawler_http_request_duration_seconds` - Response times

### Alerts

```yaml
groups:
- name: crawler
  rules:
  - alert: CrawlerHighErrorRate
    expr: rate(crawler_errors_total[5m]) > 0.1
    for: 2m
    annotations:
      summary: "High error rate in crawler service"

  - alert: CrawlerRateLimitApproached
    expr: crawler_github_rate_limit_used / crawler_github_rate_limit_limit > 0.8
    for: 1m
    annotations:
      summary: "GitHub rate limit nearly exceeded"
```

## Development

### Building

```bash
go build -o crawler ./cmd/crawler
```

### Running Locally

```bash
export GITHUB_TOKEN="your_token_here"
./crawler
```

### Testing

```bash
# Test the service
curl -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "repo_url": "https://github.com/octocat/Hello-World.git",
    "ref": "main"
  }'
```

## Security Considerations

- Store GitHub credentials in secure secret management systems
- Use GitHub Apps with minimal required permissions
- Enable TLS for all deployments
- Monitor and rotate credentials regularly
- Validate and sanitize all input URLs
- Implement proper RBAC for deployment access

## Troubleshooting

### Common Issues

1. **Rate Limit Exceeded**
   - Check `crawler_github_rate_limit_*` metrics
   - Reduce `MAX_WORKERS` or increase `API_RATE_LIMIT_THRESHOLD`
   - Consider using GitHub App authentication

2. **Memory Issues**
   - Reduce `MAX_FILE_SIZE` and `MAX_CONCURRENT_FETCHES`
   - Monitor `crawler_file_size_bytes` histogram
   - Consider horizontal scaling

3. **Timeout Errors**
   - Increase `FETCH_TIMEOUT_MS`
   - Check network connectivity to GitHub
   - Monitor `crawler_task_duration_seconds`

4. **Authentication Failures**
   - Verify token/credentials are valid and have required permissions
   - Check token expiration for GitHub Apps
   - Ensure installation ID is correct for GitHub Apps

## License

[Add your license here]
