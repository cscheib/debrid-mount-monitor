# Development Dockerfile - Compiles from Source
# =============================================
# This Dockerfile is for LOCAL DEVELOPMENT only. It compiles Go source code.
#
# For release builds, see: Dockerfile.release (uses pre-built binaries)
# The release workflow uses Dockerfile.release to avoid redundant compilation.
#
# Usage:
#   make docker          # Uses this Dockerfile
#   make docker-debug    # Uses Dockerfile.debug

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o mount-monitor ./cmd/mount-monitor

# Production stage - scratch for minimal image
FROM scratch

# Copy binary from builder
COPY --from=builder --chmod=755 /app/mount-monitor /mount-monitor

# Expose health check port
EXPOSE 8080

# Run as non-root (numeric UID for scratch compatibility)
USER 65534

ENTRYPOINT ["/mount-monitor"]
