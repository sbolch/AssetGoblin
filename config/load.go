// Package config provides functionality for loading and managing application configuration.
// It supports loading configuration from files and serializing/deserializing configuration data.
package config

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
)

// Config represents the main application configuration.
// It contains settings for the server, image processing, rate limiting, and security.
type Config struct {
	Image     Image     `mapstructure:"image"`
	Port      string    `mapstructure:"port"`
	PublicDir string    `mapstructure:"public_dir"`
	RateLimit RateLimit `mapstructure:"rate_limit"`
	Secret    string    `mapstructure:"secret"`
}

// Image contains configuration for image processing and serving.
type Image struct {
	AvifThroughVips bool              `mapstructure:"avif_through_vips"`
	CacheDir        string            `mapstructure:"cache_dir"`
	Directory       string            `mapstructure:"directory"`
	Formats         []string          `mapstructure:"formats"`
	Path            string            `mapstructure:"path"`
	Presets         map[string]string `mapstructure:"presets"`
}

// RateLimit contains configuration for request rate limiting.
type RateLimit struct {
	Limit int           `mapstructure:"limit"`
	Ttl   time.Duration `mapstructure:"ttl"`
}

// setDefaults sets default values for configuration parameters using Viper.
func setDefaults() {
	viper.SetDefault("port", "8080")
	viper.SetDefault("public_dir", "public")
	viper.SetDefault("secret", "")

	viper.SetDefault("rate_limit.limit", 0)
	viper.SetDefault("rate_limit.ttl", "1m")

	viper.SetDefault("image.formats", []string{"avif", "jpeg", "jpg", "png", "tiff", "webp"})
	viper.SetDefault("image.presets", map[string]string{
		"lg":   "960",
		"lg2x": "1920",
		"sm":   "640",
		"sm2x": "1280",
	})
	viper.SetDefault("image.path", "/img/")
	viper.SetDefault("image.directory", "assets/img")
	viper.SetDefault("image.cache_dir", "cache")
	viper.SetDefault("image.avif_through_vips", false)
}

// Load loads the configuration from a file or a previously saved gob file.
// It first tries to load from a gob file for faster loading, and if that fails,
// it falls back to loading from a config file using Viper.
// After loading from a config file, it saves the configuration to a gob file for future use.
// Returns an error if the configuration cannot be loaded.
func (config *Config) Load() error {
	setDefaults()

	if err := config.loadGob(); err == nil {
		return nil
	}

	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("unable to read config file: %w", err)
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("unable to decode config file: %w", err)
	}

	if err := config.saveGob(); err != nil {
		log.Printf("Warning: %v\n", err)
	}

	return nil
}
