# Enhanced Crawler Testing Guide

This guide explains how to test the enhanced crawler features for handling extremely large repositories.

## Enhanced Features

The enhanced crawler includes several improvements for handling large repositories:

1. **Memory Management**: Automatic memory pressure detection and response
2. **Adaptive Rate Limiting**: Dynamically adjusts API request rate based on GitHub's responses
3. **Task Pausing**: Pauses task processing instead of dropping tasks during high load
4. **Backpressure Handling**: Manages queue depth to prevent resource exhaustion
5. **Streaming Support**: Memory-efficient file content processing

## Prerequisites

1. **GitHub Token**: You need a GitHub Personal Access Token or GitHub App credentials
2. **System Requirements**: At least 4GB RAM for testing large repositories
3. **Dependencies**: `curl`, `jq` (optional but recommended), `bc` (for calculations)

## Quick Start

### 1. Build and Start the Crawler

```bash
# Build the enhanced crawler
make build

# Start with enhanced configuration for large repos
export GITHUB_TOKEN="your_github_token_here"
export ENABLE_MEMORY_MONITOR=true
export ENABLE_ADAPTIVE_RATE_LIMIT=true
export MAX_WORKERS=100
export MAX_CONCURRENT_FETCHES=200
export MEMORY_LIMIT_PERCENT=0.75

./crawler
```

### 2. Run Basic Tests

```bash
# Test basic functionality
./test_basic.sh

# Test on a small repository first
./test_large_repo.sh small
```

### 3. Test Large Repositories

```bash
# Test React repository (large)
./test_large_repo.sh large

# Test VS Code repository (extra large)
./test_large_repo.sh xlarge

# Run all tests
./test_large_repo.sh all
```

## Configuration for Large Repositories

### Memory Management

```bash
# Enable memory monitoring
export ENABLE_MEMORY_MONITOR=true

# Set memory limit (percentage of system memory)
export MEMORY_LIMIT_PERCENT=0.75

# Configure backpressure threshold
export BACKPRESSURE_THRESHOLD=0.7

# Set task buffer size for paused tasks
export TASK_BUFFER_SIZE=5000
```

### Adaptive Rate Limiting

```bash
# Enable adaptive rate limiting
export ENABLE_ADAPTIVE_RATE_LIMIT=true

# Set rate limits (requests per second)
export RATE_LIMIT_MIN_RATE=1.0
export RATE_LIMIT_MAX_RATE=30.0

# Adjustment factor for rate changes
export RATE_LIMIT_ADJUST_FACTOR=0.1
```

### Worker Pool Configuration

```bash
# For extremely large repositories
export MAX_WORKERS=200
export MAX_CONCURRENT_FETCHES=300

# File size limits
export MAX_FILE_SIZE=5242880  # 5MB per file
```

## Test Scenarios

### 1. Small Repository Test

Tests basic functionality with a small repository:

```bash
./test_large_repo.sh small
```

**Expected Results:**

- Fast completion (< 30 seconds)
- High success rate (> 95%)
- Low memory usage
- No rate limiting issues

### 2. Medium Repository Test

Tests with a medium-sized repository (Express.js):

```bash
./test_large_repo.sh medium
```

**Expected Results:**

- Completion in 1-5 minutes
- Good success rate (> 90%)
- Moderate memory usage
- Some rate limiting adjustments

### 3. Large Repository Test

Tests with React repository (~50,000 files):

```bash
./test_large_repo.sh large
```

**Expected Results:**

- Completion in 10-30 minutes
- Acceptable success rate (> 80%)
- Memory pressure handling activated
- Adaptive rate limiting in effect
- Task pausing/resuming events

### 4. Memory Stress Test

Monitors memory usage during large repository crawling:

```bash
./test_large_repo.sh memory
```

**What to Monitor:**

- Memory usage stays below configured limit
- Memory pressure detection works
- Garbage collection is triggered when needed
- Workers pause/resume based on memory pressure

### 5. Rate Limit Test

Tests adaptive rate limiting with concurrent requests:

```bash
./test_large_repo.sh rate
```

**What to Monitor:**

- Rate limiting adjusts based on GitHub responses
- No 429 (rate limit exceeded) errors
- Concurrent requests are handled properly

## Monitoring and Metrics

### Key Metrics to Watch

1. **Memory Usage**:

   ```bash
   curl -s http://localhost:8080/metrics | grep memory
   ```

2. **Rate Limiting**:

   ```bash
   curl -s http://localhost:8080/metrics | grep rate_limit
   ```

3. **Worker Pool Status**:

   ```bash
   curl -s http://localhost:8080/metrics | grep worker
   ```

4. **Queue Depth**:

   ```bash
   curl -s http://localhost:8080/metrics | grep queue
   ```

### Log Analysis

Monitor the crawler logs for:

- Memory pressure events: `Memory pressure detected`
- Worker pausing: `Pausing workers due to resource constraints`
- Rate limit adjustments: Rate limiting messages
- Task buffering: `task_buffered` events

## Troubleshooting

### High Memory Usage

If memory usage is too high:

1. Reduce `MEMORY_LIMIT_PERCENT`
2. Decrease `MAX_CONCURRENT_FETCHES`
3. Lower `MAX_FILE_SIZE`
4. Reduce `MAX_WORKERS`

### Rate Limiting Issues

If hitting rate limits:

1. Lower `RATE_LIMIT_MAX_RATE`
2. Increase `API_RATE_LIMIT_THRESHOLD` (if using GitHub App)
3. Enable `ENABLE_ADAPTIVE_RATE_LIMIT=true`

### Slow Performance

If crawling is too slow:

1. Increase `MAX_WORKERS` gradually
2. Increase `MAX_CONCURRENT_FETCHES`
3. Raise `RATE_LIMIT_MAX_RATE` (if not hitting limits)
4. Check network connectivity

### Task Dropping

If tasks are being dropped:

1. Increase `TASK_BUFFER_SIZE`
2. Adjust `BACKPRESSURE_THRESHOLD`
3. Enable memory monitoring
4. Check system resources

## Performance Benchmarks

### Expected Performance for Different Repository Sizes

| Repository Size | Files | Expected Time | Memory Usage | Success Rate |
|----------------|-------|---------------|--------------|--------------|
| Small (< 1K)   | < 1,000 | < 1 min | < 100MB | > 95% |
| Medium (1K-10K) | 1,000-10,000 | 1-10 min | 100-500MB | > 90% |
| Large (10K-100K) | 10,000-100,000 | 10-60 min | 500MB-2GB | > 80% |
| XLarge (> 100K) | > 100,000 | 1-6 hours | 1-4GB | > 70% |

### Optimization Tips

1. **For Large Repositories**:
   - Use GitHub App authentication for higher rate limits
   - Enable all enhanced features
   - Monitor memory usage closely
   - Use path filtering if possible

2. **For Production**:
   - Set conservative memory limits
   - Enable comprehensive monitoring
   - Use horizontal scaling for very large repositories
   - Implement proper error handling and retries

## Example Test Results

After running tests, you'll find results in the `test_results/` directory:

- `summary_TIMESTAMP.md`: Overall test summary
- `REPO_TIMESTAMP.json`: Detailed crawl results
- `REPO_TIMESTAMP_stats.txt`: System resource usage
- `REPO_TIMESTAMP_metrics.txt`: Prometheus metrics before/after

## Advanced Testing

### Custom Repository Testing

To test your own repository:

```bash
# Edit the test script to add your repository
vim test_large_repo.sh

# Add to TEST_REPOS array:
["custom"]="https://github.com/your-org/your-repo.git"

# Run the test
./test_large_repo.sh custom
```

### Load Testing

For load testing with multiple concurrent crawls:

```bash
# Start multiple crawls in parallel
for i in {1..5}; do
    curl -X POST http://localhost:8080/invoke \
        -H "Content-Type: application/json" \
        -d '{"repo_url": "https://github.com/facebook/react.git", "ref": "main"}' &
done

# Monitor system resources
watch -n 1 'free -h && echo "---" && ps aux | grep crawler'
```

## Conclusion

The enhanced crawler provides robust handling of large repositories through:

- Intelligent memory management
- Adaptive rate limiting
- Task pausing instead of dropping
- Comprehensive monitoring

Use the provided test scripts to validate performance and tune configuration for your specific use cases.
