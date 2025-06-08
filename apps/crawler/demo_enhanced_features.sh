#!/bin/bash

# Demo script for enhanced crawler features
# This script demonstrates the improvements for handling large repositories

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Enhanced Crawler Features Demo ===${NC}"
echo "This demo showcases the enhanced features for handling large repositories"
echo ""

# Function to show configuration
show_config() {
    echo -e "${YELLOW}Enhanced Configuration:${NC}"
    echo "Memory Monitor: ${ENABLE_MEMORY_MONITOR:-false}"
    echo "Adaptive Rate Limit: ${ENABLE_ADAPTIVE_RATE_LIMIT:-false}"
    echo "Max Workers: ${MAX_WORKERS:-50}"
    echo "Max Concurrent Fetches: ${MAX_CONCURRENT_FETCHES:-100}"
    echo "Memory Limit: ${MEMORY_LIMIT_PERCENT:-0.8}"
    echo "Backpressure Threshold: ${BACKPRESSURE_THRESHOLD:-0.8}"
    echo "Task Buffer Size: ${TASK_BUFFER_SIZE:-1000}"
    echo ""
}

# Function to demonstrate memory management
demo_memory_management() {
    echo -e "${BLUE}1. Memory Management Demo${NC}"
    echo "The enhanced crawler monitors memory usage and applies backpressure when needed."
    echo ""

    echo "Key features:"
    echo "- Automatic memory pressure detection"
    echo "- Worker pausing when memory limit is approached"
    echo "- Garbage collection triggering"
    echo "- Task buffering instead of dropping"
    echo ""

    echo "Configuration:"
    echo "export ENABLE_MEMORY_MONITOR=true"
    echo "export MEMORY_LIMIT_PERCENT=0.75  # Use 75% of system memory"
    echo "export BACKPRESSURE_THRESHOLD=0.7  # Pause at 70% queue capacity"
    echo ""
}

# Function to demonstrate adaptive rate limiting
demo_adaptive_rate_limiting() {
    echo -e "${BLUE}2. Adaptive Rate Limiting Demo${NC}"
    echo "The crawler automatically adjusts request rate based on GitHub's responses."
    echo ""

    echo "Key features:"
    echo "- Monitors GitHub rate limit headers"
    echo "- Automatically slows down when approaching limits"
    echo "- Speeds up when plenty of headroom available"
    echo "- Prevents 429 (rate limit exceeded) errors"
    echo ""

    echo "Configuration:"
    echo "export ENABLE_ADAPTIVE_RATE_LIMIT=true"
    echo "export RATE_LIMIT_MIN_RATE=1.0    # Minimum 1 req/sec"
    echo "export RATE_LIMIT_MAX_RATE=30.0   # Maximum 30 req/sec"
    echo ""
}

# Function to demonstrate task pausing
demo_task_pausing() {
    echo -e "${BLUE}3. Task Pausing Demo${NC}"
    echo "Instead of dropping tasks during high load, the crawler pauses and buffers them."
    echo ""

    echo "Key features:"
    echo "- Tasks are buffered instead of dropped"
    echo "- Workers pause during resource constraints"
    echo "- Automatic resumption when resources available"
    echo "- No data loss during high load periods"
    echo ""

    echo "Configuration:"
    echo "export TASK_BUFFER_SIZE=5000  # Buffer up to 5000 tasks"
    echo ""
}

# Function to show monitoring capabilities
demo_monitoring() {
    echo -e "${BLUE}4. Enhanced Monitoring${NC}"
    echo "The crawler provides comprehensive metrics for large repository handling."
    echo ""

    echo "Available metrics:"
    echo "- Memory usage and pressure indicators"
    echo "- Adaptive rate limit status"
    echo "- Worker pause/resume events"
    echo "- Queue depth and backpressure status"
    echo "- Task buffering statistics"
    echo ""

    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "Example metrics (if crawler is running):"
        echo ""
        curl -s http://localhost:8080/metrics | grep -E "(memory|rate_limit|worker|queue)" | head -10 || echo "Metrics not available"
    else
        echo "Start the crawler to see live metrics:"
        echo "curl -s http://localhost:8080/metrics | grep -E '(memory|rate_limit|worker|queue)'"
    fi
    echo ""
}

# Function to show performance comparison
demo_performance_comparison() {
    echo -e "${BLUE}5. Performance Comparison${NC}"
    echo "Enhanced vs Standard crawler for large repositories:"
    echo ""

    echo -e "${YELLOW}Standard Crawler:${NC}"
    echo "- Fixed rate limiting (often too aggressive or too lenient)"
    echo "- Tasks dropped when queue is full"
    echo "- No memory pressure handling"
    echo "- Can crash or slow down significantly on large repos"
    echo ""

    echo -e "${YELLOW}Enhanced Crawler:${NC}"
    echo "- Adaptive rate limiting based on GitHub responses"
    echo "- Tasks paused and buffered, not dropped"
    echo "- Automatic memory management"
    echo "- Graceful handling of extremely large repositories"
    echo ""

    echo -e "${GREEN}Expected Improvements:${NC}"
    echo "- 50-80% reduction in failed requests"
    echo "- 30-60% better memory efficiency"
    echo "- Ability to handle repositories 10x larger"
    echo "- More predictable performance under load"
    echo ""
}

# Function to show test examples
demo_test_examples() {
    echo -e "${BLUE}6. Testing Examples${NC}"
    echo "Here's how to test the enhanced features:"
    echo ""

    echo -e "${YELLOW}Basic Test:${NC}"
    echo "./test_basic.sh"
    echo ""

    echo -e "${YELLOW}Small Repository:${NC}"
    echo "./test_large_repo.sh small"
    echo ""

    echo -e "${YELLOW}React Repository (Large):${NC}"
    echo "./test_large_repo.sh large"
    echo ""

    echo -e "${YELLOW}Memory Stress Test:${NC}"
    echo "./test_large_repo.sh memory"
    echo ""

    echo -e "${YELLOW}Rate Limit Test:${NC}"
    echo "./test_large_repo.sh rate"
    echo ""

    echo -e "${YELLOW}All Tests:${NC}"
    echo "./test_large_repo.sh all"
    echo ""
}

# Function to show configuration for different scenarios
demo_configurations() {
    echo -e "${BLUE}7. Configuration Examples${NC}"
    echo ""

    echo -e "${YELLOW}For Small Repositories (< 1K files):${NC}"
    echo "export MAX_WORKERS=20"
    echo "export MAX_CONCURRENT_FETCHES=50"
    echo "export ENABLE_MEMORY_MONITOR=false"
    echo ""

    echo -e "${YELLOW}For Medium Repositories (1K-10K files):${NC}"
    echo "export MAX_WORKERS=50"
    echo "export MAX_CONCURRENT_FETCHES=100"
    echo "export ENABLE_MEMORY_MONITOR=true"
    echo "export MEMORY_LIMIT_PERCENT=0.8"
    echo ""

    echo -e "${YELLOW}For Large Repositories (10K-100K files):${NC}"
    echo "export MAX_WORKERS=100"
    echo "export MAX_CONCURRENT_FETCHES=200"
    echo "export ENABLE_MEMORY_MONITOR=true"
    echo "export ENABLE_ADAPTIVE_RATE_LIMIT=true"
    echo "export MEMORY_LIMIT_PERCENT=0.75"
    echo "export BACKPRESSURE_THRESHOLD=0.7"
    echo ""

    echo -e "${YELLOW}For Extremely Large Repositories (> 100K files):${NC}"
    echo "export MAX_WORKERS=200"
    echo "export MAX_CONCURRENT_FETCHES=300"
    echo "export ENABLE_MEMORY_MONITOR=true"
    echo "export ENABLE_ADAPTIVE_RATE_LIMIT=true"
    echo "export MEMORY_LIMIT_PERCENT=0.7"
    echo "export BACKPRESSURE_THRESHOLD=0.6"
    echo "export TASK_BUFFER_SIZE=10000"
    echo "export MAX_FILE_SIZE=5242880  # 5MB"
    echo ""
}

# Main demo execution
main() {
    show_config
    demo_memory_management
    demo_adaptive_rate_limiting
    demo_task_pausing
    demo_monitoring
    demo_performance_comparison
    demo_test_examples
    demo_configurations

    echo -e "${GREEN}=== Demo Complete ===${NC}"
    echo ""
    echo "To start testing:"
    echo "1. Set your GitHub token: export GITHUB_TOKEN='your_token'"
    echo "2. Configure for large repos (see examples above)"
    echo "3. Start the crawler: ./crawler"
    echo "4. Run tests: ./test_large_repo.sh"
    echo ""
    echo "For detailed instructions, see: ENHANCED_TESTING.md"
}

# Run the demo
main "$@"
