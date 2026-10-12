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

- Go 1.27.2 or newer
- FFmpeg on PATH for optional audio/video transformations
- `github.com/go-fuego/fuego` - HTTP server and routing
- `github.com/yuin/gopher-lua` - Portable startup-loaded Lua plugins
- `github.com/spf13/viper` - Configuration management

## Code Architecture

### Packages

- `config` - Configuration loading and management
- `image` - Image processing and serving
- `middleware` - HTTP middleware (rate limiting, authentication)
- `utils` - Common utilities (file/reader closing) and ImagePreset struct
- `plugin` - Lua plugin discovery, request callbacks, generic process execution and file serving
- `main` - Entry point and CLI handling

### Logging

Uses `log/slog` for structured logging. Always use key-value pairs:

```go
slog.Error("Failed to load config", "error", err)
slog.Info("Starting server", "port", conf.Port)
slog.Warn("Failed to save gob file", "error", err)
```

### HTTP Server

Uses Fuego with standard HTTP handlers and explicit timeouts. `newServer` is shared
by production and server tests. OpenAPI output is disabled. Runtime plugins are
`assetgoblin-*.lua` files beside the executable, loaded only when serving starts.
`-plugins-dir` overrides discovery (needed for `go run`). Examples and their
documentation live in `plugins/{audio,video,qoi,qoa}` and distribute
Lua files independently; there are no build tags or local module
replacements. Run `go test ./... -race` when changing plugin integration.

The embedded `http.Server` has timeouts configured:

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

Core process tests use an arbitrary text-processing tool. Example Lua scripts are
responsible for audio/video formats, validation, codecs, FFmpeg arguments and filters.
Keep the host application-neutral: it provides file staging, declared-tool execution,
size/deadline limits, caching and HTTP responses. Tools run with the server user's
permissions; deploy trusted plugins. FFmpeg enables example integration tests;
QOA additionally needs `qoaconv`.
Image codecs bridge through PNG to reuse core presets and transforms. Plugin
`extends` dependencies compose parent manifests in the leaf interpreter; only
leaves register routes/codecs. Process pipelines share one deadline.

- Use `features/**/*.feature` for checks wherever feasible, including parsing,
  validation, caching, concurrency and lifecycle behavior. Scenario outlines hold
  inputs and expected outcomes; Go functions implement setup, actions and assertions.
  Ordinary Go test functions should be suite runners or subprocess fixtures.
- Tests run through the standard Go testing package
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
- `plugin.go` - Executable-relative plugin discovery
- `plugin/load.go` - Lua interpreter setup and plugin API validation
- `plugin/serve.go` - Lua callbacks and root-scoped file serving
- `plugin/process.go` - Generic plugin-defined tool execution and caching
- `version.go` - Version management and auto-update
- `config/load.go` - Configuration loading
- `config/utils.go` - Configuration persistence utilities
- `image/serve.go` - Image processing and serving
- `image/utils.go` - Image utility functions
- `middleware/ratelimit.go` - Rate limiting middleware
- `middleware/signkey.go` - HMAC-based request verification
- `utils/utils.go` - Common utilities (file/reader closing)
- `utils/image.go` - ImagePreset struct definition

### Documentation

`README.md` is the overview; `docs/` holds the canonical configuration, image,
security, CLI, plugin API and development references. Example plugin READMEs contain
plugin-specific behavior and link to shared deployment/media parameters in their
catalog. Avoid copying host limits or API details into examples.
GitHub Pages renders these same Markdown sources with Hugo. `hugo.toml` mounts the
existing documentation and assets; `layouts/` contains the site templates and
Markdown link hook. The documentation workflow uses pinned Hugo 0.167.0, validates
PR builds and deploys `main`. Keep relative Markdown links valid on GitHub; the
render hook rewrites Markdown destinations to the stable HTML page URLs.
