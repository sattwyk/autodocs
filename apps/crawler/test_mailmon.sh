#!/bin/bash

# Test script specifically for mailmon repository
set -e

echo "=== Testing Mailmon Repository ==="

# Check if crawler is running
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "❌ Crawler is not running"
    exit 1
fi

echo "✅ Crawler is running"

# Create results directory
mkdir -p mailmon_test_results

# Test mailmon repository
echo "🔍 Testing mailmon repository..."
echo "URL: https://github.com/sattwyk/mailmon.git"

# Make the request with longer timeout
echo "📡 Making API request..."
response=$(curl -m 300 -s -w "\n%{http_code}\n%{time_total}" \
    -X POST http://localhost:8080/invoke \
    -H "Content-Type: application/json" \
    -d '{
        "repo_url": "https://github.com/sattwyk/mailmon.git",
        "ref": "main"
    }')

# Parse response
http_code=$(echo "$response" | tail -n 2 | head -n 1)
time_total=$(echo "$response" | tail -n 1)
json_response=$(echo "$response" | head -n -2)

echo "📊 Results:"
echo "HTTP Code: $http_code"
echo "Time: ${time_total}s"

if [ "$http_code" = "200" ]; then
    echo "✅ Request successful!"

    # Save the full JSON response
    echo "$json_response" > mailmon_test_results/mailmon_full_response.json

    # Parse and display summary if jq is available
    if command -v jq > /dev/null 2>&1; then
        echo ""
        echo "📈 Summary:"
        total_files=$(echo "$json_response" | jq -r '.total_files // "N/A"')
        processed_files=$(echo "$json_response" | jq -r '.processed_files // "N/A"')
        skipped_files=$(echo "$json_response" | jq -r '.skipped_files // "N/A"')
        error_count=$(echo "$json_response" | jq -r '.errors | length // "N/A"')
        duration=$(echo "$json_response" | jq -r '.duration // "N/A"')

        echo "Total files: $total_files"
        echo "Processed files: $processed_files"
        echo "Skipped files: $skipped_files"
        echo "Errors: $error_count"
        echo "Duration: $duration"

        # Show first few files with content
        echo ""
        echo "📁 Sample files (first 3):"
        echo "$json_response" | jq -r '.files[:3] | .[] | "- \(.path) (\(.size) bytes)"'

        # Save a formatted version
        echo "$json_response" | jq . > mailmon_test_results/mailmon_formatted.json

        echo ""
        echo "💾 Full results saved to:"
        echo "  - mailmon_test_results/mailmon_full_response.json (raw)"
        echo "  - mailmon_test_results/mailmon_formatted.json (formatted)"

    else
        echo "Raw response saved to: mailmon_test_results/mailmon_full_response.json"
    fi
else
    echo "❌ Request failed"
    echo "Response: $json_response"
fi

echo ""
echo "�� Test completed!"
