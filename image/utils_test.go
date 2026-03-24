package image

import (
	"assetgoblin/config"
	"assetgoblin/utils"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestService_FindImage verifies extension-based discovery of source images.
func TestService_FindImage(t *testing.T) {
	testDir := "test_find_image"
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	jpgPath := filepath.Join(testDir, "test.jpg")
	pngPath := filepath.Join(testDir, "test.png")
	createEmptyFile(t, jpgPath)
	createEmptyFile(t, pngPath)

	tests := []struct {
		name      string
		basePath  string
		formats   []string
		want      string
		wantFound bool
	}{
		{
			name:      "find jpg image",
			basePath:  filepath.Join(testDir, "test"),
			formats:   []string{"jpg", "png"},
			want:      jpgPath,
			wantFound: true,
		},
		{
			name:      "find png image when jpg is not in formats",
			basePath:  filepath.Join(testDir, "test"),
			formats:   []string{"png"},
			want:      pngPath,
			wantFound: true,
		},
		{
			name:      "image not found",
			basePath:  filepath.Join(testDir, "nonexistent"),
			formats:   []string{"jpg", "png"},
			want:      "",
			wantFound: false,
		},
		{
			name:      "empty formats list",
			basePath:  filepath.Join(testDir, "test"),
			formats:   []string{},
			want:      "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				Config: &config.Image{
					Formats: tt.formats,
				},
			}
			got, found := s.findImage(tt.basePath)
			if found != tt.wantFound {
				t.Errorf("findImage() found = %v, want %v", found, tt.wantFound)
			}
			if found && got != tt.want {
				t.Errorf("findImage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestService_IsValidFormat verifies format validation against configured extensions.
func TestService_IsValidFormat(t *testing.T) {
	tests := []struct {
		name    string
		formats []string
		format  string
		want    bool
	}{
		{
			name:    "valid format",
			formats: []string{"jpg", "png", "webp"},
			format:  ".jpg",
			want:    true,
		},
		{
			name:    "invalid format",
			formats: []string{"jpg", "png", "webp"},
			format:  ".gif",
			want:    false,
		},
		{
			name:    "format without dot",
			formats: []string{"jpg", "png", "webp"},
			format:  "jpg",
			want:    true,
		},
		{
			name:    "empty format",
			formats: []string{"jpg", "png", "webp"},
			format:  "",
			want:    false,
		},
		{
			name:    "empty formats list",
			formats: []string{},
			format:  ".jpg",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				Config: &config.Image{
					Formats: tt.formats,
				},
			}
			if got := s.isValidFormat(tt.format); got != tt.want {
				t.Errorf("isValidFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

// createEmptyFile creates an empty file used as a test fixture.
func createEmptyFile(t *testing.T, path string) {
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close test file %s: %v", path, err)
	}
}

// TestParseSize verifies parseSize function with various size inputs.
func TestParseSize(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		fitParam       string
		wantOutput     string
		wantHasSize    bool
		wantFitContain bool
	}{
		{
			name:           "width only",
			input:          "640",
			fitParam:       "",
			wantOutput:     "640",
			wantHasSize:    false,
			wantFitContain: false,
		},
		{
			name:           "width x height default fit",
			input:          "640x480",
			fitParam:       "",
			wantOutput:     "640x480",
			wantHasSize:    true,
			wantFitContain: true,
		},
		{
			name:           "width x height cover fit",
			input:          "640x480",
			fitParam:       "cover",
			wantOutput:     "640x480",
			wantHasSize:    true,
			wantFitContain: false,
		},
		{
			name:           "invalid format with too many x",
			input:          "640x480x",
			fitParam:       "",
			wantOutput:     "",
			wantHasSize:    false,
			wantFitContain: false,
		},
		{
			name:           "invalid non-numeric width",
			input:          "abc",
			fitParam:       "",
			wantOutput:     "",
			wantHasSize:    false,
			wantFitContain: false,
		},
		{
			name:           "invalid non-numeric height",
			input:          "640xabc",
			fitParam:       "",
			wantOutput:     "",
			wantHasSize:    false,
			wantFitContain: false,
		},
		{
			name:           "zero width",
			input:          "0",
			fitParam:       "",
			wantOutput:     "",
			wantHasSize:    false,
			wantFitContain: false,
		},
		{
			name:           "negative width",
			input:          "-100",
			fitParam:       "",
			wantOutput:     "",
			wantHasSize:    false,
			wantFitContain: false,
		},
		{
			name:           "zero height",
			input:          "640x0",
			fitParam:       "",
			wantOutput:     "",
			wantHasSize:    false,
			wantFitContain: false,
		},
		{
			name:           "empty string",
			input:          "",
			fitParam:       "",
			wantOutput:     "",
			wantHasSize:    false,
			wantFitContain: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, parts, _ := parseSize(tt.input, tt.fitParam, nil)
			if result != tt.wantOutput {
				t.Errorf("parseSize(%q, %q) = %q, want %q", tt.input, tt.fitParam, result, tt.wantOutput)
			}
			if parts.hasSize != tt.wantHasSize {
				t.Errorf("parseSize(%q, %q).hasSize = %v, want %v", tt.input, tt.fitParam, parts.hasSize, tt.wantHasSize)
			}
		})
	}
}

// TestParseSizeWithPresets verifies parseSize with preset configurations.
func TestParseSizeWithPresets(t *testing.T) {
	presets := map[string]utils.ImagePreset{
		"thumbnail": {Width: 200, Height: 200, Fit: "cover"},
		"lg":        {Width: 960, Height: 0, Fit: "contain"},
		"rotated":   {Width: 640, Rotate: 90},
		"flipped":   {Width: 640, Flip: "horizontal"},
		"bright":    {Width: 640, Brightness: 50},
		"contrast":  {Width: 640, Contrast: -25},
		"gamm":      {Width: 640, Gamma: 1.5},
		"filter":    {Width: 640, Filters: []string{"grayscale", "blur"}},
	}

	tests := []struct {
		name         string
		input        string
		fitParam     string
		wantPreset   bool
		wantWidth    int
		wantHeight   int
		wantRotate   int
		wantFlip     string
		wantBright   float64
		wantContrast float64
		wantGamma    float64
		wantFilters  int
	}{
		{
			name:       "preset with size",
			input:      "thumbnail",
			fitParam:   "",
			wantPreset: true,
			wantWidth:  200,
			wantHeight: 200,
		},
		{
			name:       "preset with width only",
			input:      "lg",
			fitParam:   "",
			wantPreset: true,
			wantWidth:  960,
			wantHeight: 0,
		},
		{
			name:       "preset with rotate",
			input:      "rotated",
			fitParam:   "",
			wantPreset: true,
			wantWidth:  640,
			wantRotate: 90,
		},
		{
			name:       "preset with flip",
			input:      "flipped",
			fitParam:   "",
			wantPreset: true,
			wantWidth:  640,
			wantFlip:   "horizontal",
		},
		{
			name:       "preset with brightness",
			input:      "bright",
			fitParam:   "",
			wantPreset: true,
			wantWidth:  640,
			wantBright: 50,
		},
		{
			name:         "preset with contrast",
			input:        "contrast",
			fitParam:     "",
			wantPreset:   true,
			wantWidth:    640,
			wantContrast: -25,
		},
		{
			name:       "preset with gamma",
			input:      "gamm",
			fitParam:   "",
			wantPreset: true,
			wantWidth:  640,
			wantGamma:  1.5,
		},
		{
			name:        "preset with filters",
			input:       "filter",
			fitParam:    "",
			wantPreset:  true,
			wantWidth:   640,
			wantFilters: 2,
		},
		{
			name:       "preset override with query fit",
			input:      "thumbnail",
			fitParam:   "cover",
			wantPreset: true,
			wantWidth:  200,
			wantHeight: 200,
		},
		{
			name:       "direct size not found in presets",
			input:      "800",
			fitParam:   "",
			wantPreset: false,
			wantWidth:  800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, parts, isPreset := parseSize(tt.input, tt.fitParam, presets)
			if isPreset != tt.wantPreset {
				t.Errorf("parseSize() isPreset = %v, want %v", isPreset, tt.wantPreset)
			}
			if parts.width != tt.wantWidth {
				t.Errorf("parseSize() width = %v, want %v", parts.width, tt.wantWidth)
			}
			if parts.height != tt.wantHeight {
				t.Errorf("parseSize() height = %v, want %v", parts.height, tt.wantHeight)
			}
			if parts.rotate != tt.wantRotate {
				t.Errorf("parseSize() rotate = %v, want %v", parts.rotate, tt.wantRotate)
			}
			if parts.flip != tt.wantFlip {
				t.Errorf("parseSize() flip = %v, want %v", parts.flip, tt.wantFlip)
			}
			if parts.brightness != tt.wantBright {
				t.Errorf("parseSize() brightness = %v, want %v", parts.brightness, tt.wantBright)
			}
			if parts.contrast != tt.wantContrast {
				t.Errorf("parseSize() contrast = %v, want %v", parts.contrast, tt.wantContrast)
			}
			if parts.gamma != tt.wantGamma {
				t.Errorf("parseSize() gamma = %v, want %v", parts.gamma, tt.wantGamma)
			}
			if len(parts.filters) != tt.wantFilters {
				t.Errorf("parseSize() filters = %v, want %v", len(parts.filters), tt.wantFilters)
			}
			if tt.wantPreset && result == "" {
				t.Errorf("parseSize() returned empty result for preset %q", tt.input)
			}
		})
	}
}

// TestEnsureAbsolute verifies path resolution for relative and absolute paths.
func TestEnsureAbsolute(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wd      string
		wantAbs bool
	}{
		{
			name:    "absolute path unchanged",
			path:    "/absolute/path",
			wd:      "/working",
			wantAbs: true,
		},
		{
			name:    "relative path becomes absolute",
			path:    "relative/path",
			wd:      "/working",
			wantAbs: true,
		},
		{
			name:    "dot relative path becomes absolute",
			path:    "./relative/path",
			wd:      "/working",
			wantAbs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureAbsolute(tt.path, tt.wd)
			if tt.wantAbs && !filepath.IsAbs(result) {
				t.Errorf("ensureAbsolute() = %q, want absolute path", result)
			}
			if !tt.wantAbs && filepath.IsAbs(result) {
				t.Errorf("ensureAbsolute() = %q, want relative path", result)
			}
			if tt.wantAbs && result != tt.path {
				if tt.wantAbs && !strings.HasPrefix(result, tt.wd) {
					t.Errorf("ensureAbsolute() = %q, want path starting with %q", result, tt.wd)
				}
			}
		})
	}
}
