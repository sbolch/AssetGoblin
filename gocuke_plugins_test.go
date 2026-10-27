package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	stdimage "image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"assetgoblin/image"
	"assetgoblin/plugin"

	"github.com/go-fuego/fuego"
)

func (s *assetgoblinSuite) featureServer() (*fuego.Server, error) {
	if s.pluginLoadErr != nil {
		return nil, s.pluginLoadErr
	}
	var codecs map[string]image.Codec
	var handlers []plugin.RouteHandler
	if s.pluginManager != nil {
		codecs, handlers = s.pluginManager.ImageCodecs(), s.pluginManager.Handlers()
	}
	server, cleanup, err := newServer(s.config, codecs, handlers...)
	if cleanup != nil {
		s.serverCleanups = append(s.serverCleanups, cleanup)
	}
	return server, err
}

func (s *assetgoblinSuite) pluginDirectory() string {
	s.Helper()
	if s.pluginDir == "" {
		s.pluginDir = filepath.Join(s.testDir, "plugins")
		if err := os.MkdirAll(s.pluginDir, 0755); err != nil {
			s.Fatal(err)
		}
	}
	return s.pluginDir
}

func (s *assetgoblinSuite) ExamplePluginInstalled(name string) {
	s.Helper()
	sourceDir := ""
	if name == "audio" || name == "video" {
		sourceDir = filepath.Join(s.testDir, "assets", name)
	}
	copyExample(s, s.pluginDirectory(), name, sourceDir)
}

func (s *assetgoblinSuite) LoadRuntimePlugins() {
	s.Helper()
	if s.server != nil {
		s.server.Close()
		s.server = nil
	}
	if s.pluginManager != nil {
		s.pluginManager.Close()
	}
	s.pluginManager, s.pluginLoadErr = plugin.Load(s.pluginDirectory())
}
func (s *assetgoblinSuite) PluginRouteCount(count int64) {
	s.Helper()
	if s.pluginLoadErr != nil {
		s.Fatal(s.pluginLoadErr)
	}
	if s.pluginManager == nil || len(s.pluginManager.Handlers()) != int(count) {
		s.Fatalf("expected %d plugin routes", count)
	}
}
func (s *assetgoblinSuite) ImageCodecRegistered(name string) {
	s.Helper()
	if s.pluginManager == nil || s.pluginManager.ImageCodecs()[name] == nil {
		s.Fatalf("codec %q not registered", name)
	}
}
func (s *assetgoblinSuite) writeRuntimePlugin(name, source string) {
	s.Helper()
	if err := os.WriteFile(filepath.Join(s.pluginDirectory(), "assetgoblin-"+name+".py"), []byte(source), 0600); err != nil {
		s.Fatal(err)
	}
}
func (s *assetgoblinSuite) ParentPlugin(body, prefix string) {
	s.writeRuntimePlugin("parent", fmt.Sprintf(`plugin = {"api_version": 1, "routes": [{"path": %q, "handle": lambda r: {"body": %q}}]}`, prefix, body))
}
func (s *assetgoblinSuite) ChildPlugin(suffix string) {
	s.writeRuntimePlugin("child", fmt.Sprintf(`def extend(parent):
    original = parent["routes"][0]["handle"]
    def handle(request):
        result = original(request)
        result["body"] += %q
        return result
    parent["routes"][0]["handle"] = handle
    return parent
plugin = {"api_version": 1, "extends": "parent", "extend": extend}`, suffix))
}
func (s *assetgoblinSuite) CyclicPlugin() {
	s.writeRuntimePlugin("cycle", `plugin = {"api_version": 1, "extends": "cycle", "extend": lambda p: p}`)
}
func (s *assetgoblinSuite) PluginLoadingFails(message string) {
	s.Helper()
	if s.pluginLoadErr == nil || !strings.Contains(s.pluginLoadErr.Error(), message) {
		s.Fatalf("expected load error containing %q, got %v", message, s.pluginLoadErr)
	}
}
func (s *assetgoblinSuite) ServerSigningSecret(secret string) { s.config.Secret = secret }
func (s *assetgoblinSuite) requireTools(names ...string) {
	s.Helper()
	for _, name := range names {
		if _, err := exec.LookPath(name); err != nil {
			s.Skip(name + " unavailable")
		}
	}
}
func (s *assetgoblinSuite) ImageConversionTools() {
	tool := "convert"
	if runtime.GOOS == "windows" {
		tool = "magick"
	}
	s.requireTools("ffmpeg", tool)
}
func (s *assetgoblinSuite) QOAConversionTools() { s.requireTools("ffmpeg", "qoaconv") }
func (s *assetgoblinSuite) PNGSource(name string, width, height int64) {
	s.Helper()
	s.config.Image.Formats = []string{"png"}
	if err := os.MkdirAll(s.config.Image.Directory, 0755); err != nil {
		s.Fatal(err)
	}
	file, err := os.Create(filepath.Join(s.config.Image.Directory, name))
	if err != nil {
		s.Fatal(err)
	}
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, int(width), int(height)))
	for y := 0; y < int(height); y++ {
		for x := 0; x < int(width); x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	encodeErr := png.Encode(file, img)
	closeErr := file.Close()
	if encodeErr != nil {
		s.Fatal(encodeErr)
	}
	if closeErr != nil {
		s.Fatal(closeErr)
	}
}
func (s *assetgoblinSuite) pluginBody() []byte {
	s.Helper()
	if s.lastResp == nil {
		s.Fatal("no response")
	}
	if s.pluginResponse == nil {
		data, err := io.ReadAll(s.lastResp.Body)
		s.lastResp.Body.Close()
		if err != nil {
			s.Fatal(err)
		}
		s.pluginResponse = data
		if s.lastResp.Request != nil && s.lastResp.Request.Method == "GET" && s.lastResp.StatusCode == 200 {
			s.pluginAssetLength = len(data)
		}
	}
	return s.pluginResponse
}
func (s *assetgoblinSuite) QOIResponse(width, height int64) {
	s.Helper()
	data := s.pluginBody()
	if len(data) < 14 || string(data[:4]) != "qoif" {
		s.Fatalf("invalid QOI response %q", data)
	}
	if binary.BigEndian.Uint32(data[4:8]) != uint32(width) || binary.BigEndian.Uint32(data[8:12]) != uint32(height) {
		s.Fatalf("wrong QOI dimensions: %v", data[:14])
	}
}
func (s *assetgoblinSuite) PNGResponse(width, height int64) {
	s.Helper()
	img, err := png.Decode(bytes.NewReader(s.pluginBody()))
	if err != nil {
		s.Fatal(err)
	}
	if img.Bounds().Dx() != int(width) || img.Bounds().Dy() != int(height) {
		s.Fatalf("wrong PNG dimensions: %v", img.Bounds())
	}
}
func (s *assetgoblinSuite) ResponseContentType(expected string) {
	s.Helper()
	if s.lastResp == nil || s.lastResp.Header.Get("Content-Type") != expected {
		s.Fatalf("expected content type %q", expected)
	}
}
func (s *assetgoblinSuite) pluginRequest(method, path, byteRange string) {
	s.Helper()
	s.pluginResponse = nil
	baseURL := s.liveURL
	if baseURL == "" {
		if s.server == nil {
			server, err := s.featureServer()
			if err != nil {
				s.Fatal(err)
			}
			s.server = httptest.NewServer(server.Handler)
		}
		baseURL = s.server.URL
	}
	req, err := http.NewRequest(method, baseURL+path, nil)
	if err != nil {
		s.Fatal(err)
	}
	if byteRange != "" {
		req.Header.Set("Range", "bytes="+byteRange)
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		s.Fatal(err)
	}
	if s.lastResp != nil {
		s.lastResp.Body.Close()
	}
	s.lastResp = resp
}
func (s *assetgoblinSuite) PluginHEAD(path string) { s.pluginRequest("HEAD", path, "") }
func (s *assetgoblinSuite) PluginRange(byteRange, path string) {
	s.pluginRequest("GET", path, byteRange)
}
func (s *assetgoblinSuite) EmptyHEADResponse() {
	s.Helper()
	length, err := strconv.Atoi(s.lastResp.Header.Get("Content-Length"))
	if len(s.pluginBody()) != 0 || err != nil || length <= 0 || (s.pluginAssetLength > 0 && length != s.pluginAssetLength) {
		s.Fatal("HEAD lost content length or returned body")
	}
}
func (s *assetgoblinSuite) saveResponse(directory, name string) {
	s.Helper()
	if err := os.MkdirAll(directory, 0755); err != nil {
		s.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name), s.pluginBody(), 0600); err != nil {
		s.Fatal(err)
	}
}
func (s *assetgoblinSuite) SaveImageResponse(name string) {
	s.saveResponse(s.config.Image.Directory, name)
}
func (s *assetgoblinSuite) SaveAudioResponse(name string) {
	s.saveResponse(filepath.Join(s.testDir, "assets/audio"), name)
}
func (s *assetgoblinSuite) WAVSource() {
	s.Helper()
	dir := filepath.Join(s.testDir, "assets/audio")
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-loglevel", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-ar", "48000", "-c:a", "pcm_s16le", filepath.Join(dir, "source.wav")).CombinedOutput()
	if err != nil {
		s.Fatalf("audio fixture: %v %s", err, output)
	}
}
func (s *assetgoblinSuite) AudioSource(name string) {
	s.Helper()
	dir := filepath.Join(s.testDir, "assets/audio")
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("validation fixture"), 0600); err != nil {
		s.Fatal(err)
	}
}
func (s *assetgoblinSuite) QOAResponse(samples int64) {
	s.Helper()
	data := s.pluginBody()
	if len(data) < 8 || string(data[:4]) != "qoaf" {
		s.Fatalf("invalid QOA response %q", data)
	}
	if binary.BigEndian.Uint32(data[4:8]) != uint32(samples) {
		s.Fatalf("wrong QOA sample count: %d", binary.BigEndian.Uint32(data[4:8]))
	}
}
func (s *assetgoblinSuite) NonemptyPluginResponse() {
	s.Helper()
	if len(s.pluginBody()) == 0 {
		s.Fatal("empty plugin response")
	}
}

type mediaMetadata struct {
	Streams []struct {
		Codec         string `json:"codec_name"`
		Width, Height int
	}
	Format struct{ Duration string }
}

func (s *assetgoblinSuite) MediaSource(kind string) {
	s.Helper()
	s.requireTools("ffmpeg", "ffprobe")
	if kind == "audio" {
		s.WAVSource()
		return
	}
	dir := filepath.Join(s.testDir, "assets/video")
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "color=c=red:s=320x240:d=1", "-c:v", "libx264", "-threads", "1", "-pix_fmt", "yuv420p", filepath.Join(dir, "source.mp4")).CombinedOutput()
	if err != nil {
		s.Fatalf("video fixture: %v %s", err, output)
	}
}
func (s *assetgoblinSuite) mediaMetadata() mediaMetadata {
	s.Helper()
	s.requireTools("ffprobe")
	file, err := os.CreateTemp(s.testDir, "media-response-*")
	if err != nil {
		s.Fatal(err)
	}
	_, writeErr := file.Write(s.pluginBody())
	closeErr := file.Close()
	if writeErr != nil {
		s.Fatal(writeErr)
	}
	if closeErr != nil {
		s.Fatal(closeErr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "stream=codec_name,width,height:format=duration", "-of", "json", file.Name()).Output()
	if err != nil {
		s.Fatal(err)
	}
	var metadata mediaMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		s.Fatal(err)
	}
	return metadata
}
func (s *assetgoblinSuite) MediaCodecAndDuration(codec string) {
	s.Helper()
	metadata := s.mediaMetadata()
	if len(metadata.Streams) != 1 || metadata.Streams[0].Codec != codec {
		s.Fatalf("expected codec %q, got %+v", codec, metadata.Streams)
	}
	duration, err := strconv.ParseFloat(metadata.Format.Duration, 64)
	if err != nil || duration < 0.4 || duration > 0.7 {
		s.Fatalf("expected half-second clip, got %q", metadata.Format.Duration)
	}
}
func (s *assetgoblinSuite) VideoDimensions(width, height int64) {
	s.Helper()
	metadata := s.mediaMetadata()
	if len(metadata.Streams) != 1 || metadata.Streams[0].Width != int(width) || metadata.Streams[0].Height != int(height) {
		s.Fatalf("expected %dx%d video, got %+v", width, height, metadata.Streams)
	}
}
func (s *assetgoblinSuite) VideoSource(name string) {
	s.Helper()
	dir := filepath.Join(s.testDir, "assets/video")
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("validation fixture"), 0600); err != nil {
		s.Fatal(err)
	}
}
func (s *assetgoblinSuite) SignedPluginRequest(path string) {
	hasher := hmac.New(sha256.New, []byte(s.config.Secret))
	hasher.Write([]byte(path))
	s.pluginRequest("GET", path+"?token="+hex.EncodeToString(hasher.Sum(nil)), "")
}

func (s *assetgoblinSuite) ResponseBodyLength(length int64) {
	s.Helper()
	if len(s.pluginBody()) != int(length) {
		s.Fatalf("expected %d response bytes", length)
	}
}

func (s *assetgoblinSuite) ImageRoutePrefix(prefix string) { s.config.Image.Path = prefix }
func (s *assetgoblinSuite) ConfigureFeatureServer()        { _, s.serverBuildErr = s.featureServer() }
func (s *assetgoblinSuite) ServerConfigurationFails(message string) {
	s.Helper()
	if s.serverBuildErr == nil || !strings.Contains(s.serverBuildErr.Error(), message) {
		s.Fatalf("expected server error containing %q, got %v", message, s.serverBuildErr)
	}
}

func (s *assetgoblinSuite) CoreImageConverter() {
	tool := "convert"
	if runtime.GOOS == "windows" {
		tool = "magick"
	}
	s.requireTools(tool)
}
func (s *assetgoblinSuite) ImageCached() {
	s.Helper()
	found := false
	err := filepath.WalkDir(s.config.Image.CacheDir, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			found = true
		}
		return err
	})
	if err != nil || !found {
		s.Fatalf("missing transformed image cache: %v", err)
	}
}
func (s *assetgoblinSuite) RequestMethod(method, path string) { s.pluginRequest(method, path, "") }
func (s *assetgoblinSuite) TokenFor(secret, path string) {
	hasher := hmac.New(sha256.New, []byte(secret))
	hasher.Write([]byte(path))
	s.token = hex.EncodeToString(hasher.Sum(nil))
}
