#!/bin/bash

# GitHub Crawler Service Test Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
BASE_URL="http://localhost:8080"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

test_endpoint() {
    local method=$1
    local endpoint=$2
    local expected_status=$3
    local data=$4

    log_info "Testing $method $endpoint"

    if [ -n "$data" ]; then
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X "$method" "$BASE_URL$endpoint")
    fi

    http_status=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$response" | sed -e 's/HTTPSTATUS\:.*//g')

    if [ "$http_status" -eq "$expected_status" ]; then
        log_info "✓ $method $endpoint returned $http_status"
        echo "Response: $body" | jq '.' 2>/dev/null || echo "Response: $body"
        return 0
    else
        log_error "✗ $method $endpoint returned $http_status, expected $expected_status"
        echo "Response: $body"
        return 1
    fi
}

# Check if service is running
check_service() {
    log_info "Checking if crawler service is running..."
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        log_info "✓ Service is running"
        return 0
    else
        log_error "✗ Service is not running or not accessible at $BASE_URL"
        log_info "Make sure the service is started with: ./crawler"
        exit 1
    fi
}

# Test basic endpoints
test_basic_endpoints() {
    log_info "Testing basic endpoints..."

    # Test root endpoint
    test_endpoint "GET" "/" 200
    echo

    # Test health endpoint
    test_endpoint "GET" "/health" 200
    echo

    # Test metrics endpoint
    log_info "Testing GET /metrics"
    metrics_response=$(curl -s "$BASE_URL/metrics")
    if echo "$metrics_response" | grep -q "crawler_"; then
        log_info "✓ GET /metrics returned crawler metrics"
    else
        log_error "✗ GET /metrics did not return expected metrics"
    fi
    echo
}

# Test crawl endpoint
test_crawl_endpoint() {
    log_info "Testing crawl endpoint..."

    if [ -z "$GITHUB_TOKEN" ]; then
        log_warn "GITHUB_TOKEN not set, skipping crawl tests"
        log_info "To test crawling, set GITHUB_TOKEN environment variable"
        return 0
    fi

    # Test with a small public repository
    local test_payload='{
        "repo_url": "https://github.com/sattwyk/mailmon.git",
        "ref": "main"
    }'

    log_info "Testing crawl with sattwyk/mailmon repository..."
    test_endpoint "POST" "/invoke" 200 "$test_payload"
    echo

    # Test with path filter
    local test_payload_filtered='{
        "repo_url": "https://github.com/sattwyk/mailmon.git",
        "ref": "main",
        "path_filter": ["README.md"]
    }'

    log_info "Testing crawl with path filter..."
    test_endpoint "POST" "/invoke" 200 "$test_payload_filtered"
    echo
}

# Test error cases
test_error_cases() {
    log_info "Testing error cases..."

    # Test invalid method on /invoke
    test_endpoint "GET" "/invoke" 405
    echo

    # Test invalid JSON
    test_endpoint "POST" "/invoke" 400 '{"invalid": json}'
    echo

    # Test missing repo_url
    test_endpoint "POST" "/invoke" 400 '{}'
    echo

    # Test invalid repo URL
    local invalid_payload='{"repo_url": "not-a-valid-url"}'
    test_endpoint "POST" "/invoke" 400 "$invalid_payload"
    echo
}

# Performance test (basic)
performance_test() {
    log_info "Running basic performance test..."

    local start_time=$(date +%s)

    # Make 10 concurrent requests to health endpoint
    for i in {1..10}; do
        curl -s "$BASE_URL/health" > /dev/null &
    done
    wait

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    log_info "✓ 10 concurrent health checks completed in ${duration}s"
    echo
}

# Main execution
main() {
    log_info "Starting GitHub Crawler Service Tests"
    echo "Base URL: $BASE_URL"
    echo "GitHub Token: ${GITHUB_TOKEN:+Set}${GITHUB_TOKEN:-Not Set}"
    echo

    # Check prerequisites
    if ! command -v curl &> /dev/null; then
        log_error "curl is required but not installed"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        log_warn "jq not found, JSON responses will not be formatted"
    fi

    # Run tests
    check_service
    echo "=================================="
    test_basic_endpoints
    echo "=================================="
    test_crawl_endpoint
    echo "=================================="
    test_error_cases
    echo "=================================="
    performance_test
    echo "=================================="

    log_info "All tests completed successfully!"
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo "Test the GitHub Crawler Service"
        echo ""
        echo "Environment Variables:"
        echo "  GITHUB_TOKEN    GitHub Personal Access Token for crawl tests"
        echo "  BASE_URL        Service base URL (default: http://localhost:8080)"
        echo ""
        echo "Examples:"
        echo "  $0                                    # Run basic tests"
        echo "  GITHUB_TOKEN=ghp_xxx $0              # Run all tests including crawl"
        echo "  BASE_URL=http://example.com:8080 $0  # Test remote service"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac