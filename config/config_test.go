package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestConfig_loadJson(t *testing.T) {
	tests := []struct {
		name           string
		jsonContent    string
		expectedConfig jsonConfig
		expectedError  error
		setup          func(jsonContent string)
		teardown       func()
	}{
		{
			name: "valid config",
			jsonContent: `{
			  "port": "8080",
			  "public_dir": "public",
			  "secret": "",
			  "rate_limit": {
				"limit": 0,
				"ttl": "1m"
			  },
			  "image": {
				"formats": ["avif", "jpeg", "jpg", "png", "tiff", "webp"],
				"presets": {
				  "lg": "960",
				  "lg2x": "1920",
				  "sm": "640",
				  "sm2x": "1280"
				},
				"path": "/img/",
				"directory": "assets/img",
				"cache_dir": "cache",
				"avif_through_vips": false
			  }
			}`,
			expectedConfig: jsonConfig{
				Image: struct {
					AvifThroughVips bool              `json:"avif_through_vips"`
					CacheDir        string            `json:"cache_dir"`
					Directory       string            `json:"directory"`
					Formats         []string          `json:"formats"`
					Path            string            `json:"path"`
					Presets         map[string]string `json:"presets"`
				}{
					AvifThroughVips: false,
					CacheDir:        "cache",
					Directory:       "assets/img",
					Formats:         []string{"avif", "jpeg", "jpg", "png", "tiff", "webp"},
					Path:            "/img/",
					Presets: map[string]string{
						"lg":   "960",
						"lg2x": "1920",
						"sm":   "640",
						"sm2x": "1280",
					},
				},
				Port:      "8080",
				PublicDir: "public",
				RateLimit: struct {
					Limit int      `json:"limit"`
					Ttl   Duration `json:"ttl"`
				}{
					Limit: 0,
					Ttl:   Duration{1 * time.Minute},
				},
				Secret: "",
			},
			setup: func(jsonContent string) {
				_ = os.WriteFile("config.json", []byte(jsonContent), 0644)
			},
			teardown: func() {
				_ = os.Remove("config.json")
			},
		},
		{
			name:          "file not found",
			jsonContent:   "",
			expectedError: errors.New("unable to read config file: open config.json"),
			setup: func(jsonContent string) {
			},
			teardown: func() {
			},
		},
		{
			name:           "invalid json",
			jsonContent:    `{invalid json}`,
			expectedConfig: jsonConfig{},
			expectedError:  errors.New("unable to unmarshal config file: invalid character 'i' looking for beginning of object key string"),
			setup: func(jsonContent string) {
				_ = os.WriteFile("config.json", []byte(`{invalid json}`), 0644)
			},
			teardown: func() {
				_ = os.Remove("config.json")
			},
		},
	}

	for _, tt := range tests {
		tt.setup(tt.jsonContent)
		t.Run(tt.name, func(t *testing.T) {
			config, err := loadJson()
			if tt.expectedError == nil {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(tt.expectedConfig, config) {
					t.Errorf("expected config %v, got %v", tt.expectedConfig, config)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %v, but got none", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError.Error()) {
					t.Errorf("expected error to contain %q, but got %q", tt.expectedError.Error(), err.Error())
				}
			}
		})
		tt.teardown()
	}
}

func TestConfig_Load(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() error
		expectedErr  bool
		validateFunc func(*Config) error
	}{
		{
			name: "valid data",
			setup: func() error {
				cfg := &Config{
					Port:      "8080",
					PublicDir: "/var/www",
					Secret:    "supersecret",
					Image: &Image{
						AvifThroughVips: true,
						CacheDir:        "/cache",
						Directory:       "/images",
						Formats:         []string{"jpg", "png"},
						Path:            "/img",
						Presets:         map[string]string{"thumbnail": "100"},
					},
					RateLimit: &RateLimit{
						Limit: 100,
						Ttl:   time.Minute,
					},
				}
				err := cfg.saveGob()
				return err
			},
			expectedErr: false,
			validateFunc: func(cfg *Config) error {
				if cfg.Port != "8080" {
					return fmt.Errorf("expected Port to be 8080, got %s", cfg.Port)
				}
				return nil
			},
		},
		{
			name: "invalid data",
			setup: func() error {
				file, err := os.Create("config.gob")
				if err != nil {
					return err
				}
				defer file.Close()
				_, err = file.Write([]byte("invalid data"))
				return err
			},
			expectedErr: true,
			validateFunc: func(cfg *Config) error {
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.setup(); err != nil {
				t.Fatalf("failed to set up test: %v", err)
			}
			defer os.Remove("config.gob")

			cfg := &Config{}
			err := cfg.Load()
			if (err != nil) != tt.expectedErr {
				t.Errorf("expected error = %v, got %v", tt.expectedErr, err)
			}

			if tt.validateFunc != nil {
				if err := tt.validateFunc(cfg); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}
