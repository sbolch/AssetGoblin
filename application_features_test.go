package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/regen-network/gocuke"

	"assetgoblin/config"
	"assetgoblin/utils"
)

// TestCLIProcessHelper is a subprocess harness; scenarios define every CLI assertion.
func TestCLIProcessHelper(t *testing.T) {
	if os.Getenv("ASSETGOBLIN_CLI_HELPER") != "1" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("ASSETGOBLIN_CLI_ARGS")), &args); err != nil {
		os.Exit(3)
	}
	flag.CommandLine = flag.NewFlagSet("assetgoblin", flag.ExitOnError)
	os.Args = append([]string{"assetgoblin"}, args...)
	transport := releaseTransport{Status: 200, Body: `{"tag_name":"v1.0.0","assets":[]}`}
	if version := os.Getenv("ASSETGOBLIN_CLI_VERSION"); version != "" {
		Version = version
	}
	if value := os.Getenv("ASSETGOBLIN_CLI_RELEASE"); value != "" {
		if err := json.Unmarshal([]byte(value), &transport); err != nil {
			os.Exit(3)
		}
	}
	http.DefaultTransport = transport
	main()
	os.Exit(0)
}
func TestApplicationFeatures(t *testing.T) {
	feature, err := filepath.Abs("features/application/*.feature")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ProgramData", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("GORACE", "atexit_sleep_ms=0")
	t.Setenv("NO_COLOR", "1")
	gocuke.NewRunner(t, &applicationSuite{}).Path(feature).Before((*applicationSuite).Before).After((*applicationSuite).After).
		Step(`^AssetGoblin runs with arguments "([^"]*)"$`, (*applicationSuite).RunCLI).
		Step(`^the command should exit with code (\d+)$`, (*applicationSuite).ExitCode).
		Step(`^a CLI Python plugin$`, (*applicationSuite).PythonPlugin).
		Step(`^command output should contain "([^"]*)"$`, (*applicationSuite).Output).
		Step(`^command output should not contain "([^"]*)"$`, (*applicationSuite).OutputAbsent).
		Step(`^command output should contain no terminal escape sequences$`, (*applicationSuite).NoEscapes).
		Step(`^command (stdout|stderr) should be empty$`, (*applicationSuite).EmptyStream).
		Step(`^the first output line should be the current version$`, (*applicationSuite).VersionFirstLine).
		Step(`^CLI configuration contains (.*)$`, (*applicationSuite).CLIConfig).
		Step(`^the CLI build version is "([^"]*)"$`, (*applicationSuite).CLIVersion).
		Step(`^CLI colors are disabled$`, (*applicationSuite).DisableColors).
		Step(`^the default plugin discovery directory is resolved$`, (*applicationSuite).DefaultPluginDirectory).
		Step(`^plugin discovery directory "([^"]*)" is resolved$`, (*applicationSuite).PluginDirectory).
		Step(`^it should be the executable directory$`, (*applicationSuite).ExecutableDirectory).
		Step(`^it should be the working directory$`, (*applicationSuite).WorkingDirectory).
		Step(`^the release API responds with status (\d+) and body (.*)$`, (*applicationSuite).ReleaseAPI).
		Step(`^the latest version is checked$`, (*applicationSuite).Latest).
		Step(`^the latest version should be "([^"]*)"$`, (*applicationSuite).Version).
		Step(`^the version check should fail$`, (*applicationSuite).VersionFailure).
		Step(`^the update archive for "([^"]*)" and "([^"]*)" at version "([^"]*)" is selected$`, (*applicationSuite).Archive).
		Step(`^the archive filename should be "([^"]*)"$`, (*applicationSuite).ArchiveName).
		Step(`^the startup banner should be present$`, (*applicationSuite).Banner).
		Step(`^the current build version should be reported$`, (*applicationSuite).BuildVersion).
		Step(`^a download of (\d+) bytes with advertised size (-?\d+) uses (terminal|redirected) progress and (succeeds|fails)$`, (*applicationSuite).DownloadProgress).
		Step(`^download progress should (refresh|not refresh) in place$`, (*applicationSuite).ProgressRefresh).
		Step(`^a startup summary for configuration (.*)$`, (*applicationSuite).StartupSummary).
		Step(`^release notes "([^"]*)" are rendered at width (\d+)$`, (*applicationSuite).ReleaseNotes).
		Step(`^every output line should fit within (\d+) columns$`, (*applicationSuite).OutputWidth).
		Run()
}

type releaseTransport struct {
	Status int
	Body   string
}

func (t releaseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Existing CLI fixtures describe a single release; GitHub's listing API returns an array.
	var release map[string]any
	if json.Unmarshal([]byte(t.Body), &release) == nil && release != nil {
		if tag, ok := release["tag_name"].(string); ok && tag != "" {
			release["assets"] = []any{map[string]any{"id": 1, "name": fixtureArchiveName(tag, runtime.GOOS, runtime.GOARCH)}, map[string]any{"id": 2, "name": "checksums.txt"}}
		}
		encoded, err := json.Marshal([]any{release})
		if err != nil {
			return nil, err
		}
		t.Body = string(encoded)
	}

	return &http.Response{StatusCode: t.Status, Status: fmt.Sprintf("%d %s", t.Status, http.StatusText(t.Status)), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(t.Body)), Request: req}, nil
}

type applicationSuite struct {
	gocuke.TestingT
	output, directory, version, archive string
	stdout, stderr                      string
	exit                                int
	err                                 error
	transport                           http.RoundTripper
	noColor                             bool
	cliVersion                          string
	dir                                 string
}

func (s *applicationSuite) Before() {
	s.transport = http.DefaultTransport
	s.err = nil
	s.output = ""
	s.stdout, s.stderr = "", ""
	s.exit = 0
	s.noColor = false
	s.cliVersion = ""
	var err error
	s.dir, err = os.MkdirTemp("", "assetgoblin-cli-features-")
	if err != nil {
		s.Fatal(err)
	}
}
func (s *applicationSuite) After() {
	http.DefaultTransport = s.transport
	os.RemoveAll(s.dir)
}
func (s *applicationSuite) RunCLI(args string) {
	s.Helper()
	encoded, err := json.Marshal(strings.Fields(args))
	if err != nil {
		s.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		s.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestCLIProcessHelper$")
	cmd.Dir = s.dir
	cmd.Env = append(os.Environ(), "ASSETGOBLIN_CLI_HELPER=1", "ASSETGOBLIN_CLI_ARGS="+string(encoded))
	if transport, ok := http.DefaultTransport.(releaseTransport); ok {
		mock, err := json.Marshal(transport)
		if err != nil {
			s.Fatal(err)
		}
		cmd.Env = append(cmd.Env, "ASSETGOBLIN_CLI_RELEASE="+string(mock))
	}
	if s.cliVersion != "" {
		cmd.Env = append(cmd.Env, "ASSETGOBLIN_CLI_VERSION="+s.cliVersion)
	}
	if s.noColor {
		cmd.Env = append(cmd.Env, "NO_COLOR=1")
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	s.stdout, s.stderr = stdout.String(), stderr.String()
	s.output = s.stdout + s.stderr
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			s.exit = exit.ExitCode()
		} else {
			s.Fatal(err)
		}
	}
}
func (s *applicationSuite) ExitCode(expected int64) {
	s.Helper()
	if s.exit != int(expected) {
		s.Fatalf("expected exit %d, got %d: %s", expected, s.exit, s.output)
	}
}
func (s *applicationSuite) Output(text string) {
	s.Helper()
	if !strings.Contains(s.output, text) {
		s.Fatalf("missing output %q: %s", text, s.output)
	}
}
func (s *applicationSuite) DefaultPluginDirectory()     { s.directory, s.err = pluginDirectory("") }
func (s *applicationSuite) PluginDirectory(path string) { s.directory, s.err = pluginDirectory(path) }
func (s *applicationSuite) ExecutableDirectory() {
	s.Helper()
	exe, err := os.Executable()
	if err != nil {
		s.Fatal(err)
	}
	if s.err != nil || s.directory != filepath.Dir(exe) {
		s.Fatalf("wrong plugin directory %s (%v)", s.directory, s.err)
	}
}
func (s *applicationSuite) WorkingDirectory() {
	s.Helper()
	wd, err := os.Getwd()
	if err != nil {
		s.Fatal(err)
	}
	if s.err != nil || s.directory != wd {
		s.Fatalf("wrong override %s (%v)", s.directory, s.err)
	}
}
func (s *applicationSuite) ReleaseAPI(status int64, body string) {
	http.DefaultTransport = releaseTransport{Status: int(status), Body: body}
}
func (s *applicationSuite) Latest() {
	latest, err := getLatestRelease()
	s.err = err
	if err == nil {
		s.version = "v" + latest.Version()
	}
}
func (s *applicationSuite) Version(version string) {
	s.Helper()
	if s.err != nil || s.version != version {
		s.Fatalf("expected version %s, got %s (%v)", version, s.version, s.err)
	}
}
func (s *applicationSuite) VersionFailure() {
	s.Helper()
	if s.err == nil {
		s.Fatal("expected release lookup error")
	}
}
func (s *applicationSuite) Archive(goos, arch, version string) {
	name := fixtureArchiveName(version, goos, arch)
	http.DefaultTransport = releaseTransport{Status: 200, Body: fmt.Sprintf(`[{"tag_name":%q,"assets":[{"id":1,"name":%q},{"id":2,"name":"checksums.txt"}]}]`, version, name)}
	updater, err := newUpdater(goos, arch, nil)
	if err != nil {
		s.Fatal(err)
	}
	latest, err := detectLatest(updater)
	if err != nil {
		s.Fatal(err)
	}
	s.archive = latest.AssetName
}
func (s *applicationSuite) ArchiveName(name string) {
	s.Helper()
	if s.archive != name {
		s.Fatalf("wrong archive name %s", s.archive)
	}
}
func (s *applicationSuite) Banner() {
	s.Helper()
	if Logo == "" || !strings.Contains(s.output, Logo) {
		s.Fatal("startup banner missing")
	}
}
func (s *applicationSuite) BuildVersion() {
	s.Helper()
	if Version == "" || !strings.Contains(s.output, Version) {
		s.Fatal("current build version missing")
	}
}

func (s *applicationSuite) CLIConfig(content string) {
	s.Helper()
	if err := os.WriteFile(filepath.Join(s.dir, "config.json"), []byte(content), 0600); err != nil {
		s.Fatal(err)
	}
}

func (s *applicationSuite) DisableColors() { s.noColor = true }

func (s *applicationSuite) NoEscapes() {
	s.Helper()
	if strings.ContainsRune(s.output, '\x1b') {
		s.Fatal("unexpected terminal escape sequences in output")
	}
}

func (s *applicationSuite) OutputAbsent(text string) {
	s.Helper()
	if strings.Contains(s.output, text) {
		s.Fatalf("unexpected output %q: %s", text, s.output)
	}
}

func (s *applicationSuite) EmptyStream(stream string) {
	s.Helper()
	output := s.stdout
	if stream == "stderr" {
		output = s.stderr
	}
	if output != "" {
		s.Fatalf("unexpected %s output: %s", stream, output)
	}
}

func (s *applicationSuite) VersionFirstLine() {
	s.Helper()
	line, _, _ := strings.Cut(s.stdout, "\n")
	if line != Version {
		s.Fatalf("expected first line %q, got %q", Version, line)
	}
}

func (s *applicationSuite) DownloadProgress(size, total int64, destination, result string) {
	s.Helper()
	var output, archive bytes.Buffer
	progress := utils.NewUpdateProgress(&output, total, destination == "terminal", cliHeading)
	var source io.Reader = strings.NewReader(strings.Repeat("x", int(size)))
	if result == "fails" {
		source = io.MultiReader(source, interruptedDownload{})
	}
	n, err := io.Copy(&archive, io.TeeReader(source, progress))
	if (err != nil) != (result == "fails") || n != size || int64(archive.Len()) != size {
		s.Fatalf("progress changed copied content: bytes=%d error=%v", n, err)
	}
	progress.Finish(err == nil)
	s.output = output.String()
}

func (s *applicationSuite) ProgressRefresh(expected string) {
	s.Helper()
	if strings.Contains(s.output, "\r") != (expected == "refresh") {
		s.Fatalf("unexpected progress refresh: %q", s.output)
	}
}

type interruptedDownload struct{}

func (interruptedDownload) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func (s *applicationSuite) StartupSummary(data string) {
	s.Helper()
	var cfg config.Config
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		s.Fatal(err)
	}
	var output bytes.Buffer
	printServerSummary(&output, &cfg, ":"+cfg.Port, []string{"/video/", "/audio/"}, []string{"zoi", "qoi"})
	s.output = output.String()
}

func (s *applicationSuite) ReleaseNotes(body string, width int64) {
	var output bytes.Buffer
	printReleaseNotes(&output, body, int(width))
	s.output = output.String()
}

func (s *applicationSuite) OutputWidth(width int64) {
	s.Helper()
	for _, line := range strings.Split(s.output, "\n") {
		if lipgloss.Width(line) > int(width) {
			s.Fatalf("line exceeds %d columns: %q", width, line)
		}
	}
}

func (s *applicationSuite) PythonPlugin() {
	s.Helper()
	source := `plugin = {"api_version": 1, "routes": [{"path": "/test/", "directory": "."}]}`
	if err := os.WriteFile(filepath.Join(s.dir, "assetgoblin-test.py"), []byte(source), 0600); err != nil {
		s.Fatal(err)
	}
}

// fixtureArchiveName independently models the published GoReleaser archive names.
func fixtureArchiveName(version, goos, arch string) string {
	osName := strings.ToUpper(goos[:1]) + goos[1:]
	if goos == "darwin" {
		osName = "macOS"
	}
	suffix := ".tar.gz"
	if goos == "windows" {
		suffix = ".zip"
	}
	return "AssetGoblin_" + version + "_" + osName + "_" + map[string]string{"amd64": "x64", "386": "x86", "arm64": "arm64"}[arch] + suffix
}

func (s *applicationSuite) CLIVersion(version string) { s.cliVersion = version }
