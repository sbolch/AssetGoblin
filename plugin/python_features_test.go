package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/regen-network/gocuke"
)

func TestPythonPluginFeatures(t *testing.T) {
	processHelper(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	gocuke.NewRunner(t, &pluginSuite{}).Path("../features/python/*.feature").
		Step(`^Python plugin source (.*)$`, (*pluginSuite).PythonSource).
		Step(`^an unrelated Python file$`, (*pluginSuite).PythonUnrelated).
		Step(`^plugin discovery runs$`, (*pluginSuite).Load).
		Step(`^(\d+) routes should be exposed$`, (*pluginSuite).RouteCount).
		Step(`^discovery should fail$`, (*pluginSuite).LoadFails).
		Step(`^a Python callback with a persistent counter$`, (*pluginSuite).PythonCallback).
		Step(`^a plugin "([^"]*)" request is sent to "([^"]*)" with range "([^"]*)"$`, (*pluginSuite).Request).
		Step(`^the plugin response status should be (\d+)$`, (*pluginSuite).Status).
		Step(`^the plugin response body should be "([^"]*)"$`, (*pluginSuite).Body).
		Step(`^request header "([^"]*)" is "([^"]*)"$`, (*pluginSuite).RequestHeader).
		Step(`^response header "([^"]*)" should be "([^"]*)"$`, (*pluginSuite).Header).
		Step(`^the callback response should have an empty body and positive content length$`, (*pluginSuite).HEAD).
		Step(`^a looping callback request is cancelled$`, (*pluginSuite).CancelCallback).
		Step(`^the manager is closed twice$`, (*pluginSuite).CloseTwice).
		Step(`^a Python counting callback$`, (*pluginSuite).PythonCounting).
		Step(`^(\d+) plugin requests arrive concurrently$`, (*pluginSuite).Concurrent).
		Step(`^every response should contain a different counter from 1 to (\d+)$`, (*pluginSuite).Counters).
		Step(`^a Python inheritance chain$`, (*pluginSuite).PythonChain).
		Step(`^its leaf revision is remembered$`, (*pluginSuite).RememberRevision).
		Step(`^the Python parent is changed$`, (*pluginSuite).PythonChangeParent).
		Step(`^its leaf revision should differ$`, (*pluginSuite).RevisionChanged).
		Step(`^an unregistered plugin with the same name$`, (*pluginSuite).PythonShadowExternal).
		Step(`^a separate unregistered plugin$`, (*pluginSuite).PythonAlongsideExternal).
		Step(`^an unbounded Python initialization$`, (*pluginSuite).PythonUnbounded).
		Step(`^an oversized Python source$`, (*pluginSuite).PythonOversized).
		Step(`^a Python uppercase tool plugin$`, (*pluginSuite).PythonTool).
		Step(`^discovery uses an unavailable Python interpreter$`, (*pluginSuite).PythonUnavailable).
		Step(`^an unregistered plugin$`, (*pluginSuite).PythonExternalOnly).
		Step(`^a valid Python codec$`, (*pluginSuite).PythonCodec).
		Step(`^a Python plugin using a dataclass$`, (*pluginSuite).PythonDataclass).
		Step(`^a Python plugin using an installed virtual environment package$`, (*pluginSuite).PythonVenv).
		Step(`^the Python codec is registered$`, (*pluginSuite).PythonCodecRegistered).
		Run()
}

func (s *pluginSuite) PythonSource(source string) { s.write("assetgoblin-custom.py", source) }
func (s *pluginSuite) PythonUnrelated()           { s.write("unrelated.py", "not valid Python") }
func (s *pluginSuite) PythonCallback() {
	s.Helper()
	if err := os.WriteFile(filepath.Join(s.media, "mapped.bin"), []byte("abcdef"), 0600); err != nil {
		s.Fatal(err)
	}
	s.PythonSource(fmt.Sprintf(`import os
count = 0
def handle(req):
    global count
    if req["query"].get("spin") == "1":
        while True: pass
    path = req["path"]
    if path == "file": return {"file": "mapped.bin"}
    if path == "escape": return {"file": "../secret.bin"}
    if path == "error": raise ValueError("private detail")
    if path == "crash": os._exit(1)
    if path == "invalid": return {"status": 99}
    if path == "header": return {"headers": {"X-Test": "bad\r\nvalue"}}
    if path == "oversized": return {"body": "x" * (1024 * 1024 + 1)}
    if path == "protocol": os.write(1, b"x" * (5 * 1024 * 1024)); return {}
    count += 1
    print("plugin diagnostic")
    return {"status": 201, "headers": {"X-Count": str(count)}, "body": ":".join([req["method"],path,req["query"].get("name",""),req["headers"].get("x-test","")])}
plugin = {"api_version": 1, "routes": [{"path": "/custom/", "directory": %q, "handle": handle}]}`, filepath.ToSlash(s.media)))
	s.Load()
	if s.loadErr != nil {
		s.Fatal(s.loadErr)
	}
}
func (s *pluginSuite) PythonCounting() {
	s.PythonSource(`count = 0
def handle(request):
    global count
    count += 1
    return {"body": str(count)}
plugin = {"api_version": 1, "routes": [{"path": "/counter/", "handle": handle}]}`)
	s.Load()
	if s.loadErr != nil {
		s.Fatal(s.loadErr)
	}
}
func (s *pluginSuite) PythonChain() {
	s.write("assetgoblin-zbase.py", `plugin = {"api_version": 1, "routes": [{"path": "/base/", "handle": lambda r: {"body": "base"}}]}`)
	for name, parent := range map[string]string{"middle": "zbase", "achild": "middle"} {
		suffix := name
		if name == "achild" {
			suffix = "child"
		}
		s.write("assetgoblin-"+name+".py", fmt.Sprintf(`def extend(parent):
    original = parent["routes"][0]["handle"]
    def handle(request):
        result = original(request)
        result["body"] += %q
        return result
    parent["routes"][0]["handle"] = handle
    return parent
plugin = {"api_version": 1, "extends": %q, "extend": extend}`, ":"+suffix, parent))
	}
}
func (s *pluginSuite) PythonChangeParent() {
	s.Helper()
	path := filepath.Join(s.directory, "assetgoblin-zbase.py")
	data, err := os.ReadFile(path)
	if err != nil {
		s.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte("\n# changed")...), 0600); err != nil {
		s.Fatal(err)
	}
}
func (s *pluginSuite) PythonShadowExternal() {
	s.write("assetgoblin-custom.external", "unregistered runtime ignored")
}
func (s *pluginSuite) PythonAlongsideExternal() {
	s.write("assetgoblin-secondary.external", "unregistered runtime input")
}
func (s *pluginSuite) PythonUnbounded() { s.PythonSource("while True: pass") }
func (s *pluginSuite) PythonOversized() { s.PythonSource("#" + string(make([]byte, maxScriptSize))) }
func (s *pluginSuite) PythonTool() {
	s.ProcessInput("custom plugin logic")
	s.PythonSource(fmt.Sprintf(`plugin = {"api_version": 1, "tools": [%q], "routes": [{"path": "/custom/", "directory": %q, "handle": lambda req: {"file": req["path"], "process": {"tool": %q, "extension": "txt", "args": ["-test.run=^TestPluginProcessHelper$", "--", "upper", "{input}", "{output}"]}}}]}`, s.tool, filepath.ToSlash(s.media), s.tool))
	s.Load()
	if s.loadErr != nil {
		s.Fatal(s.loadErr)
	}
}
func (s *pluginSuite) PythonUnavailable() {
	s.manager, s.loadErr = LoadWithPython(s.directory, filepath.Join(s.root, "missing-python"))
}
func (s *pluginSuite) PythonExternalOnly() {
	s.write("assetgoblin-external.external", "unregistered runtime input")
}
func (s *pluginSuite) PythonCodec() {
	s.PythonSource(`plugin = {"api_version": 1, "tools": ["tool"], "image_codecs": {"custom": {"mime_type": "image/custom", "decode": lambda r: {"tool": "tool", "extension": "png", "args": ["{input}", "{output}"]}, "encode": lambda r: {"tool": "tool", "extension": "custom", "args": ["{input}", "{output}"]}}}}`)
}
func (s *pluginSuite) PythonCodecRegistered() {
	s.Helper()
	if s.manager.ImageCodecs()["custom"] == nil {
		s.Fatal("Python image codec not registered")
	}
}

func (s *pluginSuite) PythonDataclass() {
	s.PythonSource(`from __future__ import annotations
from dataclasses import dataclass
@dataclass
class Greeting:
    body: str = "module works"
plugin = {"api_version": 1, "routes": [{"path": "/module/", "handle": lambda r: {"body": Greeting().body}}]}`)
}

func (s *pluginSuite) PythonVenv() {
	s.Helper()
	executable, err := exec.LookPath("python3")
	if err != nil {
		executable, err = exec.LookPath("python")
	}
	if err != nil {
		s.Fatal(err)
	}
	directory := filepath.Join(s.root, "virtual environment")
	if output, err := exec.Command(executable, "-m", "venv", "--without-pip", directory).CombinedOutput(); err != nil {
		s.Fatalf("create virtual environment: %v: %s", err, output)
	}
	selected := filepath.Join(directory, "bin", "python")
	if runtime.GOOS == "windows" {
		selected = filepath.Join(directory, "Scripts", "python.exe")
	}
	output, err := exec.Command(selected, "-I", "-c", `import sysconfig; print(sysconfig.get_paths()["purelib"])`).Output()
	if err != nil {
		s.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(strings.TrimSpace(string(output)), "assetgoblin_test_helper.py"), []byte(`greeting = "installed package"`), 0600); err != nil {
		s.Fatal(err)
	}
	s.PythonSource(`import assetgoblin_test_helper
plugin = {"api_version": 1, "routes": [{"path": "/venv/", "handle": lambda r: {"body": assetgoblin_test_helper.greeting}}]}`)
	s.manager, s.loadErr = LoadWithPython(s.directory, selected)
	if s.loadErr != nil {
		s.Fatal(s.loadErr)
	}
	s.handler = s.manager.Handlers()[0].Handler()
}
