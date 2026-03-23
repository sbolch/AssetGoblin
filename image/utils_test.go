package image

import (
	"assetgoblin/config"
	"os"
	"path/filepath"
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
