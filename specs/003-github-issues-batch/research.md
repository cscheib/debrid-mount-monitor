# Research: GitHub Issues Batch Implementation

**Feature**: 003-github-issues-batch
**Date**: 2025-12-15

## Summary

No significant research required. This feature implements 5 incremental improvements using well-established Go standard library patterns.

## Research Items

### 1. File Size Checking in Go

**Decision**: Use `os.Stat()` to check file size before reading

**Rationale**:
- `os.Stat()` is already called to check if file exists
- Reusing the `FileInfo` result avoids additional system calls
- Standard Go idiom for pre-flight checks

**Alternatives Considered**:
- `io.LimitReader`: Would require reading the file first, defeating the purpose
- Custom streaming reader: Over-engineered for this use case

### 2. Platform-Specific Behavior Detection

**Decision**: Use `runtime.GOOS` compile-time constant

**Rationale**:
- Standard Go approach for platform detection
- Evaluated at compile time when possible
- Clean conditional without build tags

**Alternatives Considered**:
- Build tags (`// +build !windows`): More complex, requires separate files
- Environment variables: Runtime overhead, not reliable

### 3. Unix Permission Bit Checking

**Decision**: Use `info.Mode().Perm()&0002` to detect world-writable

**Rationale**:
- Direct bit manipulation is idiomatic Go
- `0002` is the standard Unix "other write" permission bit
- No external dependencies needed

**Alternatives Considered**:
- `os.FileMode` named constants: Go doesn't provide these, would need custom constants
- External permission library: Violates minimal dependency principle

### 4. Slog Handler Caching

**Decision**: Pre-create handlers in `setupLogger()`, store as struct fields

**Rationale**:
- Eliminates allocations in the hot path (`Handle()` called per log)
- Pattern is standard for custom slog handlers
- No functional change, pure optimization

**Alternatives Considered**:
- Lazy initialization with sync.Once: Adds complexity for minimal benefit
- Global handlers: Violates encapsulation, harder to test

## Conclusions

All implementation patterns are straightforward applications of Go standard library. No external research or experimentation required.
