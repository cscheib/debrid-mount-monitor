# debrid-mount-monitor Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-12-14

## Active Technologies
- Go 1.21+ (required for log/slog structured logging) + Standard library only (encoding/json, os, flag, path/filepath) (002-json-config)
- N/A (configuration file is read-only input) (002-json-config)
- Go 1.21+ (required for log/slog structured logging) + Standard library only (no external dependencies) (003-github-issues-batch)
- N/A (infrastructure/configuration only - no Go code changes) + KIND v0.20+, kubectl v1.28+, Docker (004-kind-local-dev)
- N/A (no persistent storage required) (004-kind-local-dev)
- Go 1.21+ (required for log/slog structured logging) + Standard library only (net/http, encoding/json, os, time, context, log/slog) + Kubernetes REST API via net/http (no client-go dependency) (005-pod-restart-watchdog)
- Go 1.21+ (existing project standard) + Standard library only (no external dependencies per constitution) (006-dev-tooling-improvements)
- N/A (test infrastructure, no persistent storage) (006-dev-tooling-improvements)
- Go 1.21+ (required for log/slog structured logging) + Standard library only (no external dependencies per constitution) (007-tech-debt-cleanup)
- N/A (config file is read-only input) (007-tech-debt-cleanup)

- Go 1.21+ (required for log/slog structured logging) + Standard library only (net/http, os/signal, context, log/slog, encoding/json, time, sync) (001-mount-health-monitor)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.21+ (required for log/slog structured logging)

## Code Style

Go 1.21+ (required for log/slog structured logging): Follow standard conventions

## Recent Changes
- 007-tech-debt-cleanup: Added Go 1.21+ (required for log/slog structured logging) + Standard library only (no external dependencies per constitution)
- 006-dev-tooling-improvements: Added Go 1.21+ (existing project standard) + Standard library only (no external dependencies per constitution)
- 005-pod-restart-watchdog: Added Go 1.21+ (required for log/slog structured logging) + Standard library only (net/http, encoding/json, os, time, context, log/slog) + Kubernetes REST API via net/http (no client-go dependency)


<!-- MANUAL ADDITIONS START -->
## External Dependencies

This project uses minimal external dependencies:

- **github.com/spf13/pflag** - Drop-in replacement for Go's `flag` package that enforces POSIX/GNU flag conventions (single dash for short flags like `-c`, double dash for long flags like `--config`)
- **github.com/matryer/is** - Minimalist testing assertions library (zero transitive dependencies, test-only)
- **github.com/hashicorp/go-multierror** - Multi-error collection for config validation (preserves individual errors)
<!-- MANUAL ADDITIONS END -->
