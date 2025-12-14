# Research: Mount Health Monitor

**Date**: 2025-12-14
**Feature**: 001-mount-health-monitor

## Technology Decisions

### 1. Programming Language: Go

**Decision**: Go 1.21+

**Rationale**:
- Native cross-compilation: `GOOS=linux GOARCH=arm64` requires no additional tooling
- Static binary production: `CGO_ENABLED=0` produces fully static binaries
- Standard library completeness: HTTP server, signal handling, JSON logging all built-in
- Fast compilation: Full rebuild in seconds
- Memory efficiency: Low baseline memory (~5-10MB for this workload)
- Go 1.21+ provides `log/slog` for structured logging without external dependencies

**Alternatives Considered**:
- **Rust**: Excellent for static binaries, but steeper learning curve and slower compile times. Overkill for this simple service.
- **C**: Maximum control, but manual memory management adds risk for minimal benefit. No standard HTTP server.
- **Zig**: Promising but ecosystem less mature. Harder to find contributors.

### 2. HTTP Server: net/http (Standard Library)

**Decision**: Use Go's `net/http` package directly

**Rationale**:
- Zero dependencies
- Sufficient for simple probe endpoints (GET /healthz/live, GET /healthz/ready)
- Built-in graceful shutdown via `http.Server.Shutdown()`
- Performance far exceeds requirements (<100ms response time)

**Alternatives Considered**:
- **Chi/Gin/Echo**: Add routing convenience but introduce dependencies. Not needed for 2 endpoints.
- **fasthttp**: Higher performance but non-standard API. Overkill for probe endpoints.

### 3. Logging: log/slog (Standard Library)

**Decision**: Use Go 1.21's `log/slog` package

**Rationale**:
- Structured logging (JSON format) built into standard library
- Zero dependencies
- Leveled logging (Debug, Info, Warn, Error)
- Context-aware logging for request tracing

**Alternatives Considered**:
- **zerolog/zap**: More features but add dependencies. slog is sufficient.
- **log (old)**: No structured output, would require custom formatting.

### 4. Configuration: Environment Variables + Flags

**Decision**: Parse configuration from environment variables with flag overrides

**Rationale**:
- Constitution requires env var + flag support (no config files required)
- `os.Getenv()` and `flag` package are standard library
- Simple precedence: flags override env vars override defaults

**Configuration Parameters**:
| Name | Env Var | Flag | Default | Description |
|------|---------|------|---------|-------------|
| Mount paths | `MOUNT_PATHS` | `--mount-paths` | (required) | Comma-separated mount paths |
| Canary file | `CANARY_FILE` | `--canary-file` | `.health-check` | Relative path within each mount |
| Check interval | `CHECK_INTERVAL` | `--check-interval` | `30s` | Time between health checks |
| Read timeout | `READ_TIMEOUT` | `--read-timeout` | `5s` | Timeout for canary file read |
| Debounce threshold | `DEBOUNCE_THRESHOLD` | `--debounce-threshold` | `3` | Consecutive failures before unhealthy |
| HTTP port | `HTTP_PORT` | `--http-port` | `8080` | Port for probe endpoints |
| Log level | `LOG_LEVEL` | `--log-level` | `info` | Logging verbosity |
| Log format | `LOG_FORMAT` | `--log-format` | `json` | Output format (json/text) |

### 5. Health Check Strategy: Canary File Read with Timeout

**Decision**: Read a small file within each mount path with a configurable timeout

**Rationale**:
- More robust than `os.Stat()` which can return success on stale FUSE mounts
- Timeout detection catches hung mounts that would block indefinitely
- Simple implementation: `os.ReadFile()` with `context.WithTimeout()`

**Implementation Notes**:
- Use goroutine + channel pattern for timeout enforcement
- Read entire file (expected to be small, <1KB)
- Any error = unhealthy (not found, permission, timeout, I/O error)

### 6. Debounce Strategy: Consecutive Failure Counter

**Decision**: Track consecutive failures per mount, threshold before marking unhealthy for liveness

**Rationale**:
- Prevents pod restart from brief network glitches
- Configurable threshold allows tuning per environment
- Readiness probe fails immediately (any failure), liveness requires debounce

**State Machine**:
```
HEALTHY -> (check fails) -> DEGRADED (count=1)
DEGRADED -> (check fails) -> DEGRADED (count++)
DEGRADED -> (count >= threshold) -> UNHEALTHY
DEGRADED -> (check passes) -> HEALTHY (count=0)
UNHEALTHY -> (check passes) -> HEALTHY (count=0)
```

### 7. Container Base Image: scratch

**Decision**: Use `scratch` (empty) base image for production

**Rationale**:
- Smallest possible image size (<10MB total)
- No shell, no utilities = minimal attack surface
- Go static binary runs without any runtime dependencies

**Build Strategy**:
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o mount-monitor ./cmd/mount-monitor

# Production stage
FROM scratch
COPY --from=builder /app/mount-monitor /mount-monitor
ENTRYPOINT ["/mount-monitor"]
```

### 8. Multi-Architecture Build: Docker Buildx

**Decision**: Use Docker Buildx for multi-arch manifests

**Rationale**:
- Single command builds for linux/amd64 and linux/arm64
- Pushes manifest list for automatic architecture selection
- Native to modern Docker, no additional tools needed

**Build Command**:
```bash
docker buildx build --platform linux/amd64,linux/arm64 -t ghcr.io/user/mount-monitor:latest --push .
```

## Open Questions Resolved

All NEEDS CLARIFICATION items from spec have been resolved:
1. ✅ Restart mechanism → Kubernetes probes (clarification session)
2. ✅ Stale mount detection → Canary file read (clarification session)
3. ✅ Read timeout → Configurable, default 5s (clarification session)

## Dependencies Summary

**Runtime Dependencies**: None (static binary)

**Build Dependencies**:
- Go 1.21+ compiler
- Docker (for container builds)
- Docker Buildx (for multi-arch)

**Test Dependencies**: None (standard library `testing` package)
