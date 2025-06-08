#!/bin/bash

# Basic test script for the enhanced crawler
set -e

echo "=== Basic Crawler Test ==="

# Check if crawler is running
echo "Checking if crawler is running..."
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "✓ Crawler is running"
else
    echo "✗ Crawler is not running"
    echo "Please start the crawler with: GITHUB_TOKEN=your_token ./crawler"
    exit 1
fi

# Test health endpoint
echo "Testing health endpoint..."
health_response=$(curl -s http://localhost:8080/health)
echo "Health response: $health_response"

# Test metrics endpoint
echo "Testing metrics endpoint..."
metrics_response=$(curl -s http://localhost:8080/metrics | head -10)
echo "Metrics (first 10 lines):"
echo "$metrics_response"

# Test basic info endpoint
echo "Testing info endpoint..."
info_response=$(curl -s http://localhost:8080/)
echo "Info response: $info_response"

echo "Basic tests completed successfully!"
