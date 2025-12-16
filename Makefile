.PHONY: build test lint clean docker docker-debug run help \
	kind-create kind-delete kind-status kind-load kind-deploy kind-undeploy kind-logs kind-redeploy kind-help

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
	docker build -f Dockerfile -t $(BINARY_NAME):$(VERSION) .

# Build Docker debug image
docker-debug:
	docker build -f Dockerfile.debug -t $(BINARY_NAME):$(VERSION)-debug .

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
	@echo "  kind-help       - Show KIND workflow help"

# ============================================================================
# KIND Local Development Targets
# ============================================================================

# KIND cluster name (customizable via environment variable)
KIND_CLUSTER_NAME ?= debrid-mount-monitor

# Create local KIND cluster
kind-create:
	@if ! docker info > /dev/null 2>&1; then \
		echo "Error: Docker is not running. Please start Docker first."; \
		exit 1; \
	fi
	@echo "Creating KIND cluster '$(KIND_CLUSTER_NAME)'..."
	kind create cluster --name $(KIND_CLUSTER_NAME) --config deploy/kind/kind-config.yaml
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
	@echo "=== Pods in mount-monitor-dev namespace ==="
	@kubectl -n mount-monitor-dev get pods 2>/dev/null || echo "Namespace not found or no pods"

# Build image and load into KIND cluster
kind-load: docker
	@echo "Loading image into KIND cluster '$(KIND_CLUSTER_NAME)'..."
	kind load docker-image $(BINARY_NAME):$(VERSION) --name $(KIND_CLUSTER_NAME)
	@echo "Image loaded successfully."

# Deploy monitor to KIND cluster
kind-deploy:
	@echo "Deploying to KIND cluster..."
	kubectl apply -f deploy/kind/namespace.yaml
	kubectl apply -f deploy/kind/configmap.yaml
	kubectl apply -f deploy/kind/deployment.yaml
	kubectl apply -f deploy/kind/service.yaml
	@echo ""
	@echo "Waiting for pods to be ready..."
	kubectl -n mount-monitor-dev wait --for=condition=ready pod -l app=test-app-with-monitor --timeout=60s || true
	@echo ""
	@echo "Deployment complete!"
	@echo "  View logs: make kind-logs"
	@echo "  Test health: curl localhost:30080/healthz/ready"

# Remove deployment from cluster
kind-undeploy:
	@echo "Removing deployment from cluster..."
	kubectl delete namespace mount-monitor-dev --ignore-not-found
	@echo "Deployment removed."

# Tail mount-monitor container logs
kind-logs:
	kubectl -n mount-monitor-dev logs -l app=test-app-with-monitor -c mount-monitor -f

# Rebuild, reload, and restart deployment (quick iteration)
kind-redeploy: docker
	@echo "Rebuilding and redeploying..."
	kind load docker-image $(BINARY_NAME):$(VERSION) --name $(KIND_CLUSTER_NAME)
	kubectl -n mount-monitor-dev rollout restart deployment/test-app-with-monitor
	@echo ""
	@echo "Waiting for rollout to complete..."
	kubectl -n mount-monitor-dev rollout status deployment/test-app-with-monitor --timeout=60s
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
	@echo "Simulate mount failure:"
	@echo "  POD=\$$(kubectl -n mount-monitor-dev get pod -l app=test-app-with-monitor -o name)"
	@echo "  kubectl -n mount-monitor-dev exec \$$POD -c main-app -- rm /mnt/test/.health-check"
	@echo ""
	@echo "Restore mount health:"
	@echo "  kubectl -n mount-monitor-dev exec \$$POD -c main-app -- sh -c 'echo healthy > /mnt/test/.health-check'"
	@echo ""
	@echo "Cleanup:"
	@echo "  make kind-delete"
	@echo ""
	@echo "Environment variables:"
	@echo "  KIND_CLUSTER_NAME  - Cluster name (default: debrid-mount-monitor)"
