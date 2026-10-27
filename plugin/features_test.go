package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/regen-network/gocuke"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPluginFeatures(t *testing.T) {
	processHelper(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	gocuke.NewRunner(t, &pluginSuite{}).Path("../features/plugins/*.feature").
		Step(`^an unrelated Python file$`, (*pluginSuite).Unrelated).
		Step(`^plugin source (.*)$`, (*pluginSuite).Source).
		Step(`^plugin discovery runs$`, (*pluginSuite).Load).
		Step(`^discovery runs in a missing directory$`, (*pluginSuite).MissingDirectory).
		Step(`^(\d+) routes should be exposed$`, (*pluginSuite).RouteCount).
		Step(`^discovery should fail$`, (*pluginSuite).LoadFails).
		Step(`^plugins named "([^"]*)" with matching routes$`, (*pluginSuite).NamedPlugins).
		Step(`^routes should be ordered "([^"]*)"$`, (*pluginSuite).Ordered).
		Step(`^a returned route list is changed$`, (*pluginSuite).MutateRoutes).
		Step(`^plugin "([^"]*)" declares route "([^"]*)"$`, (*pluginSuite).Route).
		Step(`^an initialization script exceeding "([^"]*)"$`, (*pluginSuite).Unbounded).
		Step(`^image codec definition (.*)$`, (*pluginSuite).Codec).
		Step(`^a rooted file plugin with sample content "([^"]*)"$`, (*pluginSuite).FilePlugin).
		Step(`^a plugin "([^"]*)" request is sent to "([^"]*)" with range "([^"]*)"$`, (*pluginSuite).Request).
		Step(`^the plugin response status should be (\d+)$`, (*pluginSuite).Status).
		Step(`^the plugin response body should be "([^"]*)"$`, (*pluginSuite).Body).
		Step(`^a source symlink pointing outside its root$`, (*pluginSuite).Symlink).
		Step(`^the same request is made with its last-modified condition$`, (*pluginSuite).Conditional).
		Step(`^a custom callback with a persistent counter$`, (*pluginSuite).CustomCallback).
		Step(`^request header "([^"]*)" is "([^"]*)"$`, (*pluginSuite).RequestHeader).
		Step(`^response header "([^"]*)" should be "([^"]*)"$`, (*pluginSuite).Header).
		Step(`^the callback response should have an empty body and positive content length$`, (*pluginSuite).HEAD).
		Step(`^a looping callback request is cancelled$`, (*pluginSuite).CancelCallback).
		Step(`^the manager is closed twice$`, (*pluginSuite).CloseTwice).
		Step(`^a counting callback$`, (*pluginSuite).Counting).
		Step(`^(\d+) plugin requests arrive concurrently$`, (*pluginSuite).Concurrent).
		Step(`^every response should contain a different counter from 1 to (\d+)$`, (*pluginSuite).Counters).
		Step(`^a three-plugin inheritance chain$`, (*pluginSuite).Chain).
		Step(`^its leaf revision is remembered$`, (*pluginSuite).RememberRevision).
		Step(`^the parent script is changed$`, (*pluginSuite).ChangeParent).
		Step(`^its leaf revision should differ$`, (*pluginSuite).RevisionChanged).
		Step(`^another extension of the middle plugin$`, (*pluginSuite).Fork).
		Step(`^a base plugin$`, (*pluginSuite).Base).
		Step(`^extension source (.*)$`, (*pluginSuite).Extension).
		Step(`^an uppercase tool plugin$`, (*pluginSuite).ToolPlugin).
		Step(`^an uppercase tool plugin without a tool declaration$`, (*pluginSuite).UndeclaredToolPlugin).
		Step(`^process input "([^"]*)"$`, (*pluginSuite).ProcessInput).
		Step(`^the "([^"]*)" tool runs with revision "([^"]*)"$`, (*pluginSuite).Process).
		Step(`^the process output should be "([^"]*)"$`, (*pluginSuite).Output).
		Step(`^the output path is remembered$`, (*pluginSuite).RememberOutput).
		Step(`^the output path should be unchanged$`, (*pluginSuite).SameOutput).
		Step(`^the output path should differ$`, (*pluginSuite).DifferentOutput).
		Step(`^(\d+) completed outputs and no temporary files should remain$`, (*pluginSuite).CacheFiles).
		Step(`^processing should fail$`, (*pluginSuite).ProcessFails).
		Step(`^process plan (.*)$`, (*pluginSuite).Plan).
		Step(`^the process plan is validated$`, (*pluginSuite).ValidatePlan).
		Step(`^two uppercase stages run as a declared pipeline$`, (*pluginSuite).Pipeline).
		Step(`^two uppercase stages run without a tool declaration$`, (*pluginSuite).UndeclaredPipeline).
		Run()
}

type pluginSuite struct {
	gocuke.TestingT
	tool, root, directory, media, cache string
	manager                             *Manager
	handler                             http.HandlerFunc
	loadErr, processErr                 error
	response                            *httptest.ResponseRecorder
	headers                             http.Header
	source                              *os.File
	output, remembered, revision        string
	plan                                processPlan
	results                             []int
}

func (s *pluginSuite) Before() {
	s.Helper()
	executable, err := os.Executable()
	if err != nil {
		s.Fatal(err)
	}
	s.tool = filepath.Base(executable)
	s.root, err = os.MkdirTemp("", "assetgoblin-plugin-features-")
	if err != nil {
		s.Fatal(err)
	}
	s.directory = filepath.Join(s.root, "plugins")
	s.media = filepath.Join(s.root, "files")
	s.cache = filepath.Join(s.root, "cache")
	for _, dir := range []string{s.directory, s.media, s.cache} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			s.Fatal(err)
		}
	}
	s.headers = make(http.Header)
	s.loadErr = nil
	s.processErr = nil
	s.response = nil
	s.results = nil
	s.output = ""
	s.remembered = ""
	s.revision = ""
	s.handler = nil
}
func (s *pluginSuite) After() {
	if s.manager != nil {
		s.manager.Close()
		s.manager = nil
	}
	if s.source != nil {
		s.source.Close()
		s.source = nil
	}
	os.RemoveAll(s.root)
}
func (s *pluginSuite) write(name, source string) {
	s.Helper()
	if err := os.WriteFile(filepath.Join(s.directory, name), []byte(source), 0600); err != nil {
		s.Fatal(err)
	}
}
func (s *pluginSuite) Unrelated()           { s.write("unrelated.py", "not valid Python") }
func (s *pluginSuite) Source(source string) { s.write("assetgoblin-custom.py", source) }
func (s *pluginSuite) Load() {
	if s.manager != nil {
		s.manager.Close()
	}
	s.manager, s.loadErr = Load(s.directory)
	if s.manager != nil && len(s.manager.Handlers()) > 0 {
		s.handler = s.manager.Handlers()[0].Handler()
	}
}
func (s *pluginSuite) MissingDirectory() {
	s.manager, s.loadErr = Load(filepath.Join(s.root, "missing"))
}
func (s *pluginSuite) RouteCount(count int64) {
	s.Helper()
	if s.loadErr != nil {
		s.Fatal(s.loadErr)
	}
	if s.manager == nil || len(s.manager.Handlers()) != int(count) {
		s.Fatalf("expected %d routes", count)
	}
}
func (s *pluginSuite) LoadFails() {
	s.Helper()
	if s.loadErr == nil {
		s.Fatal("invalid plugin accepted")
	}
}
func (s *pluginSuite) Route(name, prefix string) {
	s.write("assetgoblin-"+name+".py", fmt.Sprintf("plugin = {'api_version': 1, 'routes': [{'path': %q, 'directory': '.'}]}", prefix))
}
func (s *pluginSuite) NamedPlugins(names string) {
	for _, name := range strings.Split(names, ",") {
		s.Route(name, "/"+name+"/")
	}
}
func (s *pluginSuite) Ordered(prefixes string) {
	s.Helper()
	expected := strings.Split(prefixes, ",")
	if s.manager == nil || len(s.manager.Handlers()) != len(expected) {
		s.Fatal("wrong route count")
	}
	for i, route := range s.manager.Handlers() {
		if route.Path() != expected[i] {
			s.Fatalf("route %d is %q, want %q", i, route.Path(), expected[i])
		}
	}
}
func (s *pluginSuite) MutateRoutes() { s.manager.Handlers()[0] = nil }
func (s *pluginSuite) Unbounded(limit string) {
	source := "while True: pass"
	if limit == "size" {
		source = strings.Repeat(" ", maxScriptSize+1)
	}
	s.Source(source)
}
func (s *pluginSuite) Codec(definition string) {
	s.Source("plugin = {'api_version': 1, 'image_codecs': {" + definition + "}}")
}
func (s *pluginSuite) FilePlugin(content string) {
	s.Helper()
	if err := os.WriteFile(filepath.Join(s.media, "sample.bin"), []byte(content), 0600); err != nil {
		s.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(s.media, "nested"), 0700); err != nil {
		s.Fatal(err)
	}
	s.Source(fmt.Sprintf("plugin = {'api_version': 1, 'routes': [{'path': '/a/', 'directory': %q}]}", filepath.ToSlash(s.media)))
	s.Load()
	if s.loadErr != nil {
		s.Fatal(s.loadErr)
	}
}
func (s *pluginSuite) Request(method, path, byteRange string) {
	s.Helper()
	if s.handler == nil {
		s.Fatal("plugin not loaded")
	}
	req := httptest.NewRequest(method, path, nil)
	req.Header = s.headers.Clone()
	if byteRange != "" {
		req.Header.Set("Range", byteRange)
	}
	s.response = httptest.NewRecorder()
	s.handler(s.response, req)
}
func (s *pluginSuite) Status(status int64) {
	s.Helper()
	if s.response == nil || s.response.Code != int(status) {
		s.Fatalf("expected status %d, response %+v", status, s.response)
	}
}
func (s *pluginSuite) Body(body string) {
	s.Helper()
	if s.response == nil || s.response.Body.String() != body {
		s.Fatalf("expected body %q, got %+v", body, s.response)
	}
}
func (s *pluginSuite) Symlink() {
	s.Helper()
	outside := filepath.Join(s.root, "secret.bin")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		s.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(s.media, "escape.bin")); err != nil {
		s.Skip("symlinks unavailable: ", err)
	}
}
func (s *pluginSuite) Conditional() {
	s.headers.Set("If-Modified-Since", s.response.Header().Get("Last-Modified"))
	s.Request("GET", "/a/sample.bin", "")
}
func (s *pluginSuite) CustomCallback()                  { s.PythonCallback() }
func (s *pluginSuite) RequestHeader(name, value string) { s.headers.Set(name, value) }
func (s *pluginSuite) Header(name, value string) {
	s.Helper()
	if s.response.Header().Get(name) != value {
		s.Fatalf("expected header %s=%s", name, value)
	}
}
func (s *pluginSuite) HEAD() {
	s.Helper()
	length, err := strconv.Atoi(s.response.Header().Get("Content-Length"))
	if s.response.Body.Len() != 0 || err != nil || length <= 0 {
		s.Fatal("HEAD missing metadata or containing body")
	}
}
func (s *pluginSuite) CancelCallback() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	s.response = httptest.NewRecorder()
	s.handler(s.response, httptest.NewRequest("GET", "/custom/hello?spin=1", nil).WithContext(ctx))
}
func (s *pluginSuite) CloseTwice() { s.manager.Close(); s.manager.Close() }
func (s *pluginSuite) Counting()   { s.PythonCounting() }
func (s *pluginSuite) Concurrent(count int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results := make(chan int, int(count))
	var wg sync.WaitGroup
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response := httptest.NewRecorder()
			s.handler(response, httptest.NewRequest("GET", "/counter/value", nil).WithContext(ctx))
			value, err := strconv.Atoi(response.Body.String())
			if err != nil || response.Code != 200 {
				value = -1
			}
			results <- value
		}()
	}
	wg.Wait()
	close(results)
	for value := range results {
		s.results = append(s.results, value)
	}
}
func (s *pluginSuite) Counters(count int64) {
	s.Helper()
	seen := map[int]bool{}
	for _, value := range s.results {
		if value < 1 || value > int(count) || seen[value] {
			s.Fatalf("invalid or duplicate counter %d", value)
		}
		seen[value] = true
	}
	if len(seen) != int(count) {
		s.Fatalf("expected %d counters, got %d", count, len(seen))
	}
}
func (s *pluginSuite) Chain()            { s.PythonChain() }
func (s *pluginSuite) RememberRevision() { s.revision = s.manager.scripts[0].revision }
func (s *pluginSuite) ChangeParent()     { s.PythonChangeParent() }
func (s *pluginSuite) RevisionChanged() {
	s.Helper()
	if s.manager.scripts[0].revision == s.revision {
		s.Fatal("parent edits did not invalidate leaf cache")
	}
}
func (s *pluginSuite) Fork() {
	s.write("assetgoblin-fork.py", `plugin = {"api_version": 1, "extends": "middle", "extend": lambda p: p}`)
}
func (s *pluginSuite) Base()                   { s.Route("base", "/base/") }
func (s *pluginSuite) Extension(source string) { s.Source(source) }
func (s *pluginSuite) toolPlugin(declared bool) {
	s.ProcessInput("custom plugin logic")
	tools := "[]"
	if declared {
		tools = fmt.Sprintf("[%q]", s.tool)
	}
	s.Source(fmt.Sprintf(`plugin = {"api_version": 1, "tools": %s, "routes": [{"path": "/custom/", "directory": %q, "handle": lambda req: {"file": req["path"], "process": {"tool": %q, "extension": "txt", "args": ["-test.run=^TestPluginProcessHelper$", "--", "upper", "{input}", "{output}"]}}}]}`, tools, filepath.ToSlash(s.media), s.tool))
	s.Load()
	if s.loadErr != nil {
		s.Fatal(s.loadErr)
	}
}
func (s *pluginSuite) ToolPlugin()           { s.toolPlugin(true) }
func (s *pluginSuite) UndeclaredToolPlugin() { s.toolPlugin(false) }
func (s *pluginSuite) ProcessInput(body string) {
	s.Helper()
	path := filepath.Join(s.media, "source.txt")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		s.Fatal(err)
	}
	var err error
	s.source, err = os.Open(path)
	if err != nil {
		s.Fatal(err)
	}
}
func (s *pluginSuite) Process(mode, revision string) {
	s.Helper()
	if _, err := s.source.Seek(0, 0); err != nil {
		s.Fatal(err)
	}
	deadline := 5 * time.Second
	if mode == "wait" {
		deadline = 100 * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	s.output, s.processErr = runProcess(ctx, s.source, s.source.Name(), revision, s.cache, helperPlan(s.tool, mode))
}
func (s *pluginSuite) Output(body string) {
	s.Helper()
	if s.processErr != nil {
		s.Fatal(s.processErr)
	}
	data, err := os.ReadFile(s.output)
	if err != nil || string(data) != body {
		s.Fatalf("wrong process output %q (%v)", data, err)
	}
}
func (s *pluginSuite) RememberOutput() {
	s.Helper()
	if s.processErr != nil {
		s.Fatal(s.processErr)
	}
	s.remembered = s.output
}
func (s *pluginSuite) SameOutput() {
	s.Helper()
	if s.processErr != nil || s.output != s.remembered {
		s.Fatalf("cache miss: %s (%v)", s.output, s.processErr)
	}
}
func (s *pluginSuite) DifferentOutput() {
	s.Helper()
	if s.processErr != nil || s.output == s.remembered {
		s.Fatalf("cache revision unchanged (%v)", s.processErr)
	}
}
func (s *pluginSuite) CacheFiles(count int64) {
	s.Helper()
	entries, err := os.ReadDir(s.cache)
	if err != nil {
		s.Fatal(err)
	}
	if len(entries) != int(count) {
		s.Fatalf("expected %d cached outputs, got %v", count, entries)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "input-") || strings.HasPrefix(entry.Name(), "output-") {
			s.Fatalf("temporary file leaked: %s", entry.Name())
		}
	}
}
func (s *pluginSuite) ProcessFails() {
	s.Helper()
	if s.processErr == nil {
		s.Fatal("invalid or cancelled processing succeeded")
	}
}
func (s *pluginSuite) Plan(data string) {
	s.Helper()
	if err := json.Unmarshal([]byte(data), &s.plan); err != nil {
		s.Fatal(err)
	}
}
func (s *pluginSuite) ValidatePlan() { s.processErr = s.plan.validate() }
func (s *pluginSuite) pipeline(declared bool) {
	tools := map[string]bool{}
	if declared {
		tools[s.tool] = true
	}
	s.output, s.processErr = runPipeline(context.Background(), s.source, s.source.Name(), "revision", []processPlan{helperPlan(s.tool, "upper"), helperPlan(s.tool, "upper")}, tools)
}
func (s *pluginSuite) Pipeline()           { s.pipeline(true) }
func (s *pluginSuite) UndeclaredPipeline() { s.pipeline(false) }
