.PHONY: build test lint clean docker docker-debug run help \
	kind-create kind-delete kind-status kind-load kind-deploy kind-undeploy kind-logs kind-redeploy kind-help kind-test

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

# Run tests with race detection
test-race:
	go test -v -race ./...

# Run tests with coverage (generates HTML report)
test-cover:
	go test -v -coverprofile=coverage.out -coverpkg=./internal/...,./cmd/... ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo ""
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | tail -1

# Run tests with race detection AND coverage
test-all:
	go test -v -race -coverprofile=coverage.out -coverpkg=./internal/...,./cmd/... ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo ""
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | tail -1

# Legacy alias for test-cover (backward compatibility)
test-coverage: test-cover

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Build Docker image
docker:
	docker build -f Dockerfile -t $(BINARY_NAME):$(VERSION) .

# Build Docker debug image
docker-debug:
	docker build -f Dockerfile.debug -t $(BINARY_NAME):$(VERSION)-debug .

# Run locally with example config
# Requires a config.json file in current directory or specify path with --config
run:
	go run ./cmd/mount-monitor --config=config.json --log-level=debug

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary for current platform"
	@echo "  build-linux-amd64 - Build for Linux AMD64"
	@echo "  build-linux-arm64 - Build for Linux ARM64"
	@echo "  build-all       - Build for all platforms"
	@echo "  test            - Run tests"
	@echo "  test-race       - Run tests with race detection"
	@echo "  test-cover      - Run tests with coverage report"
	@echo "  test-all        - Run tests with race detection + coverage"
	@echo "  lint            - Run linter"
	@echo "  clean           - Remove build artifacts"
	@echo "  docker          - Build Docker image"
	@echo "  docker-debug    - Build Docker debug image"
	@echo "  run             - Run locally with example config"
	@echo ""
	@echo "KIND targets (local Kubernetes development):"
	@echo "  kind-create     - Create local KIND cluster"
	@echo "  kind-delete     - Delete local KIND cluster"
	@echo "  kind-status     - Show cluster and pod status"
	@echo "  kind-load       - Build and load image into KIND"
	@echo "  kind-deploy     - Deploy monitor to KIND cluster"
	@echo "  kind-undeploy   - Remove deployment from cluster"
	@echo "  kind-logs       - Tail mount-monitor logs"
	@echo "  kind-redeploy   - Rebuild, reload, and restart"
	@echo "  kind-test       - Run automated e2e watchdog test"
	@echo "  kind-help       - Show KIND workflow help"

# ============================================================================
# KIND Local Development Targets
# ============================================================================

# KIND cluster name (customizable via environment variable)
KIND_CLUSTER_NAME ?= debrid-mount-monitor

# KIND namespace (customizable via environment variable)
KIND_NAMESPACE ?= mount-monitor-dev

# Create local KIND cluster
kind-create:
	@if ! docker info > /dev/null 2>&1; then \
		echo "Error: Docker is not running. Please start Docker first."; \
		exit 1; \
	fi
	@echo "Creating KIND cluster '$(KIND_CLUSTER_NAME)'..."
	kind create cluster --name $(KIND_CLUSTER_NAME) --config deploy/kind-config.yaml
	@echo ""
	@echo "Cluster created successfully!"
	@echo "Run 'make kind-load kind-deploy' to deploy the monitor."

# Delete local KIND cluster
kind-delete:
	@echo "Deleting KIND cluster '$(KIND_CLUSTER_NAME)'..."
	kind delete cluster --name $(KIND_CLUSTER_NAME)
	@echo "Cluster deleted."

# Show cluster and pod status
kind-status:
	@echo "=== KIND Cluster Status ==="
	@kind get clusters 2>/dev/null | grep -q $(KIND_CLUSTER_NAME) && echo "Cluster: $(KIND_CLUSTER_NAME) (running)" || echo "Cluster: $(KIND_CLUSTER_NAME) (not found)"
	@echo ""
	@echo "=== Nodes ==="
	@kubectl get nodes 2>/dev/null || echo "Cannot connect to cluster"
	@echo ""
	@echo "=== Pods in $(KIND_NAMESPACE) namespace ==="
	@kubectl -n $(KIND_NAMESPACE) get pods 2>/dev/null || echo "Namespace not found or no pods"

# Build image and load into KIND cluster
kind-load: docker
	@echo "Loading image into KIND cluster '$(KIND_CLUSTER_NAME)'..."
	kind load docker-image $(BINARY_NAME):$(VERSION) --name $(KIND_CLUSTER_NAME)
	@echo "Image loaded successfully."

# Deploy monitor to KIND cluster
# Note: Replaces hardcoded namespace in manifests with $(KIND_NAMESPACE)
kind-deploy:
	@echo "Deploying to KIND cluster (namespace: $(KIND_NAMESPACE))..."
	@kubectl create namespace $(KIND_NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	@for file in deploy/kind/*.yaml; do \
		sed 's/namespace: mount-monitor-dev/namespace: $(KIND_NAMESPACE)/g' "$$file" | kubectl apply -f -; \
	done
	@echo ""
	@echo "Waiting for pods to be ready..."
	kubectl -n $(KIND_NAMESPACE) wait --for=condition=ready pod -l app=test-app-with-monitor --timeout=60s || true
	@echo ""
	@echo "Deployment complete!"
	@echo "  View logs: make kind-logs"
	@echo "  Test health: curl localhost:30080/healthz/ready"

# Remove deployment from cluster
kind-undeploy:
	@echo "Removing deployment from $(KIND_NAMESPACE) namespace..."
	kubectl delete namespace $(KIND_NAMESPACE) --ignore-not-found
	@echo "Deployment removed."

# Tail mount-monitor container logs
kind-logs:
	kubectl -n $(KIND_NAMESPACE) logs -l app=test-app-with-monitor -c mount-monitor -f

# Rebuild, reload, and restart deployment (quick iteration)
kind-redeploy: docker
	@echo "Rebuilding and redeploying to $(KIND_NAMESPACE)..."
	kind load docker-image $(BINARY_NAME):$(VERSION) --name $(KIND_CLUSTER_NAME)
	kubectl -n $(KIND_NAMESPACE) rollout restart deployment/test-app-with-monitor
	@echo ""
	@echo "Waiting for rollout to complete..."
	kubectl -n $(KIND_NAMESPACE) rollout status deployment/test-app-with-monitor --timeout=60s
	@echo ""
	@echo "Redeploy complete!"

# Show KIND workflow help
kind-help:
	@echo "KIND Local Development Workflow"
	@echo "================================"
	@echo ""
	@echo "Quick Start:"
	@echo "  make kind-create kind-load kind-deploy"
	@echo ""
	@echo "View logs:"
	@echo "  make kind-logs"
	@echo ""
	@echo "After code changes:"
	@echo "  make kind-redeploy"
	@echo ""
	@echo "Simulate mount failure (using current namespace: $(KIND_NAMESPACE)):"
	@echo "  POD=\$$(kubectl -n $(KIND_NAMESPACE) get pod -l app=test-app-with-monitor -o name)"
	@echo "  kubectl -n $(KIND_NAMESPACE) exec \$$POD -c main-app -- rm /mnt/test/.health-check"
	@echo ""
	@echo "Restore mount health:"
	@echo "  kubectl -n $(KIND_NAMESPACE) exec \$$POD -c main-app -- sh -c 'echo healthy > /mnt/test/.health-check'"
	@echo ""
	@echo "Cleanup:"
	@echo "  make kind-delete"
	@echo ""
	@echo "Environment variables:"
	@echo "  KIND_CLUSTER_NAME  - Cluster name (default: debrid-mount-monitor)"
	@echo "  KIND_NAMESPACE     - Deployment namespace (default: mount-monitor-dev)"
	@echo "  KEEP_CLUSTER       - Set to 1 to preserve cluster after kind-test"
	@echo ""
	@echo "Deploy to custom namespace:"
	@echo "  KIND_NAMESPACE=my-namespace make kind-deploy"

# Run automated e2e watchdog test
# Creates temporary cluster, tests pod restart behavior, cleans up
kind-test:
	@echo "Running automated e2e watchdog test..."
	@KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) KIND_NAMESPACE=$(KIND_NAMESPACE) ./scripts/kind-e2e-test.sh
