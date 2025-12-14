.PHONY: build test lint clean docker docker-debug run help

# Build variables
BINARY_NAME=mount-monitor
VERSION?=dev
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

# Default target
all: test build

# Build the binary
build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/mount-monitor

# Build for Linux AMD64
build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/mount-monitor

# Build for Linux ARM64
build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/mount-monitor

# Build all platforms
build-all: build-linux-amd64 build-linux-arm64

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Build Docker image
docker:
	docker build -f build/Dockerfile -t $(BINARY_NAME):$(VERSION) .

# Build Docker debug image
docker-debug:
	docker build -f build/Dockerfile.debug -t $(BINARY_NAME):$(VERSION)-debug .

# Run locally with example config
run:
	MOUNT_PATHS=/tmp LOG_LEVEL=debug go run ./cmd/mount-monitor

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary for current platform"
	@echo "  build-linux-amd64 - Build for Linux AMD64"
	@echo "  build-linux-arm64 - Build for Linux ARM64"
	@echo "  build-all       - Build for all platforms"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  lint            - Run linter"
	@echo "  clean           - Remove build artifacts"
	@echo "  docker          - Build Docker image"
	@echo "  docker-debug    - Build Docker debug image"
	@echo "  run             - Run locally with example config"
