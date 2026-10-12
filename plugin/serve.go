package plugin

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type routeHandler struct {
	prefix    string
	directory string
	plugin    *script
	handle    *lua.LFunction
}

func (r *routeHandler) Path() string                                      { return r.prefix }
func (r *routeHandler) Handler() func(http.ResponseWriter, *http.Request) { return r.serve }

type response struct {
	status    int
	headers   http.Header
	body      string
	process   *processPlan
	processes []processPlan
	file      string
}

func (r *routeHandler) serve(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	relative, ok := strings.CutPrefix(req.URL.Path, r.prefix)
	if !ok || (relative != "" && !fs.ValidPath(relative)) {
		http.NotFound(w, req)
		return
	}
	result := response{status: http.StatusOK, file: relative}
	if r.handle != nil {
		var err error
		result, err = r.call(req, relative)
		if err != nil {
			slog.Error("Plugin request failed", "plugin", r.plugin.filename, "path", req.URL.Path, "error", err)
			http.Error(w, "Plugin request failed", http.StatusInternalServerError)
			return
		}
	}
	if result.file != "" {
		if r.directory == "" || !fs.ValidPath(result.file) {
			http.NotFound(w, req)
			return
		}
		root, err := os.OpenRoot(r.directory)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		defer root.Close()
		info, err := fs.Stat(root.FS(), result.file)
		if err != nil || info.IsDir() {
			http.NotFound(w, req)
			return
		}
		if len(result.processes) > 0 {
			source, err := root.Open(result.file)
			if err != nil {
				http.NotFound(w, req)
				return
			}
			defer source.Close()
			output, err := runPipeline(req.Context(), source, r.directory+"/"+result.file, r.plugin.revision, result.processes, r.plugin.tools)
			if err != nil {
				slog.Error("Plugin process failed", "error", err, "path", req.URL.Path)
				http.Error(w, "Plugin process failed", http.StatusInternalServerError)
				return
			}
			converted, err := os.Open(output)
			if err != nil {
				http.Error(w, "Plugin cache unavailable", 500)
				return
			}
			defer converted.Close()
			stat, err := converted.Stat()
			if err != nil {
				http.Error(w, "Plugin cache unavailable", 500)
				return
			}
			for key, values := range result.headers {
				w.Header()[key] = values
			}
			http.ServeContent(w, req, stat.Name(), stat.ModTime(), converted)
			return
		}
		for key, values := range result.headers {
			w.Header()[key] = values
		}
		// Keep the original request's Range, If-Modified-Since, etc., while letting
		// Lua map a URL to a different file beneath its configured root.
		clone := req.Clone(req.Context())
		clone.URL.Path = "/" + result.file
		clone.URL.RawPath = ""
		http.FileServerFS(root.FS()).ServeHTTP(w, clone)
		return
	}
	if r.handle == nil {
		http.NotFound(w, req)
		return
	}
	for key, values := range result.headers {
		w.Header()[key] = values
	}
	if result.status != http.StatusNoContent && result.status != http.StatusNotModified {
		w.Header().Set("Content-Length", strconv.Itoa(len(result.body)))
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.WriteHeader(result.status)
	if req.Method != http.MethodHead && result.status != http.StatusNoContent && result.status != http.StatusNotModified {
		_, _ = w.Write([]byte(result.body))
	}
}

// Only Lua execution is serialized per plugin. Files stream outside the interpreter.
func (r *routeHandler) call(req *http.Request, relative string) (response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), scriptTimeout)
	defer cancel()
	plugin := r.plugin
	select {
	case plugin.gate <- struct{}{}:
	case <-ctx.Done():
		return response{}, ctx.Err()
	}
	defer func() { <-plugin.gate }()
	if plugin.closed {
		return response{}, fmt.Errorf("plugin is closed")
	}
	state := plugin.state
	state.SetContext(ctx)
	defer state.RemoveContext()
	stackTop := state.GetTop()
	defer state.SetTop(stackTop)
	request := state.NewTable()
	request.RawSetString("method", lua.LString(req.Method))
	request.RawSetString("path", lua.LString(relative))
	request.RawSetString("full_path", lua.LString(req.URL.Path))
	query := state.NewTable()
	for key, values := range req.URL.Query() {
		if len(values) > 0 {
			query.RawSetString(key, lua.LString(values[0]))
		}
	}
	request.RawSetString("query", query)
	headers := state.NewTable()
	for key, values := range req.Header {
		headers.RawSetString(strings.ToLower(key), lua.LString(strings.Join(values, ", ")))
	}
	request.RawSetString("headers", headers)
	if err := state.CallByParam(lua.P{Fn: r.handle, NRet: 1, Protect: true}, request); err != nil {
		return response{}, err
	}
	table, ok := state.Get(-1).(*lua.LTable)
	if !ok {
		return response{}, fmt.Errorf("handle must return a response table")
	}
	return parseResponse(table)
}

func parseResponse(table *lua.LTable) (response, error) {
	result := response{status: http.StatusOK, headers: make(http.Header)}
	if value := table.RawGetString("status"); value != lua.LNil {
		number, ok := value.(lua.LNumber)
		if !ok || number < 200 || number > 599 || number != lua.LNumber(int(number)) {
			return result, fmt.Errorf("invalid response status")
		}
		result.status = int(number)
	}
	for _, field := range []struct {
		name   string
		target *string
	}{{"body", &result.body}, {"file", &result.file}} {
		value := table.RawGetString(field.name)
		if value == lua.LNil {
			continue
		}
		str, ok := value.(lua.LString)
		if !ok {
			return result, fmt.Errorf("%s must be a string", field.name)
		}
		*field.target = string(str)
	}
	if table.RawGetString("transform") != lua.LNil {
		return result, fmt.Errorf("transform is no longer supported; plugin must supply process logic")
	}
	if value := table.RawGetString("process"); value != lua.LNil {
		plan, err := parseProcess(value)
		if err != nil {
			return result, err
		}
		result.process = plan
		result.processes = append(result.processes, *plan)
	}
	if value := table.RawGetString("processes"); value != lua.LNil {
		list, ok := value.(*lua.LTable)
		if !ok || list.Len() < 1 || list.Len() > 8 || result.process != nil {
			return result, fmt.Errorf("processes requires 1..8 stages and no process")
		}
		for i := 1; i <= list.Len(); i++ {
			plan, err := parseProcess(list.RawGetInt(i))
			if err != nil {
				return result, err
			}
			result.processes = append(result.processes, *plan)
		}
	}
	if len(result.processes) > 0 && result.file == "" {
		return result, fmt.Errorf("process needs file")
	}
	if len(result.body) > maxScriptSize {
		return result, fmt.Errorf("body exceeds %d bytes; use file for streaming", maxScriptSize)
	}
	if result.file != "" && (result.body != "" || result.status != http.StatusOK) {
		return result, fmt.Errorf("file responses require status 200 and no body")
	}
	if (result.status == http.StatusNoContent || result.status == http.StatusNotModified) && result.body != "" {
		return result, fmt.Errorf("response status does not allow a body")
	}
	if value := table.RawGetString("headers"); value != lua.LNil {
		headers, ok := value.(*lua.LTable)
		if !ok {
			return result, fmt.Errorf("headers must be a table")
		}
		var headerErr error
		headers.ForEach(func(key, value lua.LValue) {
			name, nameOK := key.(lua.LString)
			text, valueOK := value.(lua.LString)
			if !nameOK || !valueOK || !validHeaderName(string(name)) || strings.ContainsAny(string(text), "\r\n\x00") {
				headerErr = fmt.Errorf("invalid response header")
				return
			}
			canonical := textproto.CanonicalMIMEHeaderKey(string(name))
			switch canonical {
			case "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Upgrade":
				headerErr = fmt.Errorf("header %s is managed by the host", canonical)
				return
			}
			result.headers.Set(canonical, string(text))
		})
		if headerErr != nil {
			return result, headerErr
		}
	}
	return result, nil
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, ch := range name {
		if ch > 127 || !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", ch)) {
			return false
		}
	}
	return true
}

func parseProcess(value lua.LValue) (*processPlan, error) {
	spec, ok := value.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("process must be table")
	}
	tool, tOK := spec.RawGetString("tool").(lua.LString)
	extension, eOK := spec.RawGetString("extension").(lua.LString)
	args, aOK := spec.RawGetString("args").(*lua.LTable)
	if !tOK || !eOK || !aOK {
		return nil, fmt.Errorf("process needs tool, extension and args")
	}
	plan := &processPlan{Tool: string(tool), Extension: string(extension)}
	for i := 1; i <= args.Len(); i++ {
		arg, ok := args.RawGetInt(i).(lua.LString)
		if !ok {
			return nil, fmt.Errorf("process arguments must be strings")
		}
		plan.Args = append(plan.Args, string(arg))
	}
	return plan, plan.validate()
}
