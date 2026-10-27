package plugin

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"strings"
)

type routeHandler struct {
	prefix    string
	directory string
	plugin    *script
	handle    Callback
}

func (r *routeHandler) Path() string                                      { return r.prefix }
func (r *routeHandler) Handler() func(http.ResponseWriter, *http.Request) { return r.serve }

type response struct {
	status    int
	headers   http.Header
	body      string
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
		// plugins map a URL to a different file beneath its configured root.
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

// Only callback execution is serialized per plugin; files stream concurrently.
func (r *routeHandler) call(req *http.Request, relative string) (response, error) {
	query := map[string]any{}
	for key, values := range req.URL.Query() {
		if len(values) > 0 {
			query[key] = values[0]
		}
	}
	headers := map[string]any{}
	for key, values := range req.Header {
		headers[strings.ToLower(key)] = strings.Join(values, ", ")
	}
	value, err := r.plugin.invoke(req.Context(), r.handle, map[string]any{"method": req.Method, "path": relative, "full_path": req.URL.Path, "query": query, "headers": headers})
	if err != nil {
		return response{}, err
	}
	return parseResponse(value)
}

func parseResponse(value any) (response, error) {
	table, ok := value.(map[string]any)
	if !ok {
		return response{}, fmt.Errorf("handle must return a response object")
	}
	result := response{status: http.StatusOK, headers: make(http.Header)}
	if value := table["status"]; value != nil {
		number, ok := value.(float64)
		if !ok || number < 200 || number > 599 || number != float64(int(number)) {
			return result, fmt.Errorf("invalid response status")
		}
		result.status = int(number)
	}
	for _, field := range []struct {
		name   string
		target *string
	}{{"body", &result.body}, {"file", &result.file}} {
		value := table[field.name]
		if value == nil {
			continue
		}
		str, ok := value.(string)
		if !ok {
			return result, fmt.Errorf("%s must be a string", field.name)
		}
		*field.target = str
	}
	if table["transform"] != nil {
		return result, fmt.Errorf("transform is no longer supported; plugin must supply process logic")
	}
	if value := table["process"]; value != nil {
		plan, err := parseProcess(value)
		if err != nil {
			return result, err
		}
		result.processes = append(result.processes, *plan)
	}
	if value := table["processes"]; value != nil {
		list, ok := value.([]any)
		if !ok || len(list) < 1 || len(list) > 8 || table["process"] != nil {
			return result, fmt.Errorf("processes requires 1..8 stages and no process")
		}
		for _, value := range list {
			plan, err := parseProcess(value)
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
	if value := table["headers"]; value != nil {
		headers, ok := value.(map[string]any)
		if !ok {
			return result, fmt.Errorf("headers must be an object")
		}
		for name, value := range headers {
			text, valueOK := value.(string)
			if !valueOK || !validHeaderName(name) || strings.ContainsAny(text, "\r\n\x00") {
				return result, fmt.Errorf("invalid response header")
			}
			canonical := textproto.CanonicalMIMEHeaderKey(name)
			switch canonical {
			case "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Upgrade":
				return result, fmt.Errorf("header %s is managed by the host", canonical)
			}
			result.headers.Set(canonical, text)
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

func parseProcess(value any) (*processPlan, error) {
	spec, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("process must be an object")
	}
	tool, tOK := spec["tool"].(string)
	extension, eOK := spec["extension"].(string)
	args, aOK := spec["args"].([]any)
	if !tOK || !eOK || !aOK {
		return nil, fmt.Errorf("process needs tool, extension and args")
	}
	plan := &processPlan{Tool: tool, Extension: extension}
	for _, value := range args {
		arg, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("process arguments must be strings")
		}
		plan.Args = append(plan.Args, arg)
	}
	return plan, plan.validate()
}
