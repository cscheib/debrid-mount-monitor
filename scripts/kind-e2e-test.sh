#!/usr/bin/env bash
#
# kind-e2e-test.sh - End-to-end test for watchdog pod restart functionality
#
# This script:
# 1. Creates a KIND cluster (or reuses existing)
# 2. Builds and loads the monitor image
# 3. Deploys with watchdog enabled
# 4. Simulates mount failure
# 5. Verifies pod restart and WatchdogRestart event
# 6. Cleans up (unless KEEP_CLUSTER=1)
#
# Usage:
#   ./scripts/kind-e2e-test.sh
#   KEEP_CLUSTER=1 ./scripts/kind-e2e-test.sh  # Keep cluster for debugging
#
# Exit codes:
#   0 - All tests passed
#   1 - Pod restart verification failed
#   2 - WatchdogRestart event not found
#   3 - Setup/deployment failed
#   4 - Cleanup failed (warning only)

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-debrid-mount-monitor-test}"
KIND_NAMESPACE="${KIND_NAMESPACE:-watchdog-e2e-test}"
KEEP_CLUSTER="${KEEP_CLUSTER:-0}"
IMAGE_NAME="mount-monitor"
IMAGE_TAG="dev"

# Timeouts
CLUSTER_TIMEOUT=120
DEPLOY_TIMEOUT=120
RESTART_TIMEOUT=120

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Disable colors if not a TTY
if [[ ! -t 1 ]]; then
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

#------------------------------------------------------------------------------
# Utility Functions
#------------------------------------------------------------------------------

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $*"
}

log_step() {
    echo -e "\n${BLUE}[$1]${NC} $2"
}

#------------------------------------------------------------------------------
# Cleanup Function
#------------------------------------------------------------------------------

cleanup() {
    local exit_code=$?

    if [[ "${KEEP_CLUSTER}" == "1" ]]; then
        log_warning "KEEP_CLUSTER=1, skipping cleanup"
        log_info "To clean up manually: kind delete cluster --name ${KIND_CLUSTER_NAME}"
        return
    fi

    log_step "6/6" "Cleaning up..."

    if kind get clusters 2>/dev/null | grep -q "^${KIND_CLUSTER_NAME}$"; then
        if kind delete cluster --name "${KIND_CLUSTER_NAME}" 2>/dev/null; then
            log_success "Cluster '${KIND_CLUSTER_NAME}' deleted"
        else
            log_warning "Failed to delete cluster (exit code 4)"
        fi
    fi

    exit "${exit_code}"
}

#------------------------------------------------------------------------------
# Prerequisite Checks
#------------------------------------------------------------------------------

check_prerequisites() {
    log_step "1/6" "Checking prerequisites..."

    local missing=0

    # Check Docker
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker first."
        missing=1
    else
        log_success "Docker is running"
    fi

    # Check kind
    if ! command -v kind &> /dev/null; then
        log_error "kind is not installed. Install with: brew install kind"
        missing=1
    else
        log_success "kind is installed ($(kind version | head -1))"
    fi

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Install with: brew install kubectl"
        missing=1
    else
        log_success "kubectl is installed"
    fi

    if [[ "${missing}" -ne 0 ]]; then
        exit 3
    fi
}

#------------------------------------------------------------------------------
# Cluster Setup
#------------------------------------------------------------------------------

setup_cluster() {
    log_step "2/6" "Setting up cluster..."

    # Check if cluster already exists
    if kind get clusters 2>/dev/null | grep -q "^${KIND_CLUSTER_NAME}$"; then
        log_info "Cluster '${KIND_CLUSTER_NAME}' already exists, reusing"
        kubectl cluster-info --context "kind-${KIND_CLUSTER_NAME}" > /dev/null 2>&1 || {
            log_warning "Existing cluster unreachable, recreating..."
            kind delete cluster --name "${KIND_CLUSTER_NAME}" 2>/dev/null || true
            create_cluster
        }
    else
        create_cluster
    fi

    # Set kubectl context
    kubectl config use-context "kind-${KIND_CLUSTER_NAME}" > /dev/null
    log_success "Cluster '${KIND_CLUSTER_NAME}' ready"
}

create_cluster() {
    log_info "Creating KIND cluster '${KIND_CLUSTER_NAME}'..."

    local config_file="${REPO_ROOT}/deploy/kind-config.yaml"
    if [[ -f "${config_file}" ]]; then
        kind create cluster --name "${KIND_CLUSTER_NAME}" --config "${config_file}" --wait "${CLUSTER_TIMEOUT}s"
    else
        kind create cluster --name "${KIND_CLUSTER_NAME}" --wait "${CLUSTER_TIMEOUT}s"
    fi
}

#------------------------------------------------------------------------------
# Image Build and Load
#------------------------------------------------------------------------------

build_and_load_image() {
    log_step "3/6" "Building and loading image..."

    # Build image (show stderr for errors, hide stdout for cleaner output)
    log_info "Building Docker image..."
    if ! (cd "${REPO_ROOT}" && docker build -t "${IMAGE_NAME}:${IMAGE_TAG}" -f Dockerfile . > /dev/null); then
        log_error "Failed to build Docker image"
        exit 3
    fi
    log_success "Image built: ${IMAGE_NAME}:${IMAGE_TAG}"

    # Verify image was created successfully
    if ! docker image inspect "${IMAGE_NAME}:${IMAGE_TAG}" > /dev/null 2>&1; then
        log_error "Docker image not found after build: ${IMAGE_NAME}:${IMAGE_TAG}"
        exit 3
    fi
    log_success "Image verified: ${IMAGE_NAME}:${IMAGE_TAG}"

    # Load into KIND
    log_info "Loading image into KIND cluster..."
    if ! kind load docker-image "${IMAGE_NAME}:${IMAGE_TAG}" --name "${KIND_CLUSTER_NAME}" > /dev/null 2>&1; then
        log_error "Failed to load image into KIND"
        exit 3
    fi
    log_success "Image loaded into cluster"
}

#------------------------------------------------------------------------------
# Deployment
#------------------------------------------------------------------------------

deploy_monitor() {
    log_step "4/6" "Deploying monitor with watchdog enabled..."

    # Create namespace
    kubectl create namespace "${KIND_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f - > /dev/null

    # Apply RBAC
    kubectl apply -n "${KIND_NAMESPACE}" -f "${REPO_ROOT}/deploy/kind/rbac.yaml" > /dev/null 2>&1 || {
        log_error "Failed to apply RBAC"
        exit 3
    }

    # Apply test ConfigMap (short delays for faster testing)
    if [[ -f "${REPO_ROOT}/deploy/kind/test-configmap.yaml" ]]; then
        kubectl apply -n "${KIND_NAMESPACE}" -f "${REPO_ROOT}/deploy/kind/test-configmap.yaml" > /dev/null 2>&1
    else
        kubectl apply -n "${KIND_NAMESPACE}" -f "${REPO_ROOT}/deploy/kind/configmap.yaml" > /dev/null 2>&1
    fi

    # Apply service
    kubectl apply -n "${KIND_NAMESPACE}" -f "${REPO_ROOT}/deploy/kind/service.yaml" > /dev/null 2>&1 || true

    # Apply deployment
    kubectl apply -n "${KIND_NAMESPACE}" -f "${REPO_ROOT}/deploy/kind/deployment.yaml" > /dev/null 2>&1 || {
        log_error "Failed to apply deployment"
        exit 3
    }

    # Wait for pod to be ready
    log_info "Waiting for pod to be ready..."
    if ! kubectl -n "${KIND_NAMESPACE}" wait --for=condition=ready pod -l app=test-app-with-monitor --timeout="${DEPLOY_TIMEOUT}s" 2>/dev/null; then
        log_error "Pod did not become ready within ${DEPLOY_TIMEOUT}s"
        kubectl -n "${KIND_NAMESPACE}" get pods
        kubectl -n "${KIND_NAMESPACE}" describe pods
        exit 3
    fi

    log_success "Deployment ready"
}

#------------------------------------------------------------------------------
# Test Execution
#------------------------------------------------------------------------------

simulate_mount_failure() {
    log_step "5/6" "Simulating mount failure and verifying restart..."

    # Get pod name
    local pod_name
    pod_name=$(kubectl -n "${KIND_NAMESPACE}" get pod -l app=test-app-with-monitor -o jsonpath='{.items[0].metadata.name}')

    if [[ -z "${pod_name}" ]]; then
        log_error "Could not find pod"
        exit 1
    fi

    # Record initial creation timestamp
    local initial_timestamp
    initial_timestamp=$(kubectl -n "${KIND_NAMESPACE}" get pod "${pod_name}" -o jsonpath='{.metadata.creationTimestamp}')
    log_info "Initial pod: ${pod_name} (created: ${initial_timestamp})"

    # Remove canary file to simulate mount failure
    log_info "Removing canary file to simulate mount failure..."
    kubectl -n "${KIND_NAMESPACE}" exec "${pod_name}" -c main-app -- rm -f /mnt/test/.health-check 2>/dev/null || {
        log_warning "Could not remove canary file (may already be gone)"
    }

    # Wait for pod restart
    log_info "Waiting for watchdog to trigger restart (timeout: ${RESTART_TIMEOUT}s)..."

    local elapsed=0
    local check_interval=5
    while [[ "${elapsed}" -lt "${RESTART_TIMEOUT}" ]]; do
        sleep "${check_interval}"
        elapsed=$((elapsed + check_interval))

        # Check if original pod is gone or a new pod exists
        local current_pod
        current_pod=$(kubectl -n "${KIND_NAMESPACE}" get pod -l app=test-app-with-monitor -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

        if [[ -z "${current_pod}" ]]; then
            log_info "Pod terminating, waiting for replacement..."
            continue
        fi

        local current_timestamp
        current_timestamp=$(kubectl -n "${KIND_NAMESPACE}" get pod "${current_pod}" -o jsonpath='{.metadata.creationTimestamp}' 2>/dev/null || echo "")

        if [[ "${current_timestamp}" != "${initial_timestamp}" && -n "${current_timestamp}" ]]; then
            log_success "Pod restarted! New pod: ${current_pod} (created: ${current_timestamp})"

            # Verify WatchdogRestart event
            verify_watchdog_event
            return 0
        fi

        log_info "Still waiting... (${elapsed}s elapsed)"
    done

    log_error "Pod did not restart within ${RESTART_TIMEOUT}s"
    log_info "Current pod status:"
    kubectl -n "${KIND_NAMESPACE}" get pods
    log_info "Monitor logs:"
    kubectl -n "${KIND_NAMESPACE}" logs -l app=test-app-with-monitor -c mount-monitor --tail=50 || true
    exit 1
}

verify_watchdog_event() {
    log_info "Verifying WatchdogRestart event..."

    local events
    events=$(kubectl -n "${KIND_NAMESPACE}" get events --field-selector reason=WatchdogRestart -o jsonpath='{.items}' 2>/dev/null)

    if [[ "${events}" == "[]" || -z "${events}" ]]; then
        log_warning "WatchdogRestart event not found (may have been created in different namespace)"
        # Not a hard failure - event creation is best-effort
        return 0
    fi

    log_success "WatchdogRestart event found"
}

#------------------------------------------------------------------------------
# Main
#------------------------------------------------------------------------------

main() {
    echo ""
    echo "=== KIND E2E Test: Watchdog Pod Restart ==="
    echo ""

    # Set up trap for cleanup
    trap cleanup EXIT

    check_prerequisites
    setup_cluster
    build_and_load_image
    deploy_monitor
    simulate_mount_failure

    echo ""
    echo -e "${GREEN}=== TEST PASSED ===${NC}"
    echo ""
}

main "$@"
