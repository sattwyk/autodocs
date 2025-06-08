#!/bin/bash

# Test script for enhanced crawler on large repositories
# This script tests the crawler against the React repository and other large codebases

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CRAWLER_URL="http://localhost:8080"
TEST_RESULTS_DIR="./test_results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Test repositories (ordered by size)
declare -A TEST_REPOS=(
    ["small"]="https://github.com/octocat/Hello-World.git"
    ["medium"]="https://github.com/expressjs/express.git"
    ["large"]="https://github.com/facebook/react.git"
    ["xlarge"]="https://github.com/microsoft/vscode.git"
)

# Enhanced configuration for large repos
export ENABLE_MEMORY_MONITOR=true
export ENABLE_ADAPTIVE_RATE_LIMIT=true
export BACKPRESSURE_THRESHOLD=0.7
export TASK_BUFFER_SIZE=5000
export MAX_WORKERS=100
export MAX_CONCURRENT_FETCHES=200
export MEMORY_LIMIT_PERCENT=0.75
export MAX_FILE_SIZE=5242880  # 5MB
export RATE_LIMIT_MIN_RATE=1.0
export RATE_LIMIT_MAX_RATE=30.0

echo -e "${BLUE}=== Enhanced Crawler Large Repository Test ===${NC}"
echo "Timestamp: $TIMESTAMP"
echo "Results will be saved to: $TEST_RESULTS_DIR"

# Create results directory
mkdir -p "$TEST_RESULTS_DIR"

# Function to check if crawler is running
check_crawler() {
    echo -e "${YELLOW}Checking if crawler is running...${NC}"
    if curl -s "$CRAWLER_URL/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Crawler is running${NC}"
        return 0
    else
        echo -e "${RED}✗ Crawler is not running${NC}"
        return 1
    fi
}

# Function to get system stats
get_system_stats() {
    echo "=== System Stats ===" > "$1"
    echo "Timestamp: $(date)" >> "$1"
    echo "Memory:" >> "$1"
    free -h >> "$1"
    echo "" >> "$1"
    echo "CPU:" >> "$1"
    nproc >> "$1"
    echo "" >> "$1"
    echo "Disk:" >> "$1"
    df -h . >> "$1"
    echo "" >> "$1"
}

# Function to test a repository
test_repository() {
    local repo_name="$1"
    local repo_url="$2"
    local test_file="$TEST_RESULTS_DIR/${repo_name}_${TIMESTAMP}.json"
    local stats_file="$TEST_RESULTS_DIR/${repo_name}_${TIMESTAMP}_stats.txt"
    local metrics_file="$TEST_RESULTS_DIR/${repo_name}_${TIMESTAMP}_metrics.txt"

    echo -e "${BLUE}Testing repository: $repo_name${NC}"
    echo "URL: $repo_url"

    # Get initial system stats
    get_system_stats "$stats_file"

    # Get initial metrics
    echo "=== Initial Metrics ===" > "$metrics_file"
    curl -s "$CRAWLER_URL/metrics" >> "$metrics_file" 2>/dev/null || echo "Failed to get initial metrics" >> "$metrics_file"
    echo "" >> "$metrics_file"

    # Prepare request payload
    local payload=$(cat <<EOF
{
    "repo_url": "$repo_url",
    "ref": "main"
}
EOF
)

    echo "Starting crawl..."
    local start_time=$(date +%s)

    # Make the request and capture response
    local response=$(curl -s -w "\n%{http_code}\n%{time_total}" \
        -X POST "$CRAWLER_URL/invoke" \
        -H "Content-Type: application/json" \
        -d "$payload")

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    # Parse response
    local http_code=$(echo "$response" | tail -n 2 | head -n 1)
    local time_total=$(echo "$response" | tail -n 1)
    local json_response=$(echo "$response" | head -n -2)

    # Save response
    echo "$json_response" > "$test_file"

    # Get final metrics
    echo "=== Final Metrics ===" >> "$metrics_file"
    curl -s "$CRAWLER_URL/metrics" >> "$metrics_file" 2>/dev/null || echo "Failed to get final metrics" >> "$metrics_file"

    # Get final system stats
    echo "=== Final System Stats ===" >> "$stats_file"
    get_system_stats "/tmp/final_stats"
    cat "/tmp/final_stats" >> "$stats_file"

    # Analyze results
    echo -e "${YELLOW}Results for $repo_name:${NC}"
    echo "HTTP Code: $http_code"
    echo "Duration: ${duration}s (curl time: ${time_total}s)"

    if [ "$http_code" = "200" ]; then
        echo -e "${GREEN}✓ Request successful${NC}"

        # Parse JSON response for key metrics
        if command -v jq > /dev/null 2>&1; then
            local total_files=$(echo "$json_response" | jq -r '.total_files // "N/A"')
            local processed_files=$(echo "$json_response" | jq -r '.processed_files // "N/A"')
            local skipped_files=$(echo "$json_response" | jq -r '.skipped_files // "N/A"')
            local error_count=$(echo "$json_response" | jq -r '.errors | length // "N/A"')
            local crawl_duration=$(echo "$json_response" | jq -r '.duration // "N/A"')

            echo "Total files: $total_files"
            echo "Processed files: $processed_files"
            echo "Skipped files: $skipped_files"
            echo "Errors: $error_count"
            echo "Crawl duration: $crawl_duration"

            # Calculate success rate
            if [ "$total_files" != "N/A" ] && [ "$total_files" != "0" ]; then
                local success_rate=$(echo "scale=2; $processed_files * 100 / $total_files" | bc -l 2>/dev/null || echo "N/A")
                echo "Success rate: ${success_rate}%"
            fi
        else
            echo "jq not available, raw response saved to $test_file"
        fi
    else
        echo -e "${RED}✗ Request failed${NC}"
        echo "Response: $json_response"
    fi

    echo "Results saved to: $test_file"
    echo "Stats saved to: $stats_file"
    echo "Metrics saved to: $metrics_file"
    echo ""
}

# Function to run memory stress test
memory_stress_test() {
    echo -e "${BLUE}=== Memory Stress Test ===${NC}"
    local stress_file="$TEST_RESULTS_DIR/memory_stress_${TIMESTAMP}.txt"

    echo "Running memory stress test on React repository..."
    echo "This test will monitor memory usage during crawling"

    # Start memory monitoring in background
    (
        echo "=== Memory Usage During Crawl ===" > "$stress_file"
        while true; do
            echo "$(date): $(free -m | grep '^Mem:' | awk '{print $3 "MB used / " $2 "MB total"}')" >> "$stress_file"
            sleep 5
        done
    ) &
    local monitor_pid=$!

    # Run the test
    test_repository "react_stress" "https://github.com/facebook/react.git"

    # Stop monitoring
    kill $monitor_pid 2>/dev/null || true

    echo "Memory stress test completed. Results in: $stress_file"
}

# Function to run rate limit test
rate_limit_test() {
    echo -e "${BLUE}=== Rate Limit Test ===${NC}"
    local rate_file="$TEST_RESULTS_DIR/rate_limit_${TIMESTAMP}.txt"

    echo "Testing adaptive rate limiting with multiple concurrent requests..."

    # Set aggressive rate limiting for testing
    export RATE_LIMIT_MAX_RATE=5.0
    export API_RATE_LIMIT_THRESHOLD=50

    echo "=== Rate Limit Test ===" > "$rate_file"
    echo "Max rate: $RATE_LIMIT_MAX_RATE req/s" >> "$rate_file"
    echo "Threshold: $API_RATE_LIMIT_THRESHOLD" >> "$rate_file"
    echo "" >> "$rate_file"

    # Run multiple small repo tests concurrently
    for i in {1..3}; do
        echo "Starting concurrent test $i..."
        test_repository "concurrent_$i" "https://github.com/octocat/Hello-World.git" &
    done

    # Wait for all tests to complete
    wait

    echo "Rate limit test completed. Check individual test files for results."
}

# Function to generate summary report
generate_summary() {
    local summary_file="$TEST_RESULTS_DIR/summary_${TIMESTAMP}.md"

    echo "# Enhanced Crawler Test Summary" > "$summary_file"
    echo "Generated: $(date)" >> "$summary_file"
    echo "" >> "$summary_file"

    echo "## Configuration" >> "$summary_file"
    echo "- Memory Monitor: $ENABLE_MEMORY_MONITOR" >> "$summary_file"
    echo "- Adaptive Rate Limit: $ENABLE_ADAPTIVE_RATE_LIMIT" >> "$summary_file"
    echo "- Max Workers: $MAX_WORKERS" >> "$summary_file"
    echo "- Max Concurrent Fetches: $MAX_CONCURRENT_FETCHES" >> "$summary_file"
    echo "- Memory Limit: $MEMORY_LIMIT_PERCENT" >> "$summary_file"
    echo "- Max File Size: $MAX_FILE_SIZE bytes" >> "$summary_file"
    echo "" >> "$summary_file"

    echo "## Test Results" >> "$summary_file"
    echo "" >> "$summary_file"

    # Analyze each test result
    for result_file in "$TEST_RESULTS_DIR"/*_${TIMESTAMP}.json; do
        if [ -f "$result_file" ]; then
            local repo_name=$(basename "$result_file" "_${TIMESTAMP}.json")
            echo "### $repo_name" >> "$summary_file"

            if command -v jq > /dev/null 2>&1; then
                local total=$(jq -r '.total_files // "N/A"' "$result_file")
                local processed=$(jq -r '.processed_files // "N/A"' "$result_file")
                local duration=$(jq -r '.duration // "N/A"' "$result_file")

                echo "- Total files: $total" >> "$summary_file"
                echo "- Processed files: $processed" >> "$summary_file"
                echo "- Duration: $duration" >> "$summary_file"
            fi
            echo "" >> "$summary_file"
        fi
    done

    echo "Summary report generated: $summary_file"
}

# Main execution
main() {
    echo -e "${BLUE}Starting enhanced crawler tests...${NC}"

    # Check if crawler is running
    if ! check_crawler; then
        echo -e "${RED}Please start the crawler first with: make run${NC}"
        exit 1
    fi

    # Check dependencies
    if ! command -v bc > /dev/null 2>&1; then
        echo -e "${YELLOW}Warning: bc not found, some calculations may not work${NC}"
    fi

    # Run tests based on arguments
    case "${1:-all}" in
        "small")
            test_repository "small" "${TEST_REPOS[small]}"
            ;;
        "medium")
            test_repository "medium" "${TEST_REPOS[medium]}"
            ;;
        "large")
            test_repository "large" "${TEST_REPOS[large]}"
            ;;
        "xlarge")
            test_repository "xlarge" "${TEST_REPOS[xlarge]}"
            ;;
        "memory")
            memory_stress_test
            ;;
        "rate")
            rate_limit_test
            ;;
        "all")
            echo "Running all tests..."
            test_repository "small" "${TEST_REPOS[small]}"
            test_repository "medium" "${TEST_REPOS[medium]}"
            test_repository "large" "${TEST_REPOS[large]}"
            memory_stress_test
            rate_limit_test
            ;;
        *)
            echo "Usage: $0 [small|medium|large|xlarge|memory|rate|all]"
            echo "Default: all"
            exit 1
            ;;
    esac

    # Generate summary
    generate_summary

    echo -e "${GREEN}All tests completed!${NC}"
    echo "Results available in: $TEST_RESULTS_DIR"
}

# Run main function with all arguments
main "$@"
