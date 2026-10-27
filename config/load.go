// Package config loads application settings and persists the effective configuration in a gob cache.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"assetgoblin/utils"

	"github.com/spf13/viper"
)

// Config represents the main application configuration.
// It contains settings for the server, image processing, rate limiting, and security.
type Config struct {
	Image            Image     `mapstructure:"image"`
	Port             string    `mapstructure:"port"`
	PublicDir        string    `mapstructure:"public_dir"`
	RateLimit        RateLimit `mapstructure:"rate_limit"`
	Secret           string    `mapstructure:"secret"`
	UsedConfigFile   string    `mapstructure:"-" json:"used_config_file"`
	LoadedFromGob    bool      `mapstructure:"-" json:"loaded_from_gob"`
	PythonExecutable string    `mapstructure:"python_executable"`
}

// Image contains configuration for image processing and serving.
type Image struct {
	AvifThroughVips bool                         `mapstructure:"avif_through_vips"`
	CacheDir        string                       `mapstructure:"cache_dir"`
	Directory       string                       `mapstructure:"directory"`
	Formats         []string                     `mapstructure:"formats"`
	Path            string                       `mapstructure:"path"`
	Presets         map[string]utils.ImagePreset `mapstructure:"presets"`
}

// RateLimit contains configuration for request rate limiting.
type RateLimit struct {
	Limit int           `mapstructure:"limit"`
	Ttl   time.Duration `mapstructure:"ttl"`
}

// Load prefers the gob cache, then reads configuration files with defaults.
// Cache write failures are logged without preventing startup.
func (config *Config) Load() error {
	var loaded Config
	if err := loaded.loadGob(); err == nil {
		loaded.LoadedFromGob = true
		*config = loaded
		return nil
	}
	loaded = Config{}
	settings := viper.New()
	settings.SetDefault("port", "8080")
	settings.SetDefault("public_dir", "public")
	settings.SetDefault("secret", "")
	settings.SetDefault("python_executable", "")

	settings.SetDefault("rate_limit.limit", 0)
	settings.SetDefault("rate_limit.ttl", "1m")

	settings.SetDefault("image.formats", []string{"avif", "jpeg", "jpg", "png", "tiff", "webp"})
	settings.SetDefault("image.presets", map[string]utils.ImagePreset{})
	settings.SetDefault("image.path", "/img/")
	settings.SetDefault("image.directory", "assets/img")
	settings.SetDefault("image.cache_dir", filepath.Join(defaultCacheDir(), "img"))
	settings.SetDefault("image.avif_through_vips", false)
	settings.SetConfigName("config")
	for _, directory := range searchConfigPaths() {
		settings.AddConfigPath(directory)
	}
	if err := settings.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			return fmt.Errorf("unable to read config file: %w", err)
		}
		loaded.UsedConfigFile = "none (defaults only)"
	} else {
		loaded.UsedConfigFile = settings.ConfigFileUsed()
	}
	if err := settings.Unmarshal(&loaded); err != nil {
		return fmt.Errorf("unable to decode config file: %w", err)
	}
	if err := loaded.normalizePresets(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if err := loaded.saveGob(); err != nil {
		slog.Warn("Failed to save gob file", "error", err)
	}
	*config = loaded
	return nil
}
