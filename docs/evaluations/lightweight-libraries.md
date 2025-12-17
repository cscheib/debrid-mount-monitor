# Evaluation: Lightweight Libraries for debrid-mount-monitor

> **Date**: 2025-12-16
> **Status**: Evaluated - Recommendations provided

## Executive Summary

Several lightweight libraries could improve developer experience without violating the project's minimal-dependency philosophy. The best candidates have **zero transitive dependencies** and address real pain points identified in the codebase.

**Top Recommendations:**
1. `github.com/matryer/is` - Testing assertions (0 transitive deps)
2. `github.com/hashicorp/go-multierror` - Config validation errors (1 transitive dep)

---

## Dependency Footprint Analysis

| Library | Purpose | Transitive Deps | Binary Impact | Meets Constitution? |
|---------|---------|-----------------|---------------|---------------------|
| `spf13/pflag` (current) | CLI flags | 0 | ~200KB | ✅ Yes |
| `matryer/is` | Test assertions | **0** | ~50KB | ✅ Yes |
| `hashicorp/go-multierror` | Multi-error collection | **1** (errwrap) | ~100KB | ✅ Yes |
| `cenkalti/backoff/v4` | Retry logic | **0** | ~150KB | ✅ Yes |
| `kelseyhightower/envconfig` | Env var parsing | **0** | ~100KB | ✅ Yes |
| `fsnotify/fsnotify` | File watching | **1** (x/sys) | ~200KB | ⚠️ Marginal |
| `stretchr/testify` | Test framework | **4** | ~500KB | ❌ Too heavy |

**Current binary size**: 8.8MB

---

## Detailed Analysis

### 1. Testing: `github.com/matryer/is` ⭐ RECOMMENDED

**Problem it solves:**
The test suite has 138 `t.Errorf` calls and 58 `t.Error` calls with verbose, repetitive patterns:

```go
// Current (verbose)
if cfg.Mounts[0].Name != "movies" {
    t.Errorf("expected mount[0].name 'movies', got %q", cfg.Mounts[0].Name)
}
if cfg.Mounts[0].Path != "/mnt/movies" {
    t.Errorf("expected mount[0].path '/mnt/movies', got %q", cfg.Mounts[0].Path)
}

// With matryer/is (concise)
is.Equal(cfg.Mounts[0].Name, "movies")
is.Equal(cfg.Mounts[0].Path, "/mnt/movies")
```

**Why matryer/is over testify:**
- **Zero dependencies** (testify has 4)
- Minimalist API (~10 functions vs testify's 100+)
- Produces clean, readable failure output
- Philosophy matches project's "minimal" ethos

**Dependency tree:**
```
depcheck github.com/matryer/is@v1.4.1
(no transitive dependencies)
```

**Impact:**
- Would reduce test code by ~15-20%
- Clearer failure messages
- Test-only dependency (no production impact)

---

### 2. Error Handling: `github.com/hashicorp/go-multierror` ⭐ RECOMMENDED

**Problem it solves:**
Config validation collects multiple errors via string concatenation:

```go
// Current (config.go lines 192-194)
if errMsg != "" {
    return nil, fmt.Errorf("config validation failed:\n%s", errMsg)
}

// With go-multierror
var result *multierror.Error
if cfg.HTTPPort <= 0 {
    result = multierror.Append(result, errors.New("http_port must be positive"))
}
// ... more validations
return result.ErrorOrNil()
```

**Benefits:**
- Structured error collection
- Each error preservable individually
- Standard `error` interface
- Used extensively in HashiCorp ecosystem (battle-tested)

**Dependency tree:**
```
depcheck github.com/hashicorp/go-multierror@v1.1.1
github.com/hashicorp/go-multierror@v1.1.1 github.com/hashicorp/errwrap@v1.0.0
```

**Impact:**
- Improves ~30 lines in config validation
- Better error UX for users with invalid configs
- Single transitive dep (errwrap is tiny)

---

### 3. Retry Logic: `github.com/cenkalti/backoff/v4` ⚠️ OPTIONAL

**Current implementation** (watchdog.go lines 386-436):
- Custom exponential backoff: ~50 lines
- Works well, is readable

**Would provide:**
- Jitter support (reduces thundering herd)
- Context integration
- Configurable strategies

**Verdict:** Low value. Current implementation is simple and domain-specific. The library would save ~30 lines but add a dependency for minimal benefit.

---

### 4. Environment Config: `github.com/kelseyhightower/envconfig` ⚠️ OPTIONAL

**Current usage:**
- Manual `os.Getenv()` in k8s_client.go for `KUBERNETES_SERVICE_HOST/PORT`
- Only ~10 lines of env var handling

**Would provide:**
- Struct tag-based env parsing
- Type conversion, defaults, required flags

**Verdict:** Low value. The project intentionally uses JSON config + CLI flags. Env vars are only for Kubernetes runtime detection, not user config.

---

### 5. File Watching: `github.com/fsnotify/fsnotify` ❌ NOT RECOMMENDED

**Would enable:** Config hot-reloading without restart

**Issues:**
- Adds `golang.org/x/sys` as transitive dependency
- Config reload requires state machine complexity
- Current approach (restart to reload) is simpler and acceptable

**Verdict:** Complexity outweighs benefit. Constitution favors simplicity.

---

### 6. Testing Framework: `github.com/stretchr/testify` ❌ NOT RECOMMENDED

**Dependency tree:**
```
github.com/stretchr/testify@v1.11.1 github.com/davecgh/go-spew@v1.1.1
github.com/stretchr/testify@v1.11.1 github.com/pmezard/go-difflib@v1.0.0
github.com/stretchr/testify@v1.11.1 github.com/stretchr/objx@v0.5.2
github.com/stretchr/testify@v1.11.1 gopkg.in/yaml.v3@v3.0.1
```

**Issues:**
- 4 transitive dependencies
- Much larger API surface than needed
- `matryer/is` provides equivalent value with zero deps

---

## Constitutional Compliance

For any new dependency, the constitution requires:

| Criterion | matryer/is | go-multierror | cenkalti/backoff |
|-----------|------------|---------------|------------------|
| Solves problem stdlib can't | ✅ Cleaner assertions | ✅ Multi-error pattern | ⚠️ Already solved |
| Zero/minimal transitive deps | ✅ 0 | ✅ 1 | ✅ 0 |
| Actively maintained | ✅ Yes | ✅ Yes | ✅ Yes |
| < 1MB binary impact | ✅ ~50KB | ✅ ~100KB | ✅ ~150KB |
| No CGO | ✅ Pure Go | ✅ Pure Go | ✅ Pure Go |

---

## Security Considerations

| Library | Risk Profile | Notes |
|---------|--------------|-------|
| `matryer/is` | **Minimal** | Test-only dependency; never compiled into production binary |
| `hashicorp/go-multierror` | **Low** | Widely audited in HashiCorp ecosystem (Terraform, Vault, Consul); minimal attack surface |

Both recommended libraries:
- Have no network, filesystem, or OS-level operations
- Are pure data manipulation (error aggregation, test assertions)
- Are actively maintained with security advisories tracked

---

## Recommendations Summary

### Add These (High Value, Low Risk)

| Library | Purpose | Action |
|---------|---------|--------|
| `github.com/matryer/is` | Test assertions | Add as test dependency |
| `github.com/hashicorp/go-multierror` | Config validation | Add to config package |

### Skip These (Low Value or High Cost)

| Library | Reason to Skip |
|---------|----------------|
| `stretchr/testify` | 4 transitive deps; matryer/is is lighter |
| `cenkalti/backoff` | Current impl is fine; low code savings |
| `kelseyhightower/envconfig` | Not using env vars for config |
| `fsnotify/fsnotify` | Adds complexity; restart-to-reload is acceptable |

---

## Implementation Path

If approved, the implementation order would be:

1. **matryer/is** (test-only, no production impact)
   - Update constitution's Approved Dependencies table
   - Refactor tests incrementally
   - ~2 hours of work

2. **go-multierror** (production code)
   - Update constitution's Approved Dependencies table
   - Refactor `internal/config/config.go` validation
   - ~1 hour of work

---

## Files That Would Change

**For matryer/is:**
- `go.mod` - add dependency
- `tests/unit/*.go` - refactor assertions (8 files)
- `.specify/memory/constitution.md` - add to approved deps

**For go-multierror:**
- `go.mod` - add dependency
- `internal/config/config.go` - refactor validation (~30 lines)
- `.specify/memory/constitution.md` - add to approved deps
