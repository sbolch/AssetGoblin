package image

import (
	"assetgoblin/config"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestImage_Serve(t *testing.T) {
	os.MkdirAll("testdata", 0755)
	defer os.RemoveAll("testdata")
	defer os.RemoveAll("cache")

	tests := []struct {
		name       string
		urlPath    string
		setup      func() *Service
		wantStatus int
	}{
		{
			name:    "valid request",
			urlPath: "/images/thumbnail/sample.jpg",
			setup: func() *Service {
				createJpg("testdata/sample.jpg")
				return &Service{
					Config: &config.Image{
						Directory: "testdata",
						Presets:   map[string]string{"thumbnail": "100"},
						CacheDir:  "cache",
						Formats:   []string{"jpg", "png"},
					},
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "missing preset",
			urlPath: "/images/invalidpreset/sample.jpg",
			setup: func() *Service {
				return &Service{
					Config: &config.Image{
						Directory: "testdata",
						Presets:   map[string]string{"thumbnail": "100"},
						CacheDir:  "cache",
						Formats:   []string{"jpg", "png"},
					},
				}
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "invalid format",
			urlPath: "/images/thumbnail/sample.txt",
			setup: func() *Service {
				return &Service{
					Config: &config.Image{
						Directory: "testdata",
						Presets:   map[string]string{"thumbnail": "100"},
						CacheDir:  "cache",
						Formats:   []string{"jpg", "png"},
					},
				}
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "file not found",
			urlPath: "/images/thumbnail/notfound.jpg",
			setup: func() *Service {
				return &Service{
					Config: &config.Image{
						Directory: "testdata",
						Presets:   map[string]string{"thumbnail": "100"},
						CacheDir:  "cache",
						Formats:   []string{"jpg", "png"},
					},
				}
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:    "short URL path",
			urlPath: "/images/short",
			setup: func() *Service {
				return &Service{
					Config: &config.Image{
						Directory: "testdata",
						Presets:   map[string]string{"thumbnail": "100"},
						CacheDir:  "cache",
						Formats:   []string{"jpg", "png"},
					},
				}
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := tt.setup()
			req := httptest.NewRequest(http.MethodGet, tt.urlPath, nil)
			rec := httptest.NewRecorder()
			service.Serve(rec, req)
			if status := rec.Result().StatusCode; status != tt.wantStatus {
				t.Errorf("Serve() = %d, want %d", status, tt.wantStatus)
			}
		})
	}
}

func createJpg(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			img.Set(x, y, image.White)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			return
		}
	}(file)

	return jpeg.Encode(file, img, nil)
}
