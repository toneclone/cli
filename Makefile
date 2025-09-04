# ToneClone CLI Makefile

# Build variables
VERSION ?= 1.0.0
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION ?= $(shell go version | awk '{print $$3}')

# Build flags
LDFLAGS := -ldflags "\
	-X github.com/toneclone/cli/cmd.Version=$(VERSION) \
	-X github.com/toneclone/cli/cmd.GitCommit=$(GIT_COMMIT) \
	-X github.com/toneclone/cli/cmd.BuildDate=$(BUILD_DATE) \
	-X github.com/toneclone/cli/cmd.GoVersion=$(GO_VERSION)"

# Build targets
BINARY_NAME := toneclone
BUILD_DIR := ./bin

.PHONY: help build build-all test test-unit test-integration test-coverage clean install dev fmt lint vet mod-tidy

help: ## Show this help message
	@echo "ToneClone CLI Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the CLI binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./main.go

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux amd64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./main.go
	
	# Linux arm64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./main.go
	
	# macOS amd64 (Intel)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./main.go
	
	# macOS arm64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./main.go
	
	# Windows amd64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./main.go
	
	@echo "Built binaries:"
	@ls -la $(BUILD_DIR)/

test: test-unit test-integration ## Run all tests

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	go test -v -race ./internal/... ./pkg/... ./cmd/...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -race ./test/...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./internal/... ./pkg/... ./cmd/... ./test/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-short: ## Run tests in short mode (skip integration tests)
	@echo "Running unit tests (short mode)..."
	go test -v -race -short ./...

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

install: build ## Install the CLI binary
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

dev: ## Build and install for development
	@echo "Building and installing for development..."
	go install $(LDFLAGS) ./main.go

fmt: ## Format Go code
	@echo "Formatting Go code..."
	go fmt ./...

lint: ## Run golint
	@echo "Running golint..."
	@command -v golint >/dev/null 2>&1 || { echo "golint not installed. Install with: go install golang.org/x/lint/golint@latest"; exit 1; }
	golint ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

mod-tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	go mod tidy

mod-download: ## Download go modules
	@echo "Downloading go modules..."
	go mod download

check: fmt vet lint test-short ## Run all checks (format, vet, lint, test)

release-check: check test ## Run comprehensive checks before release

# Development targets
dev-setup: ## Set up development environment
	@echo "Setting up development environment..."
	go mod download
	@echo "Installing development tools..."
	go install golang.org/x/lint/golint@latest
	@echo "Development environment ready!"

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"

# Docker targets (for future use)
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t toneclone-cli:$(VERSION) .

docker-run: ## Run CLI in Docker container
	@echo "Running CLI in Docker..."
	docker run --rm -it toneclone-cli:$(VERSION)

# Release targets
release-build: clean build-all ## Build release binaries
	@echo "Creating release archives..."
	@mkdir -p $(BUILD_DIR)/releases
	
	# Create tar.gz for Unix systems
	tar -czf $(BUILD_DIR)/releases/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-linux-amd64
	tar -czf $(BUILD_DIR)/releases/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-linux-arm64
	tar -czf $(BUILD_DIR)/releases/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-darwin-amd64
	tar -czf $(BUILD_DIR)/releases/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-darwin-arm64
	
	# Create zip for Windows
	zip -j $(BUILD_DIR)/releases/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe
	
	@echo "Release archives created in $(BUILD_DIR)/releases/"
	@ls -la $(BUILD_DIR)/releases/