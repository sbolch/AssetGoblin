package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/regen-network/gocuke"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	http.DefaultTransport = releaseTransport{status: 200, body: `{"tag_name":"development","assets":[]}`}
	main()
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
	gocuke.NewRunner(t, &applicationSuite{}).Path(feature).Before((*applicationSuite).Before).After((*applicationSuite).After).
		Step(`^AssetGoblin runs with arguments "([^"]*)"$`, (*applicationSuite).RunCLI).
		Step(`^the command should exit with code (\d+)$`, (*applicationSuite).ExitCode).
		Step(`^command output should contain "([^"]*)"$`, (*applicationSuite).Output).
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
		Step(`^checksum lines "([^"]*)"$`, (*applicationSuite).ChecksumLines).
		Step(`^the checksum for "([^"]*)" is selected$`, (*applicationSuite).SelectChecksum).
		Step(`^its checksum should be "([^"]*)"$`, (*applicationSuite).Checksum).
		Step(`^its checksum should be missing$`, (*applicationSuite).ChecksumMissing).
		Step(`^a release asset "([^"]*)" at "([^"]*)"$`, (*applicationSuite).Asset).
		Step(`^the update URL for "([^"]*)" is selected$`, (*applicationSuite).SelectUpdateURL).
		Step(`^the checksum URL is selected$`, (*applicationSuite).SelectChecksumURL).
		Step(`^the selected asset URL should be "([^"]*)"$`, (*applicationSuite).AssetURL).
		Step(`^no matching asset should be found$`, (*applicationSuite).AssetMissing).
		Step(`^the startup banner should be present$`, (*applicationSuite).Banner).
		Step(`^the current build version should be reported$`, (*applicationSuite).BuildVersion).
		Run()
}

type releaseTransport struct {
	status int
	body   string
}

func (t releaseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: t.status, Status: fmt.Sprintf("%d %s", t.status, http.StatusText(t.status)), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(t.body)), Request: req}, nil
}

type applicationSuite struct {
	assets     []releaseAsset
	assetURL   string
	assetFound bool
	gocuke.TestingT
	output, directory, version, archive, checksum string
	exit                                          int
	err                                           error
	checksums                                     map[string]string
	transport                                     http.RoundTripper
	oldRelease                                    *release
	dir                                           string
}

func (s *applicationSuite) Before() {
	s.transport = http.DefaultTransport
	s.oldRelease = r
	s.assets = nil
	s.assetURL = ""
	s.assetFound = false
	s.err = nil
	s.output = ""
	s.exit = 0
	var err error
	s.dir, err = os.MkdirTemp("", "assetgoblin-cli-features-")
	if err != nil {
		s.Fatal(err)
	}
}
func (s *applicationSuite) After() {
	http.DefaultTransport = s.transport
	r = s.oldRelease
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
	body, err := cmd.CombinedOutput()
	s.output = string(body)
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
	http.DefaultTransport = releaseTransport{status: int(status), body: body}
}
func (s *applicationSuite) Latest() { s.version, s.err = getLatestVersion() }
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
	s.archive = releaseArchiveName(version, goos, arch)
}
func (s *applicationSuite) ArchiveName(name string) {
	s.Helper()
	if s.archive != name {
		s.Fatalf("wrong archive name %s", s.archive)
	}
}
func (s *applicationSuite) ChecksumLines(lines string) {
	s.checksums = parseChecksums(strings.ReplaceAll(lines, ";", "\n"))
}
func (s *applicationSuite) SelectChecksum(name string) { s.checksum = s.checksums[name] }
func (s *applicationSuite) Checksum(value string) {
	s.Helper()
	if s.checksum != value {
		s.Fatalf("wrong checksum %q", s.checksum)
	}
}
func (s *applicationSuite) ChecksumMissing() {
	s.Helper()
	if s.checksum != "" {
		s.Fatal("missing archive unexpectedly has a checksum")
	}
}

func (s *applicationSuite) Asset(name, url string) {
	s.assets = append(s.assets, releaseAsset{Name: name, BrowserDownloadURL: url})
}
func (s *applicationSuite) SelectUpdateURL(name string) {
	s.assetURL, s.assetFound = findUpdateURL(s.assets, name)
}
func (s *applicationSuite) SelectChecksumURL() { s.assetURL, s.assetFound = findChecksumURL(s.assets) }
func (s *applicationSuite) AssetURL(url string) {
	s.Helper()
	if !s.assetFound || s.assetURL != url {
		s.Fatalf("wrong selected asset URL %q (found %v)", s.assetURL, s.assetFound)
	}
}
func (s *applicationSuite) AssetMissing() {
	s.Helper()
	if s.assetFound || s.assetURL != "" {
		s.Fatal("unexpected matching release asset")
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
