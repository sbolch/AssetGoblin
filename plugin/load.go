// Package plugin loads Python plugins and explicitly registered optional runtimes at server startup.
package plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"assetgoblin/image"
)

const (
	scriptTimeout = time.Second
	maxScriptSize = 1 << 20
)

// RouteHandler is a route mounted beneath the server's global middleware.
type RouteHandler interface {
	Path() string
	Handler() func(http.ResponseWriter, *http.Request)
}

// Manager owns the plugin runtimes and registered routes. Close after requests stop.
type Manager struct {
	scripts []*script
	routes  []RouteHandler
	codecs  map[string]image.Codec
}

type script struct {
	close    func()
	gate     chan struct{}
	closed   bool
	filename string
	tools    map[string]bool
	revision string
	parents  []string
	routes   []RouteHandler
	codecs   map[string]image.Codec
}

// Load discovers assetgoblin-*.py and registered backend files, in filename order.
// Python takes precedence when both files have the same plugin name. Invalid plugins fail
// startup atomically; unrelated files are ignored. An empty directory is valid.
func Load(directory string) (*Manager, error) { return load(directory, "") }

func load(directory, pythonExecutable string) (*Manager, error) {
	backends := registeredBackends()
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read plugin directory: %w", err)
	}
	manager := &Manager{codecs: map[string]image.Codec{}}
	catalog := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "assetgoblin-") {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".py" && backends[ext] == nil {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "assetgoblin-"), ext)
		if catalog[name] == "" || ext == ".py" {
			catalog[name] = filepath.Join(directory, entry.Name())
		}
	}

	replaced := map[string]bool{}
	seen := map[string]bool{}
	names := make([]string, 0, len(catalog))
	for _, filename := range catalog {
		names = append(names, filename)
	}
	slices.Sort(names)
	for _, filename := range names {
		name := filepath.Base(filename)
		var plugin *script
		var routes []RouteHandler
		var err error
		if filepath.Ext(filename) == ".py" {
			plugin, routes, err = loadPython(filename, catalog, pythonExecutable)
		} else {
			plugin, routes, err = loadBackend(backends[filepath.Ext(filename)], filename, catalog)
		}
		if err != nil {
			manager.Close()
			return nil, fmt.Errorf("plugin %s: %w", name, err)
		}
		manager.scripts = append(manager.scripts, plugin)
		plugin.routes = routes
		for _, parent := range plugin.parents {
			replaced[parent] = true
		}
	}
	for _, plugin := range manager.scripts {
		if replaced[plugin.filename] {
			continue
		}
		for _, route := range plugin.routes {
			if seen[route.Path()] {
				manager.Close()
				return nil, fmt.Errorf("duplicate plugin route %q", route.Path())
			}
			seen[route.Path()] = true
			manager.routes = append(manager.routes, route)
		}
		for name, codec := range plugin.codecs {
			if manager.codecs[name] != nil {
				manager.Close()
				return nil, fmt.Errorf("duplicate image codec %q", name)
			}
			manager.codecs[name] = codec
		}
	}

	return manager, nil
}

// Handlers returns an independent route list. Plugins cannot modify the server registry.
func (m *Manager) Handlers() []RouteHandler { return slices.Clone(m.routes) }

// Close releases interpreters. It is safe to call more than once.
func (m *Manager) Close() error {
	for _, plugin := range m.scripts {
		plugin.gate <- struct{}{}
		if !plugin.closed {
			plugin.closed = true
			plugin.close()
		}
		<-plugin.gate
	}
	return nil
}

func newScript(filename string) *script {
	return &script{filename: filename, gate: make(chan struct{}, 1), tools: map[string]bool{}, codecs: map[string]image.Codec{}}
}

func (s *script) invoke(ctx context.Context, fn Callback, request map[string]any) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, scriptTimeout)
	defer cancel()
	select {
	case s.gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-s.gate }()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.closed {
		return nil, fmt.Errorf("plugin is closed")
	}
	return fn(ctx, request)
}

func (plugin *script) loadManifest(value any) ([]RouteHandler, error) {
	manifest, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("plugin must return a manifest object")
	}
	if manifest["api_version"] != float64(1) {
		return nil, fmt.Errorf("unsupported api_version (expected 1)")
	}
	list, ok := manifest["routes"].([]any)
	if !ok && manifest["routes"] != nil {
		return nil, fmt.Errorf("routes must be an array")
	}
	if !ok && manifest["image_codecs"] == nil {
		return nil, fmt.Errorf("plugin needs routes or image_codecs")
	}
	if value := manifest["tools"]; value != nil {
		tools, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("tools must be an array")
		}
		for _, v := range tools {
			name, ok := v.(string)
			if !ok || !validToolName(name) {
				return nil, fmt.Errorf("invalid tool declaration")
			}
			plugin.tools[name] = true
		}
	}
	routes := make([]RouteHandler, 0, len(list))
	for i, v := range list {
		spec, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("route %d must be an object", i+1)
		}
		prefix, ok := spec["path"].(string)
		if !ok || !validPrefix(prefix) {
			return nil, fmt.Errorf("route %d has invalid path", i+1)
		}
		route := &routeHandler{prefix: prefix, plugin: plugin}
		if value := spec["directory"]; value != nil {
			directory, ok := value.(string)
			if !ok || directory == "" {
				return nil, fmt.Errorf("route %d has invalid directory", i+1)
			}
			var err error
			route.directory, err = filepath.Abs(directory)
			if err != nil {
				return nil, err
			}
		}
		if spec["media"] != nil {
			return nil, fmt.Errorf("media route types are no longer supported; declare plugin tools and custom logic")
		}
		if value := spec["handle"]; value != nil {
			route.handle, ok = value.(Callback)
			if !ok {
				return nil, fmt.Errorf("route %d handle must be a function", i+1)
			}
		}
		if route.directory == "" && route.handle == nil {
			return nil, fmt.Errorf("route %d needs directory or handle", i+1)
		}
		routes = append(routes, route)
	}
	if err := plugin.loadCodecs(manifest["image_codecs"]); err != nil {
		return nil, err
	}
	if len(routes) == 0 && len(plugin.codecs) == 0 {
		return nil, fmt.Errorf("plugin needs nonempty routes or image_codecs")
	}
	return routes, nil
}
func validPrefix(prefix string) bool {
	decoded, err := url.PathUnescape(prefix)
	return err == nil && decoded == prefix && prefix != "/" && strings.HasPrefix(prefix, "/") &&
		strings.HasSuffix(prefix, "/") && path.Clean(prefix)+"/" == prefix && !strings.ContainsAny(prefix, "{} \t\r\n")
}
