package config

import (
	"os"
	"testing"
)

func TestConfig_SaveGob(t *testing.T) {
	_ = os.Remove("config.gob")
	defer os.Remove("config.gob")

	cfg := &Config{
		Port:      "9090",
		PublicDir: "test-public",
		Secret:    "test-secret",
	}

	err := cfg.saveGob()
	if err != nil {
		t.Fatalf("saveGob() error = %v", err)
	}

	if _, err := os.Stat("config.gob"); os.IsNotExist(err) {
		t.Errorf("saveGob() did not create config.gob file")
	}
}

func TestConfig_LoadGob(t *testing.T) {
	_ = os.Remove("config.gob")
	defer os.Remove("config.gob")

	tests := []struct {
		name     string
		setupCfg *Config
		wantErr  bool
	}{
		{
			name: "successful load",
			setupCfg: &Config{
				Port:      "9090",
				PublicDir: "test-public",
				Secret:    "test-secret",
			},
			wantErr: false,
		},
		{
			name:     "file does not exist",
			setupCfg: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupCfg != nil {
				if err := tt.setupCfg.saveGob(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			} else {
				_ = os.Remove("config.gob")
			}

			var loadedCfg Config
			err := loadedCfg.loadGob()
			if (err != nil) != tt.wantErr {
				t.Errorf("loadGob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.setupCfg != nil {
				if loadedCfg.Port != tt.setupCfg.Port {
					t.Errorf("loadGob() loaded Port = %v, want %v", loadedCfg.Port, tt.setupCfg.Port)
				}
				if loadedCfg.PublicDir != tt.setupCfg.PublicDir {
					t.Errorf("loadGob() loaded PublicDir = %v, want %v", loadedCfg.PublicDir, tt.setupCfg.PublicDir)
				}
				if loadedCfg.Secret != tt.setupCfg.Secret {
					t.Errorf("loadGob() loaded Secret = %v, want %v", loadedCfg.Secret, tt.setupCfg.Secret)
				}
			}
		})
	}
}

func TestCloseFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-close-file-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	closeFile(tmpFile)

	_, err = tmpFile.Write([]byte("test"))
	if err == nil {
		t.Errorf("Expected error writing to closed file, got nil")
	}
}

func TestConfig_SaveLoadGob_Integration(t *testing.T) {
	_ = os.Remove("config.gob")
	defer os.Remove("config.gob")

	originalCfg := &Config{
		Port:      "9090",
		PublicDir: "test-public",
		Secret:    "test-secret",
		RateLimit: RateLimit{
			Limit: 100,
			Ttl:   60000000000, // 1 minute in nanoseconds
		},
		Image: Image{
			AvifThroughVips: true,
			CacheDir:        "test-cache",
			Directory:       "test-dir",
			Formats:         []string{"jpg", "png", "webp"},
			Path:            "/test-path/",
			Presets: map[string]string{
				"small":  "100",
				"medium": "300",
				"large":  "600",
			},
		},
	}

	err := originalCfg.saveGob()
	if err != nil {
		t.Fatalf("saveGob() error = %v", err)
	}

	var loadedCfg Config
	err = loadedCfg.loadGob()
	if err != nil {
		t.Fatalf("loadGob() error = %v", err)
	}

	if loadedCfg.Port != originalCfg.Port {
		t.Errorf("Port = %v, want %v", loadedCfg.Port, originalCfg.Port)
	}
	if loadedCfg.PublicDir != originalCfg.PublicDir {
		t.Errorf("PublicDir = %v, want %v", loadedCfg.PublicDir, originalCfg.PublicDir)
	}
	if loadedCfg.Secret != originalCfg.Secret {
		t.Errorf("Secret = %v, want %v", loadedCfg.Secret, originalCfg.Secret)
	}

	if loadedCfg.RateLimit.Limit != originalCfg.RateLimit.Limit {
		t.Errorf("RateLimit.Limit = %v, want %v", loadedCfg.RateLimit.Limit, originalCfg.RateLimit.Limit)
	}
	if loadedCfg.RateLimit.Ttl != originalCfg.RateLimit.Ttl {
		t.Errorf("RateLimit.Ttl = %v, want %v", loadedCfg.RateLimit.Ttl, originalCfg.RateLimit.Ttl)
	}

	if loadedCfg.Image.AvifThroughVips != originalCfg.Image.AvifThroughVips {
		t.Errorf("Image.AvifThroughVips = %v, want %v", loadedCfg.Image.AvifThroughVips, originalCfg.Image.AvifThroughVips)
	}
	if loadedCfg.Image.CacheDir != originalCfg.Image.CacheDir {
		t.Errorf("Image.CacheDir = %v, want %v", loadedCfg.Image.CacheDir, originalCfg.Image.CacheDir)
	}
	if loadedCfg.Image.Directory != originalCfg.Image.Directory {
		t.Errorf("Image.Directory = %v, want %v", loadedCfg.Image.Directory, originalCfg.Image.Directory)
	}
	if loadedCfg.Image.Path != originalCfg.Image.Path {
		t.Errorf("Image.Path = %v, want %v", loadedCfg.Image.Path, originalCfg.Image.Path)
	}

	if len(loadedCfg.Image.Formats) != len(originalCfg.Image.Formats) {
		t.Errorf("Image.Formats length = %v, want %v", len(loadedCfg.Image.Formats), len(originalCfg.Image.Formats))
	} else {
		for i, format := range originalCfg.Image.Formats {
			if loadedCfg.Image.Formats[i] != format {
				t.Errorf("Image.Formats[%d] = %v, want %v", i, loadedCfg.Image.Formats[i], format)
			}
		}
	}

	if len(loadedCfg.Image.Presets) != len(originalCfg.Image.Presets) {
		t.Errorf("Image.Presets length = %v, want %v", len(loadedCfg.Image.Presets), len(originalCfg.Image.Presets))
	} else {
		for key, value := range originalCfg.Image.Presets {
			if loadedCfg.Image.Presets[key] != value {
				t.Errorf("Image.Presets[%s] = %v, want %v", key, loadedCfg.Image.Presets[key], value)
			}
		}
	}
}
