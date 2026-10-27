package plugin

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"sync"
)

// Backend evaluates a plugin in an optional runtime. Manifests contain string-keyed
// maps, []any arrays, float64 numbers, strings, booleans, nil and Callback functions.
// Load returns the manifest, ancestor filenames, cache revision and a non-nil closer.
// On error the backend must release its resources. The host serializes callbacks
// and closing, supplies deadlines, and validates every manifest and response.
// A backend may not mutate the discovery catalog.
type Backend interface {
	Extension() string
	Load(context.Context, string, map[string]string) (any, []string, string, func(), error)
}

// Callback is a runtime function invoked with normalized request metadata.
type Callback = func(context.Context, map[string]any) (any, error)

var backendRegistry = struct {
	sync.RWMutex
	items map[string]Backend
}{items: make(map[string]Backend)}

// RegisterBackend enables an optional runtime for subsequent Load calls. Call it
// during application initialization. Python is built in and always takes priority
// over optional runtimes with the same plugin name.
func RegisterBackend(backend Backend) error {
	if backend == nil {
		return fmt.Errorf("nil plugin backend")
	}
	ext := backend.Extension()
	if len(ext) < 2 || ext[0] != '.' || strings.Trim(ext[1:], "abcdefghijklmnopqrstuvwxyz0123456789") != "" || ext == ".py" {
		return fmt.Errorf("invalid or reserved backend extension %q", ext)
	}
	backendRegistry.Lock()
	defer backendRegistry.Unlock()
	if backendRegistry.items[ext] != nil {
		return fmt.Errorf("backend already registered for %q", ext)
	}
	backendRegistry.items[ext] = backend
	return nil
}

func registeredBackends() map[string]Backend {
	backendRegistry.RLock()
	defer backendRegistry.RUnlock()
	return maps.Clone(backendRegistry.items)
}

func loadBackend(backend Backend, filename string, catalog map[string]string) (*script, []RouteHandler, error) {
	ctx, cancel := context.WithTimeout(context.Background(), scriptTimeout)
	defer cancel()
	value, parents, revision, closeRuntime, err := backend.Load(ctx, filename, catalog)
	if err != nil {
		return nil, nil, err
	}
	if closeRuntime == nil {
		return nil, nil, fmt.Errorf("backend returned no closer")
	}
	plugin := newScript(filename)
	plugin.close = closeRuntime
	plugin.parents, plugin.revision = parents, revision
	routes, err := plugin.loadManifest(value)
	if err != nil {
		closeRuntime()
		return nil, nil, err
	}
	return plugin, routes, nil
}
