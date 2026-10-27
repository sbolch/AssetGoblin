package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/regen-network/gocuke"
)

func TestUpdaterFeatures(t *testing.T) {
	gocuke.NewRunner(t, &updaterSuite{}).Path("features/updater/*.feature").Before((*updaterSuite).Before).After((*updaterSuite).After).
		Step(`^an updater release for "([^"]*)" and "([^"]*)"$`, (*updaterSuite).Release).
		Step(`^the update fixture has "([^"]*)"$`, (*updaterSuite).Problem).
		Step(`^the release is installed into a temporary executable$`, (*updaterSuite).Install).
		Step(`^the update should (succeed|fail)$`, (*updaterSuite).Result).
		Step(`^the installed executable should contain "([^"]*)"$`, (*updaterSuite).Executable).
		Step(`^the download progress should finish successfully$`, (*updaterSuite).Progress).
		Step(`^the download progress should report failure$`, (*updaterSuite).ProgressFailure).
		Step(`^the current build is "([^"]*)"$`, (*updaterSuite).Current).
		Step(`^an update should be (available|unavailable)$`, (*updaterSuite).Available).
		Run()
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type updaterSuite struct {
	gocuke.TestingT
	transport                                 http.RoundTripper
	version, dir, target, goos, arch, problem string
	data                                      []byte
	updater                                   *selfupdate.Updater
	latest                                    *selfupdate.Release
	output                                    bytes.Buffer
	err                                       error
}

func (s *updaterSuite) Before() {
	s.transport, s.version = http.DefaultTransport, Version
	s.problem, s.err, s.latest = "", nil, nil
	s.output.Reset()
	var err error
	s.dir, err = os.MkdirTemp("", "assetgoblin-updater-features-")
	if err != nil {
		s.Fatal(err)
	}
}
func (s *updaterSuite) After() {
	http.DefaultTransport, Version = s.transport, s.version
	os.RemoveAll(s.dir)
}
func (s *updaterSuite) Release(goos, arch string) {
	s.Helper()
	s.goos, s.arch = goos, arch
	s.target = filepath.Join(s.dir, "assetgoblin")
	if goos == "windows" {
		s.target += ".exe"
	}
	if err := os.WriteFile(s.target, []byte("original executable"), 0755); err != nil {
		s.Fatal(err)
	}
	var archive bytes.Buffer
	name := filepath.Base(s.target)
	if goos == "windows" {
		writer := zip.NewWriter(&archive)
		file, err := writer.Create(name)
		if err != nil {
			s.Fatal(err)
		}
		if _, err := io.WriteString(file, "updated executable"); err != nil {
			s.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			s.Fatal(err)
		}
	} else {
		compressed := gzip.NewWriter(&archive)
		writer := tar.NewWriter(compressed)
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len("updated executable"))}); err != nil {
			s.Fatal(err)
		}
		if _, err := io.WriteString(writer, "updated executable"); err != nil {
			s.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			s.Fatal(err)
		}
		if err := compressed.Close(); err != nil {
			s.Fatal(err)
		}
	}
	s.data = archive.Bytes()
	http.DefaultTransport = roundTripFunc(s.respond)
	var err error
	s.updater, err = newUpdater(goos, arch, &s.output)
	if err != nil {
		s.Fatal(err)
	}
}
func (s *updaterSuite) Problem(problem string) { s.problem = problem }
func (s *updaterSuite) respond(req *http.Request) (*http.Response, error) {
	if req.URL.Host != "api.github.com" {
		return nil, fmt.Errorf("unexpected update host %q", req.URL.Host)
	}
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	name := fixtureArchiveName("v1.2.3", s.goos, s.arch)
	data := s.data
	if s.problem == "damaged archive" {
		data = []byte("damaged archive")
	}
	sum := fmt.Sprintf("%x  %s\n", sha256.Sum256(data), name)
	status, contentType := 200, "application/json"
	var body []byte
	switch {
	case strings.HasSuffix(req.URL.Path, "/releases/assets/1"):
		contentType, body = "application/octet-stream", data
		if s.problem == "download failure" {
			status = 503
		}
		if s.problem == "damaged archive" {
			body = []byte("damaged archive")
		}
	case strings.HasSuffix(req.URL.Path, "/releases/assets/2"):
		contentType = "application/octet-stream"
		if s.problem == "checksum mismatch" {
			sum = strings.Repeat("0", 64) + "  " + name
		}
		if s.problem == "missing checksum entry" {
			sum = strings.Repeat("0", 64) + "  other.tar.gz"
		}
		body = []byte(sum)
	default:
		assets := []any{map[string]any{"id": 1, "name": name, "size": len(s.data), "browser_download_url": "https://example.com/" + name}}
		if s.problem == "missing platform archive" {
			assets = nil
		}
		if s.problem != "missing checksum document" {
			assets = append(assets, map[string]any{"id": 2, "name": "checksums.txt", "browser_download_url": "https://example.com/checksums.txt"})
		}
		// Newer drafts and prereleases must not displace a stable release.
		releases := []any{map[string]any{"id": 12, "tag_name": "v9.0.0", "draft": true, "assets": assets}, map[string]any{"id": 11, "tag_name": "v8.0.0", "prerelease": true, "assets": assets}, map[string]any{"id": 10, "tag_name": "v1.2.3", "assets": assets}}
		if s.problem == "no releases" {
			releases = nil
		}
		var err error
		body, err = json.Marshal(releases)
		if err != nil {
			return nil, err
		}
	}
	reader := io.NopCloser(bytes.NewReader(body))
	if s.problem == "interrupted download" && strings.HasSuffix(req.URL.Path, "/releases/assets/1") {
		reader = io.NopCloser(io.MultiReader(bytes.NewReader(body[:10]), brokenDownload{}))
	}
	return &http.Response{StatusCode: status, Status: fmt.Sprintf("%d %s", status, http.StatusText(status)), Header: http.Header{"Content-Type": []string{contentType}}, Body: reader, Request: req}, nil
}
func (s *updaterSuite) Install() {
	s.latest, s.err = detectLatest(s.updater)
	if s.err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if s.problem == "cancelled download" {
		cancel()
	}
	s.err = s.updater.UpdateTo(ctx, s.latest, s.target)
}
func (s *updaterSuite) Result(result string) {
	s.Helper()
	if (s.err == nil) != (result == "succeed") {
		s.Fatalf("update should %s: %v", result, s.err)
	}
}
func (s *updaterSuite) Executable(expected string) {
	s.Helper()
	got, err := os.ReadFile(s.target)
	if err != nil || string(got) != expected {
		s.Fatalf("executable %q, error %v", got, err)
	}
	if expected == "updated executable" && s.goos != "windows" {
		info, err := os.Stat(s.target)
		if err != nil || info.Mode().Perm()&0111 == 0 {
			s.Fatalf("updated file is not executable: %v", err)
		}
	}
}
func (s *updaterSuite) Progress() {
	s.Helper()
	if !strings.Contains(s.output.String(), "100%") || strings.Contains(s.output.String(), "Download failed.") {
		s.Fatalf("unexpected progress: %s", s.output.String())
	}
}
func (s *updaterSuite) Current(version string) { Version = version }
func (s *updaterSuite) Available(expected string) {
	s.Helper()
	latest, err := detectLatest(s.updater)
	if err != nil {
		s.Fatal(err)
	}
	if got := updateAvailable(latest); got != (expected == "available") {
		s.Fatalf("update available: %v", got)
	}
}

type brokenDownload struct{}

func (brokenDownload) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (s *updaterSuite) ProgressFailure() {
	s.Helper()
	if !strings.Contains(s.output.String(), "Download failed.") {
		s.Fatalf("missing download failure: %s", s.output.String())
	}
}
