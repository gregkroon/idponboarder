# Harness Onboarder Makefile

# Build variables
BINARY_NAME=harness-onboarder
VERSION?=1.0.0
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"
BUILD_DIR=build

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Default target
.PHONY: all
all: test build

# Build the binary
.PHONY: build
build:
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Build for multiple platforms
.PHONY: build-all
build-all: clean
	# Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	
	# macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	
	# Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Download dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run linting
.PHONY: lint
lint:
	golangci-lint run

# Format code
.PHONY: fmt
fmt:
	$(GOCMD) fmt ./...

# Verify modules
.PHONY: verify
verify:
	$(GOMOD) verify

# Run the application locally
.PHONY: run
run:
	$(GOCMD) run . --dry-run --log-level=debug

# Install to GOPATH/bin
.PHONY: install
install:
	$(GOCMD) install $(LDFLAGS) .

# Create example config if it doesn't exist
.PHONY: init-config
init-config:
	@if [ ! -f config.yaml ]; then \
		cp config.example.yaml config.yaml; \
		echo "Created config.yaml from example. Please edit it with your settings."; \
	else \
		echo "config.yaml already exists"; \
	fi

# Docker build
.PHONY: docker-build
docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

# Release preparation
.PHONY: release-prep
release-prep: clean test build-all
	cd $(BUILD_DIR) && \
	sha256sum * > SHA256SUMS

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Run tests and build"
	@echo "  build        - Build binary for current platform"
	@echo "  build-all    - Build binaries for all platforms"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  lint         - Run linting"
	@echo "  fmt          - Format code"
	@echo "  verify       - Verify modules"
	@echo "  run          - Run application locally (dry-run mode)"
	@echo "  install      - Install to GOPATH/bin"
	@echo "  init-config  - Create config.yaml from example"
	@echo "  docker-build - Build Docker image"
	@echo "  release-prep - Prepare release artifacts"
	@echo "  help         - Show this help"