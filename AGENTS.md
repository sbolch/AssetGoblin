# AGENTS.md - AssetGoblin Development Guide

## Project Overview

AssetGoblin is a Go web server for serving and transforming static assets, particularly images with support for different formats and presets.

## Build & Test Commands

```bash
# Build the project
go build ./...

# Run tests
go test ./...

# Run tests with race detector
go test ./... -race

# Run vet for static analysis
go vet ./...

# Run with coverage
go test -cover ./...

# Generate documentation
go doc -all ./...

# Run the application
go run . --serve

# Build for release
GOOS=linux GOARCH=amd64 go build -o assetgoblin-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o assetgoblin-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o assetgoblin-windows-amd64.exe
```

## Key Dependencies

- `github.com/spf13/viper` - Configuration management

## Code Architecture

### Packages

- `config` - Configuration loading and management
- `image` - Image processing and serving
- `middleware` - HTTP middleware (rate limiting, authentication)
- `utils` - Common utilities (file/reader closing) and ImagePreset struct
- `main` - Entry point and CLI handling

### Logging

Uses `log/slog` for structured logging. Always use key-value pairs:

```go
slog.Error("Failed to load config", "error", err)
slog.Info("Starting server", "port", conf.Port)
slog.Warn("Failed to save gob file", "error", err)
```

### HTTP Server

Uses `http.Server` with timeouts configured:

```go
srv := &http.Server{
    Addr:         ":" + conf.Port,
    Handler:      handler,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
}
```

### Testing

- Tests use standard Go testing package
- Use `t.Helper()` for helper functions
- Use table-driven tests where appropriate
- Isolate tests that require environment variables using `t.Setenv()`

### Best Practices

1. **Error handling**: Use `os.Exit(1)` for fatal errors, `slog.Error` for errors that allow continuation
2. **Permissions**: Use explicit permissions (e.g., `0755`) instead of `os.ModePerm`
3. **HTTP clients**: Always set timeouts on `http.Client`
4. **Slices**: Use `slices.Contains` for format checking, `slices.IndexFunc` for finding elements
5. **Strings**: Use `strings.TrimPrefix` instead of manual string manipulation
6. **IP extraction**: Always extract IP from `req.RemoteAddr` before using in rate limiter
7. **File closing**: Use `utils.CloseFile` and `utils.CloseReader` for safe file/reader closing with error logging

### Configuration

- Config is loaded from `config.*` files (JSON, TOML, YAML, HCL, envfile)
- Config is cached in gob format for faster loading
- Use `-clear-gob` flag to clear the cached config

### Key Files

- `main.go` - Entry point with CLI flag parsing
- `serve.go` - HTTP server setup
- `version.go` - Version management and auto-update
- `config/load.go` - Configuration loading
- `config/utils.go` - Configuration persistence utilities
- `image/serve.go` - Image processing and serving
- `image/utils.go` - Image utility functions
- `middleware/ratelimit.go` - Rate limiting middleware
- `middleware/signkey.go` - HMAC-based request verification
- `utils/utils.go` - Common utilities (file/reader closing)
- `utils/image_preset.go` - ImagePreset struct definition
