# GitHub Crawler Service Makefile

.PHONY: build run test clean docker-build docker-run help deps fmt lint

# Variables
BINARY_NAME=crawler
DOCKER_IMAGE=crawler:latest
MAIN_PATH=./cmd/crawler

# Default target
.DEFAULT_GOAL := help

## Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) $(MAIN_PATH)

## Run the application (requires GITHUB_TOKEN)
run: build
	@echo "Starting $(BINARY_NAME)..."
	@./$(BINARY_NAME)

## Run with sample token (for testing without real token)
run-dev: build
	@echo "Starting $(BINARY_NAME) in development mode..."
	GITHUB_TOKEN=fake_token_for_testing ./$(BINARY_NAME)

## Run tests
test:
	@echo "Running Go tests..."
	go test -v ./...

## Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## Run unit tests only
test-unit:
	@echo "Running unit tests..."
	go test -v -short ./...

## Run integration tests only
test-integration-go:
	@echo "Running Go integration tests..."
	go test -v -run TestIntegration ./...

## Run benchmarks
test-bench:
	@echo "Running benchmarks..."
	go test -v -bench=. -benchmem ./...

## Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -v -race ./...

## Run all test suites
test-all: test-unit test-integration-go test-bench test-coverage
	@echo "All tests completed!"

## Run integration tests (requires service to be running)
test-integration:
	@echo "Running integration tests..."
	@./test.sh

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

## Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, running go vet instead"; \
		go vet ./...; \
	fi

## Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	go clean

## Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

## Run Docker container
docker-run: docker-build
	@echo "Running Docker container..."
	docker run -p 8080:8080 \
		-e GITHUB_TOKEN="$(GITHUB_TOKEN)" \
		$(DOCKER_IMAGE)

## Run with Docker Compose (if docker-compose.yml exists)
docker-compose-up:
	@if [ -f docker-compose.yml ]; then \
		docker-compose up --build; \
	else \
		echo "docker-compose.yml not found"; \
	fi

## Health check
health:
	@echo "Checking service health..."
	@curl -s http://localhost:8080/health | jq '.' || \
		echo "Service not running or jq not installed"

## Show metrics
metrics:
	@echo "Fetching metrics..."
	@curl -s http://localhost:8080/metrics | grep "^crawler_" | head -20 || \
		echo "Service not running or no metrics available"

## Performance test with a small repository
perf-test:
	@echo "Running performance test..."
	@if [ -z "$(GITHUB_TOKEN)" ]; then \
		echo "Error: GITHUB_TOKEN required for performance test"; \
		exit 1; \
	fi
	time curl -X POST http://localhost:8080/invoke \
		-H "Content-Type: application/json" \
		-d '{"repo_url": "https://github.com/octocat/Hello-World.git", "ref": "main"}' \
		| jq '.'

## Generate load (requires service running)
load-test:
	@echo "Generating load (10 concurrent requests)..."
	@for i in {1..10}; do \
		curl -s http://localhost:8080/health > /dev/null & \
	done; \
	wait; \
	echo "Load test completed"

## Show logs (if running in systemd)
logs:
	@if systemctl is-active --quiet crawler; then \
		journalctl -u crawler -f; \
	else \
		echo "Service not running in systemd"; \
	fi

## Install as systemd service (requires sudo)
install-service:
	@echo "Installing systemd service..."
	@sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "[Unit]" | sudo tee /etc/systemd/system/crawler.service
	@echo "Description=GitHub Crawler Service" | sudo tee -a /etc/systemd/system/crawler.service
	@echo "After=network.target" | sudo tee -a /etc/systemd/system/crawler.service
	@echo "" | sudo tee -a /etc/systemd/system/crawler.service
	@echo "[Service]" | sudo tee -a /etc/systemd/system/crawler.service
	@echo "Type=simple" | sudo tee -a /etc/systemd/system/crawler.service
	@echo "User=crawler" | sudo tee -a /etc/systemd/system/crawler.service
	@echo "ExecStart=/usr/local/bin/$(BINARY_NAME)" | sudo tee -a /etc/systemd/system/crawler.service
	@echo "Restart=always" | sudo tee -a /etc/systemd/system/crawler.service
	@echo "Environment=GITHUB_TOKEN=$(GITHUB_TOKEN)" | sudo tee -a /etc/systemd/system/crawler.service
	@echo "" | sudo tee -a /etc/systemd/system/crawler.service
	@echo "[Install]" | sudo tee -a /etc/systemd/system/crawler.service
	@echo "WantedBy=multi-user.target" | sudo tee -a /etc/systemd/system/crawler.service
	@sudo systemctl daemon-reload
	@sudo systemctl enable crawler
	@echo "Service installed. Start with: sudo systemctl start crawler"

## Show help
help:
	@echo "GitHub Crawler Service - Available targets:"
	@echo ""
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/##//g' | sort

## Development workflow
dev: deps fmt lint build test
	@echo "Development workflow completed!"

## Release workflow
release: clean deps fmt lint test build
	@echo "Release build completed!"
	@echo "Binary: $(BINARY_NAME)"
	@ls -la $(BINARY_NAME)

## Show version and build info
version:
	@echo "Go version: $(shell go version)"
	@echo "Build target: $(BINARY_NAME)"
	@echo "Main path: $(MAIN_PATH)"
	@if [ -f $(BINARY_NAME) ]; then \
		echo "Binary size: $(shell ls -lah $(BINARY_NAME) | awk '{print $$5}')"; \
		echo "Binary modified: $(shell ls -la $(BINARY_NAME) | awk '{print $$6" "$$7" "$$8}')"; \
	fi
