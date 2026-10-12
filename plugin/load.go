// Package plugin loads portable Lua request plugins at server startup.
package plugin

import (
	"assetgoblin/image"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
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

// Manager owns the Lua interpreters and registered routes. Close after requests stop.
type Manager struct {
	scripts []*script
	routes  []RouteHandler
	codecs  map[string]image.Codec
}

type script struct {
	state    *lua.LState
	gate     chan struct{}
	closed   bool
	filename string
	tools    map[string]bool
	revision string
	parents  []string
	routes   []RouteHandler
	codecs   map[string]image.Codec
}

// Load discovers assetgoblin-*.lua files, in filename order. Invalid plugins fail
// startup atomically; unrelated files are ignored. An empty directory is valid.
func Load(directory string) (*Manager, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read plugin directory: %w", err)
	}
	manager := &Manager{codecs: map[string]image.Codec{}}
	catalog := map[string]string{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "assetgoblin-") && strings.HasSuffix(entry.Name(), ".lua") {
			catalog[strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "assetgoblin-"), ".lua")] = filepath.Join(directory, entry.Name())
		}
	}
	replaced := map[string]bool{}
	seen := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "assetgoblin-") || !strings.HasSuffix(name, ".lua") {
			continue
		}
		filename := filepath.Join(directory, name)
		plugin, routes, err := loadScript(filename, catalog)
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
func (m *Manager) Handlers() []RouteHandler { return append([]RouteHandler(nil), m.routes...) }

// Close releases interpreters. It is safe to call more than once.
func (m *Manager) Close() error {
	for _, plugin := range m.scripts {
		plugin.gate <- struct{}{}
		if !plugin.closed {
			plugin.closed = true
			plugin.state.Close()
		}
		<-plugin.gate
	}
	return nil
}

func loadScript(filename string, catalog map[string]string) (_ *script, _ []RouteHandler, err error) {
	state := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer func() {
		if err != nil {
			state.Close()
		}
	}()
	for _, lib := range []struct {
		name string
		open lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase}, {lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString}, {lua.MathLibName, lua.OpenMath},
	} {
		if err = state.CallByParam(lua.P{Fn: state.NewFunction(lib.open), NRet: 0, Protect: true}, lua.LString(lib.name)); err != nil {
			return nil, nil, err
		}
	}
	// No host filesystem, process, package loader, or debug access from Lua.
	for _, name := range []string{"dofile", "loadfile", "print"} {
		state.SetGlobal(name, lua.LNil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), scriptTimeout)
	defer cancel()
	state.SetContext(ctx)
	defer state.RemoveContext()
	manifest, parents, revision, err := evaluateManifest(state, filename, catalog, map[string]bool{})
	if err != nil {
		return nil, nil, err
	}
	if manifest.RawGetString("api_version") != lua.LNumber(1) {
		return nil, nil, fmt.Errorf("unsupported api_version (expected 1)")
	}
	list, ok := manifest.RawGetString("routes").(*lua.LTable)
	if !ok && manifest.RawGetString("routes") != lua.LNil {
		return nil, nil, fmt.Errorf("routes must be an array")
	}
	if !ok && manifest.RawGetString("image_codecs") == lua.LNil {
		return nil, nil, fmt.Errorf("plugin needs routes or image_codecs")
	}
	plugin := &script{state: state, gate: make(chan struct{}, 1), filename: filename, tools: map[string]bool{}, revision: revision, parents: parents, codecs: map[string]image.Codec{}}
	if value := manifest.RawGetString("tools"); value != lua.LNil {
		list, ok := value.(*lua.LTable)
		if !ok {
			return nil, nil, fmt.Errorf("tools must be an array")
		}
		for i := 1; i <= list.Len(); i++ {
			name, ok := list.RawGetInt(i).(lua.LString)
			if !ok || !validToolName(string(name)) {
				return nil, nil, fmt.Errorf("invalid tool declaration")
			}
			plugin.tools[string(name)] = true
		}
	}
	count := 0
	if list != nil {
		count = list.Len()
	}
	routes := make([]RouteHandler, 0, count)
	for i := 1; i <= count; i++ {
		spec, ok := list.RawGetInt(i).(*lua.LTable)
		if !ok {
			return nil, nil, fmt.Errorf("route %d must be a table", i)
		}
		prefix, ok := spec.RawGetString("path").(lua.LString)
		if !ok || !validPrefix(string(prefix)) {
			return nil, nil, fmt.Errorf("route %d has invalid path", i)
		}
		route := &routeHandler{prefix: string(prefix), plugin: plugin}
		if value := spec.RawGetString("directory"); value != lua.LNil {
			directory, ok := value.(lua.LString)
			if !ok || directory == "" {
				return nil, nil, fmt.Errorf("route %d has invalid directory", i)
			}
			route.directory, err = filepath.Abs(string(directory))
			if err != nil {
				return nil, nil, err
			}
		}
		if spec.RawGetString("media") != lua.LNil {
			return nil, nil, fmt.Errorf("media route types are no longer supported; declare plugin tools and custom logic")
		}
		if value := spec.RawGetString("handle"); value != lua.LNil {
			route.handle, ok = value.(*lua.LFunction)
			if !ok {
				return nil, nil, fmt.Errorf("route %d handle must be a function", i)
			}
		}
		if route.directory == "" && route.handle == nil {
			return nil, nil, fmt.Errorf("route %d needs directory or handle", i)
		}
		routes = append(routes, route)
	}
	if err := plugin.loadCodecs(manifest.RawGetString("image_codecs")); err != nil {
		return nil, nil, err
	}
	if len(routes) == 0 && len(plugin.codecs) == 0 {
		return nil, nil, fmt.Errorf("plugin needs nonempty routes or image_codecs")
	}
	return plugin, routes, nil
}

func validPrefix(prefix string) bool {
	decoded, err := url.PathUnescape(prefix)
	return err == nil && decoded == prefix && prefix != "/" && strings.HasPrefix(prefix, "/") &&
		strings.HasSuffix(prefix, "/") && path.Clean(prefix)+"/" == prefix && !strings.ContainsAny(prefix, "{} \t\r\n")
}
