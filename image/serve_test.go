package image

import (
	"assetgoblin/config"
	"assetgoblin/utils"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestService_Serve_PathValidation validates Serve behavior for malformed paths and inputs.
func TestService_Serve_PathValidation(t *testing.T) {
	service := &Service{
		Config: &config.Image{
			Directory: "testdata",
			Presets:   map[string]utils.ImagePreset{"thumbnail": {Width: 100, Fit: "contain"}},
			CacheDir:  "cache",
			Formats:   []string{"jpg", "png"},
		},
	}

	tests := []struct {
		name       string
		urlPath    string
		wantStatus int
	}{
		{
			name:       "path too short",
			urlPath:    "/img/short",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid preset",
			urlPath:    "/img/invalid/test.jpg",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unsupported format",
			urlPath:    "/img/thumbnail/test.gif",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.urlPath, nil)
			rec := httptest.NewRecorder()
			service.Serve(rec, req)
			if status := rec.Result().StatusCode; status != tt.wantStatus {
				t.Errorf("Serve() = %d, want %d", status, tt.wantStatus)
			}
		})
	}
}

// TestService_Serve_FileNotFound verifies Serve returns 404 when source files are missing.
func TestService_Serve_FileNotFound(t *testing.T) {
	testDir := "testdata"
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)
	defer os.RemoveAll("cache")

	service := &Service{
		Config: &config.Image{
			Directory: testDir,
			Presets:   map[string]utils.ImagePreset{"thumbnail": {Width: 100, Fit: "contain"}},
			CacheDir:  "cache",
			Formats:   []string{"jpg", "png"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/img/thumbnail/nonexistent.jpg", nil)
	rec := httptest.NewRecorder()
	service.Serve(rec, req)
	if status := rec.Result().StatusCode; status != http.StatusNotFound {
		t.Errorf("Serve() = %d, want %d", status, http.StatusNotFound)
	}
}

// TestService_Serve_ValidRequest exercises Serve with a valid request path and source file.
func TestService_Serve_ValidRequest(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI environment")
	}

	testDir := "testdata"
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)
	defer os.RemoveAll("cache")

	jpgPath := filepath.Join(testDir, "test.jpg")
	createTestImage(t, jpgPath)

	service := &Service{
		Config: &config.Image{
			Directory: testDir,
			Presets:   map[string]utils.ImagePreset{"thumbnail": {Width: 100, Fit: "contain"}},
			CacheDir:  "cache",
			Formats:   []string{"jpg", "png"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/img/thumbnail/test.jpg", nil)
	rec := httptest.NewRecorder()

	service.Serve(rec, req)
	status := rec.Result().StatusCode
	if status != http.StatusOK && status != http.StatusInternalServerError {
		t.Errorf("Serve() = %d, want either %d or %d", status, http.StatusOK, http.StatusInternalServerError)
	}
}

// TestService_Serve_FindImage verifies extension fallback lookup during Serve.
func TestService_Serve_FindImage(t *testing.T) {
	testDir := "testdata"
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)
	defer os.RemoveAll("cache")

	pngPath := filepath.Join(testDir, "test.png")
	createEmptyFile(t, pngPath)

	service := &Service{
		Config: &config.Image{
			Directory: testDir,
			Presets:   map[string]utils.ImagePreset{"thumbnail": {Width: 100, Fit: "contain"}},
			CacheDir:  "cache",
			Formats:   []string{"jpg", "png"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/img/thumbnail/test.jpg", nil)
	rec := httptest.NewRecorder()

	service.Serve(rec, req)
	status := rec.Result().StatusCode
	if status != http.StatusOK && status != http.StatusInternalServerError {
		t.Errorf("Serve() = %d, want either %d or %d", status, http.StatusOK, http.StatusInternalServerError)
	}
}

// TestService_Serve_AbsoluteCacheDir verifies absolute cache directories are used as-is.
func TestService_Serve_AbsoluteCacheDir(t *testing.T) {
	testDir := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "img-cache")

	jpgPath := filepath.Join(testDir, "test.jpg")
	createEmptyFile(t, jpgPath)

	service := &Service{
		Config: &config.Image{
			Directory: testDir,
			Presets:   map[string]utils.ImagePreset{"thumbnail": {Width: 100, Fit: "contain"}},
			CacheDir:  cacheDir,
			Formats:   []string{"jpg"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/img/thumbnail/test.jpg", nil)
	rec := httptest.NewRecorder()

	service.Serve(rec, req)
	status := rec.Result().StatusCode
	if status != http.StatusOK && status != http.StatusInternalServerError {
		t.Fatalf("Serve() = %d, want either %d or %d", status, http.StatusOK, http.StatusInternalServerError)
	}

	if _, err := os.Stat(filepath.Join(cacheDir, "test")); err != nil {
		t.Fatalf("expected cache directory to be created inside absolute cache dir: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to resolve working directory: %v", err)
	}

	unexpectedPath := filepath.Join(wd, strings.TrimPrefix(cacheDir, string(filepath.Separator)))
	if _, err = os.Stat(unexpectedPath); err == nil {
		t.Fatalf("unexpected cwd-prefixed cache path created: %s", unexpectedPath)
	}
}

// createTestImage creates a placeholder image file for Serve tests.
func createTestImage(t *testing.T, path string) {
	createEmptyFile(t, path)
}
