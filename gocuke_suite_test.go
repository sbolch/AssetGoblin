package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	stdimage "image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"assetgoblin/config"
	"assetgoblin/image"
	"assetgoblin/middleware"
	"assetgoblin/plugin"
	"assetgoblin/utils"

	"github.com/regen-network/gocuke"
)

type assetgoblinSuite struct {
	liveURL           string
	liveStop          func()
	pluginManager     *plugin.Manager
	pluginLoadErr     error
	serverBuildErr    error
	pluginDir         string
	pluginResponse    []byte
	pluginAssetLength int
	gocuke.TestingT

	testDir   string
	config    *config.Config
	server    *httptest.Server
	lastResp  *http.Response
	responses []*http.Response

	imageService *image.Service
	signkey      middleware.Signkey
	ratelimit    *middleware.RateLimit
	token        string
	handler      http.Handler
}

func newSuite() *assetgoblinSuite {
	return &assetgoblinSuite{
		config: &config.Config{
			Port:      "8080",
			PublicDir: "testdata/public",
			Image: config.Image{
				Directory: "testdata/assets/img",
				CacheDir:  "testdata/cache/img",
				Formats:   []string{"jpg", "png"},
				Path:      "/img/",
				Presets:   map[string]utils.ImagePreset{},
			},
		},
	}
}

func (s *assetgoblinSuite) cleanup() {
	if s.liveStop != nil {
		s.liveStop()
		s.liveStop = nil
	}
	s.liveURL = ""
	if s.server != nil {
		s.server.Close()
		s.server = nil
	}
	if s.pluginManager != nil {
		s.pluginManager.Close()
		s.pluginManager = nil
	}
	s.pluginLoadErr = nil
	s.serverBuildErr = nil
	s.pluginResponse = nil
	s.pluginAssetLength = 0
	for _, r := range s.responses {
		if r != nil && r.Body != nil {
			r.Body.Close()
		}
	}
	if s.lastResp != nil && s.lastResp.Body != nil {
		s.lastResp.Body.Close()
	}
	s.lastResp = nil
	s.responses = nil
	if s.testDir != "" {
		os.RemoveAll(s.testDir)
		s.testDir = ""
	}
}

// --- Hooks ---

func (s *assetgoblinSuite) Before() {
	s.cleanup()
	s.pluginDir = ""
	var err error
	s.testDir, err = os.MkdirTemp("", "assetgoblin-features-")
	if err != nil {
		s.Fatal(err)
	}
	s.config = &config.Config{
		Port: "8080", PublicDir: filepath.Join(s.testDir, "public"),
		Image: config.Image{Directory: filepath.Join(s.testDir, "assets/img"), CacheDir: filepath.Join(s.testDir, "cache/img"), Formats: []string{"jpg", "png"}, Path: "/img/", Presets: map[string]utils.ImagePreset{}},
	}

}

func (s *assetgoblinSuite) After() {
	s.cleanup()
}

// --- Server steps ---

func (s *assetgoblinSuite) AConfiguredServerWithPublicDir(publicDir string) {
	s.config.PublicDir = filepath.Join(s.testDir, strings.TrimPrefix(publicDir, "testdata/"))
}

func (s *assetgoblinSuite) AStaticFileExistsWithContent(filePath, content string) {
	fullPath := filepath.Join(s.config.PublicDir, filePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		s.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		s.Fatalf("failed to write file: %v", err)
	}
}

func (s *assetgoblinSuite) AConfiguredServerWithImagePresetOfWidth(preset string, width int64) {
	s.config.Image.Presets[preset] = utils.ImagePreset{Width: int(width), Fit: "contain"}
}

func (s *assetgoblinSuite) AnImageFileExists(imagePath string) {
	s.writeImage(filepath.Join(s.testDir, strings.TrimPrefix(imagePath, "testdata/")))
}

func (s *assetgoblinSuite) AConfiguredServerWithSupportedImageFormatsAnd(format1, format2 string) {
	s.config.Image.Formats = []string{format1, format2}
}

func (s *assetgoblinSuite) AConfiguredServer() {}

func (s *assetgoblinSuite) ISendAGetRequestTo(path string) { s.pluginRequest(http.MethodGet, path, "") }

func (s *assetgoblinSuite) TheResponseStatusShouldBe(expected int64) {
	s.Helper()
	if s.lastResp == nil {
		s.Fatal("no response available")
	}
	if s.lastResp.StatusCode != int(expected) {
		s.Fatalf("expected status %d, got %d", expected, s.lastResp.StatusCode)
	}
}

func (s *assetgoblinSuite) TheResponseBodyShouldContain(expected string) {
	s.Helper()
	if s.lastResp == nil {
		s.Fatal("no response available")
	}
	body := s.pluginBody()
	if !strings.Contains(string(body), expected) {
		s.Fatalf("expected body to contain %q, got %q", expected, body)
	}
}

func (s *assetgoblinSuite) TheImageResponseShouldBeSuccessful() {
	s.Helper()
	_, vipsErr := exec.LookPath("vipsthumbnail")
	_, convertErr := exec.LookPath("convert")
	_, magickErr := exec.LookPath("magick")
	if vipsErr != nil && convertErr != nil && magickErr != nil {
		s.Skip("image converter not installed")
	}
	s.TheResponseStatusShouldBe(200)
}

func (s *assetgoblinSuite) writeImage(path string) {
	s.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		s.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		s.Fatal(err)
	}
	err = jpeg.Encode(file, stdimage.NewRGBA(stdimage.Rect(0, 0, 100, 100)), nil)
	closeErr := file.Close()
	if err != nil {
		s.Fatal(err)
	}
	if closeErr != nil {
		s.Fatal(closeErr)
	}
}

// --- Image processing steps ---

func (s *assetgoblinSuite) AnImageServiceConfiguredWithSupportedFormatsAnd(format1, format2 string) {
	s.config.Image.Formats = []string{format1, format2}
}

func (s *assetgoblinSuite) ASourceImageExistsInTheImageDirectory(imagePath string) {
	fullPath := filepath.Join(s.config.Image.Directory, imagePath)
	s.writeImage(fullPath)
}

func (s *assetgoblinSuite) AnImageServiceConfiguredWithPresetOfWidth(preset string, width int64) {
	s.config.Image.Presets[preset] = utils.ImagePreset{Width: int(width), Fit: "contain"}
	s.imageService = &image.Service{Config: &s.config.Image}
}

func (s *assetgoblinSuite) IRequestAnImageWithPath(path string) {
	s.imageService = &image.Service{
		Config: &config.Image{
			Directory: s.config.Image.Directory,
			CacheDir:  s.config.Image.CacheDir,
			Formats:   s.config.Image.Formats,
			Presets:   s.config.Image.Presets,
			Path:      s.config.Image.Path,
		},
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	s.imageService.Serve(rec, req)
	if s.lastResp != nil && s.lastResp.Body != nil {
		s.lastResp.Body.Close()
	}
	s.lastResp = rec.Result()
}

func (s *assetgoblinSuite) IRequest(path string) {
	if s.imageService == nil {
		s.imageService = &image.Service{
			Config: &config.Image{
				Directory: s.config.Image.Directory,
				CacheDir:  s.config.Image.CacheDir,
				Formats:   s.config.Image.Formats,
				Presets:   s.config.Image.Presets,
				Path:      s.config.Image.Path,
			},
		}
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	s.imageService.Serve(rec, req)
	if s.lastResp != nil && s.lastResp.Body != nil {
		s.lastResp.Body.Close()
	}
	s.lastResp = rec.Result()
}

// --- Middleware steps ---

func (s *assetgoblinSuite) ASignkeyMiddlewareWithSecret(secret string) {
	s.signkey = middleware.Signkey{Secret: secret}
}

func (s *assetgoblinSuite) AValidTokenIsGeneratedForPath(path string) {
	hasher := hmac.New(sha256.New, []byte(s.signkey.Secret))
	hasher.Write([]byte(path))
	s.token = hex.EncodeToString(hasher.Sum(nil))
}

func (s *assetgoblinSuite) ISendAGetRequestToWithThatToken(path string) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s.handler = s.signkey.Verify(nextHandler)

	req := httptest.NewRequest(http.MethodGet, path+"?token="+s.token, nil)
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	if s.lastResp != nil && s.lastResp.Body != nil {
		s.lastResp.Body.Close()
	}
	s.lastResp = rec.Result()
}

func (s *assetgoblinSuite) ISendAGetRequestToWithToken(path, token string) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s.handler = s.signkey.Verify(nextHandler)

	req := httptest.NewRequest(http.MethodGet, path+"?token="+token, nil)
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	if s.lastResp != nil && s.lastResp.Body != nil {
		s.lastResp.Body.Close()
	}
	s.lastResp = rec.Result()
}

func (s *assetgoblinSuite) ISendAGetRequestToWithoutAToken(path string) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s.handler = s.signkey.Verify(nextHandler)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	if s.lastResp != nil && s.lastResp.Body != nil {
		s.lastResp.Body.Close()
	}
	s.lastResp = rec.Result()
}

func (s *assetgoblinSuite) ARateLimitMiddlewareConfiguredForRequestsPerMinute(limit int64) {
	s.ratelimit = middleware.NewRateLimit(&config.RateLimit{
		Limit: int(limit),
		Ttl:   time.Minute,
	})
}

func (s *assetgoblinSuite) ISendRequestsWithinTheWindow(count int64) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s.handler = s.ratelimit.Limit(nextHandler)
	s.responses = nil

	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := int64(0); i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()
			s.handler.ServeHTTP(rec, req)
			mu.Lock()
			s.responses = append(s.responses, rec.Result())
			mu.Unlock()
		}()
	}
	wg.Wait()
}

func (s *assetgoblinSuite) AllRequestsShouldBeSuccessful() {
	for _, resp := range s.responses {
		if resp.StatusCode != http.StatusOK {
			s.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	}
}

func (s *assetgoblinSuite) ARateLimitMiddlewareConfiguredForRequestsWithAMsWindow(limit int64, windowMs int64) {
	s.ratelimit = middleware.NewRateLimit(&config.RateLimit{
		Limit: int(limit),
		Ttl:   time.Duration(windowMs) * time.Millisecond,
	})
}

func (s *assetgoblinSuite) ISendRequestsInRapidSuccession(count int64) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s.handler = s.ratelimit.Limit(nextHandler)
	s.responses = nil

	for i := int64(0); i < count; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		s.handler.ServeHTTP(rec, req)
		s.responses = append(s.responses, rec.Result())
	}
}

func (s *assetgoblinSuite) The4thRequestShouldBeRateLimited() {
	if len(s.responses) < 4 {
		s.Fatalf("expected at least 4 responses, got %d", len(s.responses))
	}
	if s.responses[3].StatusCode != http.StatusTooManyRequests {
		s.Fatalf("expected 429, got %d", s.responses[3].StatusCode)
	}
}

func (s *assetgoblinSuite) IWaitForTheWindowToExpire() {
	time.Sleep(150 * time.Millisecond)
}

func (s *assetgoblinSuite) ISendAnotherRequest() {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	s.responses = append(s.responses, rec.Result())
}

func (s *assetgoblinSuite) ThatRequestShouldBeSuccessful() {
	if len(s.responses) == 0 {
		s.Fatal("no responses recorded")
	}
	last := s.responses[len(s.responses)-1]
	if last.StatusCode != http.StatusOK {
		s.Fatalf("expected 200, got %d", last.StatusCode)
	}
}
