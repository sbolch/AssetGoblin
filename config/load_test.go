package config

import (
	"assetgoblin/utils"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// isolateConfigAndCacheEnv sets per-test config/cache environment directories.
func isolateConfigAndCacheEnv(t *testing.T) {
	t.Helper()

	tempHome := t.TempDir()
	tempConfig := t.TempDir()
	tempCache := t.TempDir()
	tempProgramData := t.TempDir()
	tempAppData := t.TempDir()
	tempLocalAppData := t.TempDir()

	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", tempConfig)
	t.Setenv("XDG_CACHE_HOME", tempCache)
	t.Setenv("ProgramData", tempProgramData)
	t.Setenv("APPDATA", tempAppData)
	t.Setenv("LOCALAPPDATA", tempLocalAppData)
}

// TestConfig_Load verifies config loading from defaults and gob cache.
func TestConfig_Load(t *testing.T) {
	isolateConfigAndCacheEnv(t)

	_ = RemoveGobFile()
	defer RemoveGobFile()

	tests := []struct {
		name         string
		setupFunc    func() error
		wantErr      bool
		validateFunc func(t *testing.T, cfg *Config)
		cleanupFunc  func() error
	}{
		{
			name: "default config when no config file exists",
			setupFunc: func() error {
				// Remove config file if it exists
				return os.RemoveAll("config.yaml")
			},
			wantErr: false,
			validateFunc: func(t *testing.T, cfg *Config) {
				if cfg.Port != "8080" {
					t.Errorf("Expected default port 8080, got %s", cfg.Port)
				}
				if cfg.PublicDir != "public" {
					t.Errorf("Expected default public_dir 'public', got %s", cfg.PublicDir)
				}
				if cfg.Secret != "" {
					t.Errorf("Expected default secret '', got %s", cfg.Secret)
				}
				if cfg.RateLimit.Limit != 0 {
					t.Errorf("Expected default rate limit 0, got %d", cfg.RateLimit.Limit)
				}
				if cfg.RateLimit.Ttl != time.Minute {
					t.Errorf("Expected default rate limit TTL 1m, got %v", cfg.RateLimit.Ttl)
				}
				if len(cfg.Image.Formats) != 6 {
					t.Errorf("Expected 6 default image formats, got %d", len(cfg.Image.Formats))
				}
				if len(cfg.Image.Presets) != 0 {
					t.Errorf("Expected 0 default image presets, got %d", len(cfg.Image.Presets))
				}
				if cfg.Image.Path != "/img/" {
					t.Errorf("Expected default image path '/img/', got %s", cfg.Image.Path)
				}
				if cfg.Image.Directory != "assets/img" {
					t.Errorf("Expected default image directory 'assets/img', got %s", cfg.Image.Directory)
				}
				if cfg.Image.CacheDir != filepath.Join(defaultCacheDir(), "img") {
					t.Errorf("Expected default cache directory '%s', got %s", filepath.Join(defaultCacheDir(), "img"), cfg.Image.CacheDir)
				}
				if cfg.Image.AvifThroughVips != false {
					t.Errorf("Expected default avif_through_vips false, got %v", cfg.Image.AvifThroughVips)
				}
				if cfg.UsedConfigFile != "none (defaults only)" {
					t.Errorf("Expected used config file to be 'none (defaults only)', got %s", cfg.UsedConfigFile)
				}
				if cfg.LoadedFromGob {
					t.Errorf("Expected LoadedFromGob false, got true")
				}
			},
			cleanupFunc: func() error {
				return nil
			},
		},
		{
			name: "load from gob file",
			setupFunc: func() error {
				cfg := &Config{
					Port:      "9090",
					PublicDir: "custom-public",
					Secret:    "test-secret",
					RateLimit: RateLimit{
						Limit: 100,
						Ttl:   time.Second * 30,
					},
					Image: Image{
						Path:            "/custom-img/",
						Directory:       "custom-assets/img",
						CacheDir:        "custom-cache",
						AvifThroughVips: true,
						Formats:         []string{"jpeg", "png"},
						Presets:         map[string]utils.ImagePreset{"custom": {Width: 800, Fit: "contain"}},
					},
					UsedConfigFile: "/tmp/test-config.yaml",
				}
				return cfg.saveGob()
			},
			wantErr: false,
			validateFunc: func(t *testing.T, cfg *Config) {
				if cfg.Port != "9090" {
					t.Errorf("Expected port 9090, got %s", cfg.Port)
				}
				if cfg.PublicDir != "custom-public" {
					t.Errorf("Expected public_dir 'custom-public', got %s", cfg.PublicDir)
				}
				if cfg.Secret != "test-secret" {
					t.Errorf("Expected secret 'test-secret', got %s", cfg.Secret)
				}
				if cfg.RateLimit.Limit != 100 {
					t.Errorf("Expected rate limit 100, got %d", cfg.RateLimit.Limit)
				}
				if cfg.RateLimit.Ttl != time.Second*30 {
					t.Errorf("Expected rate limit TTL 30s, got %v", cfg.RateLimit.Ttl)
				}
				if len(cfg.Image.Formats) != 2 {
					t.Errorf("Expected 2 image formats, got %d", len(cfg.Image.Formats))
				}
				if len(cfg.Image.Presets) != 1 {
					t.Errorf("Expected 1 image preset, got %d", len(cfg.Image.Presets))
				}
				if cfg.Image.Path != "/custom-img/" {
					t.Errorf("Expected image path '/custom-img/', got %s", cfg.Image.Path)
				}
				if cfg.Image.Directory != "custom-assets/img" {
					t.Errorf("Expected image directory 'custom-assets/img', got %s", cfg.Image.Directory)
				}
				if cfg.Image.CacheDir != "custom-cache" {
					t.Errorf("Expected cache directory 'custom-cache', got %s", cfg.Image.CacheDir)
				}
				if !cfg.Image.AvifThroughVips {
					t.Errorf("Expected avif_through_vips true, got %v", cfg.Image.AvifThroughVips)
				}
				if cfg.UsedConfigFile != "/tmp/test-config.yaml" {
					t.Errorf("Expected used config file '/tmp/test-config.yaml', got %s", cfg.UsedConfigFile)
				}
				if !cfg.LoadedFromGob {
					t.Errorf("Expected LoadedFromGob true, got false")
				}
			},
			cleanupFunc: func() error {
				return RemoveGobFile()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				if err := tt.setupFunc(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			if tt.cleanupFunc != nil {
				defer func() {
					if err := tt.cleanupFunc(); err != nil {
						t.Fatalf("Cleanup failed: %v", err)
					}
				}()
			}

			var cfg Config
			err := cfg.Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, &cfg)
			}
		})
	}
}

// TestSetDefaults verifies baseline default values after loading configuration.
func TestSetDefaults(t *testing.T) {
	isolateConfigAndCacheEnv(t)

	setDefaults()

	var cfg Config
	if err := cfg.Load(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", cfg.Port)
	}
	if cfg.PublicDir != "public" {
		t.Errorf("Expected default public_dir 'public', got %s", cfg.PublicDir)
	}
	if cfg.Secret != "" {
		t.Errorf("Expected default secret '', got %s", cfg.Secret)
	}
	if cfg.RateLimit.Limit != 0 {
		t.Errorf("Expected default rate limit 0, got %d", cfg.RateLimit.Limit)
	}
	if cfg.RateLimit.Ttl != time.Minute {
		t.Errorf("Expected default rate limit TTL 1m, got %v", cfg.RateLimit.Ttl)
	}
	if len(cfg.Image.Formats) != 6 {
		t.Errorf("Expected 6 default image formats, got %d", len(cfg.Image.Formats))
	}
	if len(cfg.Image.Presets) != 0 {
		t.Errorf("Expected 0 default image presets, got %d", len(cfg.Image.Presets))
	}
	if cfg.Image.Path != "/img/" {
		t.Errorf("Expected default image path '/img/', got %s", cfg.Image.Path)
	}
	if cfg.Image.Directory != "assets/img" {
		t.Errorf("Expected default image directory 'assets/img', got %s", cfg.Image.Directory)
	}
	if cfg.Image.CacheDir != filepath.Join(defaultCacheDir(), "img") {
		t.Errorf("Expected default cache directory '%s', got %s", filepath.Join(defaultCacheDir(), "img"), cfg.Image.CacheDir)
	}
	if cfg.Image.AvifThroughVips != false {
		t.Errorf("Expected default avif_through_vips false, got %v", cfg.Image.AvifThroughVips)
	}
}

// TestConfigLoad_SetsUsedConfigFileFromDisk ensures disk-backed config metadata is set.
func TestConfigLoad_SetsUsedConfigFileFromDisk(t *testing.T) {
	isolateConfigAndCacheEnv(t)
	_ = RemoveGobFile()
	defer RemoveGobFile()

	workdir := t.TempDir()
	t.Chdir(workdir)

	configContent := []byte("port: \"9091\"\npublic_dir: \"from-file\"\n")
	if err := os.WriteFile(filepath.Join(workdir, "config.yaml"), configContent, 0644); err != nil {
		t.Fatalf("Failed to write config.yaml: %v", err)
	}

	var cfg Config
	if err := cfg.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Port != "9091" {
		t.Fatalf("Expected port 9091, got %s", cfg.Port)
	}
	if cfg.UsedConfigFile == "" || cfg.UsedConfigFile == "none (defaults only)" {
		t.Fatalf("Expected UsedConfigFile to point to config file, got %q", cfg.UsedConfigFile)
	}
	if cfg.LoadedFromGob {
		t.Fatalf("Expected LoadedFromGob false when loading from config file")
	}
}
