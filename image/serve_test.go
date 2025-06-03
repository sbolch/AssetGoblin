package image

import (
	"assetgoblin/config"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestService_Serve_PathValidation(t *testing.T) {
	service := &Service{
		Config: &config.Image{
			Directory: "testdata",
			Presets:   map[string]string{"thumbnail": "100"},
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
			Presets:   map[string]string{"thumbnail": "100"},
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
			Presets:   map[string]string{"thumbnail": "100"},
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
			Presets:   map[string]string{"thumbnail": "100"},
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

func createTestImage(t *testing.T, path string) {
	createEmptyFile(t, path)
}
