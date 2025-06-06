# Crawler Service Testing Guide

This document describes the comprehensive test suite for the GitHub Crawler Service.

## Test Structure

The test suite is organized into several categories:

### Unit Tests

- **Config Tests** (`internal/config/config_test.go`): Tests configuration loading, validation, and helper functions
- **GitHub Client Tests** (`internal/github/client_test.go`): Tests GitHub API client functionality
- **Metrics Tests** (`internal/metrics/metrics_test.go`): Tests Prometheus metrics collection
- **Worker Pool Tests** (`internal/worker/pool_test.go`): Tests worker pool management and task processing
- **Model Tests** (`internal/model/types_test.go`): Tests data model serialization and validation
- **Main Server Tests** (`cmd/crawler/main_test.go`): Tests HTTP handlers and server functionality

### Integration Tests

- **Integration Tests** (`cmd/crawler/integration_test.go`): Tests complete workflows and component interactions

## Running Tests

### All Tests

```bash
make test
```

### Unit Tests Only

```bash
make test-unit
```

### Integration Tests Only

```bash
make test-integration-go
```

### Tests with Coverage

```bash
make test-coverage
```

This generates `coverage.html` with a visual coverage report.

### Benchmarks

```bash
make test-bench
```

### Race Detection

```bash
make test-race
```

### Complete Test Suite

```bash
make test-all
```

## Test Coverage

The test suite covers:

### Configuration (`config` package)

- ✅ Environment variable loading
- ✅ Configuration validation
- ✅ Default value handling
- ✅ GitHub authentication methods (PAT and App)
- ✅ Helper method functionality
- ✅ Error handling for invalid configurations

### GitHub Client (`github` package)

- ✅ Client initialization
- ✅ Authentication setup (PAT and GitHub App)
- ✅ Repository URL parsing
- ✅ API request handling
- ✅ Rate limiting
- ✅ Retry logic
- ✅ Error handling

### Metrics (`metrics` package)

- ✅ Metrics initialization
- ✅ Counter increments
- ✅ Gauge updates
- ✅ Histogram observations
- ✅ All metric types (HTTP, GitHub API, Worker Pool, etc.)

### Worker Pool (`worker` package)

- ✅ Pool creation and lifecycle
- ✅ Worker start/stop functionality
- ✅ Task submission and processing
- ✅ Queue management
- ✅ File filtering logic
- ✅ Binary content detection
- ✅ File size validation
- ✅ Concurrency control

### Data Models (`model` package)

- ✅ JSON serialization/deserialization
- ✅ Request/response structures
- ✅ Error handling structures
- ✅ GitHub API response models

### Main Server (`main` package)

- ✅ Server initialization
- ✅ HTTP endpoint handlers
- ✅ Middleware functionality
- ✅ Request validation
- ✅ Error responses
- ✅ Health checks
- ✅ Metrics endpoint

### Integration Tests

- ✅ Complete workflow testing
- ✅ Error handling scenarios
- ✅ Concurrent request handling
- ✅ Configuration validation
- ✅ Worker pool integration

## Test Environment Setup

### Required Environment Variables

For tests that require GitHub authentication:

```bash
export GITHUB_TOKEN="your-github-token"
```

### Optional Environment Variables

```bash
export GITHUB_APP_ID="123456"
export GITHUB_APP_KEY="-----BEGIN RSA PRIVATE KEY-----..."
export GITHUB_INSTALL_ID="789012"
```

## Test Data and Mocking

### Mock Servers

- HTTP test servers are used to simulate GitHub API responses
- Test data includes various repository structures and file types
- Error scenarios are simulated with appropriate HTTP status codes

### Test Isolation

- Each test clears and sets its own environment variables
- Tests use temporary configurations to avoid interference
- Worker pools are properly started and stopped in tests

## Performance Testing

### Benchmarks

The test suite includes benchmarks for:

- HTTP endpoint performance
- Configuration loading
- Metrics recording
- Worker pool operations

### Load Testing

Integration tests include:

- Concurrent request handling
- Worker pool stress testing
- Memory usage validation

## Continuous Integration

### Test Commands for CI

```bash
# Quick test suite for PR validation
make test-unit

# Full test suite for main branch
make test-all

# Coverage reporting
make test-coverage
```

### Test Timeouts

- Unit tests: Fast execution (< 1 second each)
- Integration tests: Moderate execution (< 30 seconds each)
- Benchmarks: Variable execution time

## Test Maintenance

### Adding New Tests

1. Follow the existing test structure and naming conventions
2. Use table-driven tests for multiple scenarios
3. Include both positive and negative test cases
4. Add appropriate setup and teardown
5. Update this documentation

### Test Dependencies

- `github.com/stretchr/testify` for assertions and test utilities
- Standard Go testing package
- `net/http/httptest` for HTTP testing
- `prometheus/client_golang/prometheus/testutil` for metrics testing

## Coverage Goals

Target coverage levels:

- Unit tests: > 90%
- Integration tests: > 80%
- Overall coverage: > 85%

## Known Limitations

1. Some GitHub API tests use mock servers instead of real API calls
2. Integration tests require proper environment setup
3. Performance benchmarks may vary based on system resources
4. Some error scenarios are difficult to reproduce in tests

## Troubleshooting

### Common Issues

1. **Tests fail with authentication errors**: Ensure `GITHUB_TOKEN` is set
2. **Race condition warnings**: Run with `-race` flag to identify issues
3. **Timeout errors**: Increase test timeouts for slower systems
4. **Coverage gaps**: Use `go tool cover` to identify untested code

### Debug Commands

```bash
# Verbose test output
go test -v ./...

# Run specific test
go test -v -run TestSpecificFunction ./...

# Debug with race detection
go test -v -race ./...
```
