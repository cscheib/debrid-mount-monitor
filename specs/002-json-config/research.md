# Research: JSON Configuration File

**Feature**: 002-json-config
**Date**: 2025-12-14

## Research Summary

This document captures design decisions and best practices research for implementing JSON configuration file support.

---

## 1. JSON Parsing in Go

### Decision: Use `encoding/json` from standard library

**Rationale**:
- Go's `encoding/json` is battle-tested and sufficient for configuration parsing
- Maintains zero external dependencies per constitution principle I
- Supports struct tags for field mapping and validation
- Handles duration parsing when combined with custom UnmarshalJSON

**Alternatives Considered**:
| Alternative | Rejected Because |
|-------------|------------------|
| `github.com/spf13/viper` | External dependency; overkill for simple JSON config |
| `gopkg.in/yaml.v3` | YAML not requested; adds dependency |
| `github.com/BurntSushi/toml` | TOML not requested; adds dependency |

---

## 2. Duration Parsing in JSON

### Decision: Accept string format (e.g., "30s", "5m") with custom unmarshaling

**Rationale**:
- JSON has no native duration type
- String format like "30s" is human-readable and matches existing CLI flag format
- Go's `time.ParseDuration()` handles this natively
- Consistent with existing environment variable parsing

**Implementation Pattern**:
```go
type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
    var s string
    if err := json.Unmarshal(b, &s); err != nil {
        return err
    }
    parsed, err := time.ParseDuration(s)
    if err != nil {
        return err
    }
    *d = Duration(parsed)
    return nil
}
```

**Alternatives Considered**:
| Alternative | Rejected Because |
|-------------|------------------|
| Numeric seconds (int) | Less readable; "30" vs "30s" |
| ISO 8601 duration | Overkill; Go doesn't parse natively |

---

## 3. Configuration File Discovery

### Decision: Check `--config` flag first, then `./config.json` default

**Rationale**:
- Explicit flag takes precedence (user intent is clear)
- Default location `./config.json` follows common conventions
- Silently skip if default doesn't exist (backwards compatible)
- Error only if explicitly specified path doesn't exist

**Discovery Logic**:
```
1. If --config flag provided:
   - File MUST exist → error if not found
   - Load and parse file
2. Else if ./config.json exists:
   - Load and parse file (silent fallback)
3. Else:
   - Continue without file config (backwards compatible)
```

**Alternatives Considered**:
| Alternative | Rejected Because |
|-------------|------------------|
| XDG config directories | Adds complexity; container deployments use mounted files |
| `/etc/mount-monitor/config.json` | Requires elevated permissions in some environments |
| Environment variable for path | Redundant; already have --config flag |

---

## 4. Per-Mount Configuration Inheritance

### Decision: Per-mount fields inherit from global defaults when not specified

**Rationale**:
- Reduces configuration verbosity for common cases
- Allows targeted overrides for specific mounts
- Matches existing behavior where global settings apply to all mounts

**Inheritance Chain**:
```
Per-mount value (if specified)
    ↓ (fallback)
Config file global value (if specified)
    ↓ (fallback)
Environment variable (if set)
    ↓ (fallback)
CLI flag (if specified)
    ↓ (fallback)
Hardcoded default
```

**Note**: Environment variables and CLI flags override config file entirely, including per-mount settings. This maintains the existing precedence model.

---

## 5. JSON Schema Structure

### Decision: Flat global settings + mounts array

**Rationale**:
- Simple, readable structure
- Mirrors the internal Config struct
- Easy to validate and document

**Schema**:
```json
{
  "checkInterval": "30s",
  "readTimeout": "5s",
  "shutdownTimeout": "30s",
  "debounceThreshold": 3,
  "httpPort": 8080,
  "logLevel": "info",
  "logFormat": "json",
  "canaryFile": ".health-check",
  "mounts": [
    {
      "name": "movies",
      "path": "/mnt/movies",
      "canaryFile": ".ready",
      "failureThreshold": 5
    },
    {
      "name": "tv",
      "path": "/mnt/tv"
    }
  ]
}
```

**Alternatives Considered**:
| Alternative | Rejected Because |
|-------------|------------------|
| Nested `global:` / `mounts:` sections | Adds nesting depth without benefit |
| YAML anchors/aliases | Not using YAML |

---

## 6. Error Message Design

### Decision: Include file path, line/position when possible, and specific field

**Rationale**:
- Actionable error messages reduce debugging time
- JSON parse errors from stdlib include position info
- Field-specific validation errors help locate issues

**Error Message Examples**:
```
config: file not found: /path/to/config.json
config: parse error at position 42: unexpected comma
config: mount[0]: missing required field "path"
config: mount[2] "backup": failureThreshold must be >= 1, got 0
```

---

## 7. Backwards Compatibility Strategy

### Decision: Config file is additive; existing behavior unchanged when absent

**Rationale**:
- Existing deployments must continue to work (SC-005)
- Config file layer inserts into precedence chain without disrupting it
- No behavioral changes when config file is not used

**Compatibility Guarantees**:
1. No config file + existing env vars → identical behavior to current version
2. No config file + existing CLI flags → identical behavior to current version
3. Config file + env vars → env vars win (override config file)
4. Config file + CLI flags → CLI flags win (override everything)

---

## 8. Mount Name Usage

### Decision: Name is optional; used in logs and status responses when provided

**Rationale**:
- Names improve operational visibility for multi-mount setups
- Optional to avoid breaking simpler configurations
- Appears in health status JSON and structured logs

**Name Display Locations**:
- Startup log: `"mount registered" name=movies path=/mnt/movies`
- Health status JSON: `{"name": "movies", "path": "/mnt/movies", "status": "healthy"}`
- State transition logs: `"state changed" mount=movies from=healthy to=degraded`

---

## Unresolved Items

None. All research questions have been resolved with decisions documented above.
