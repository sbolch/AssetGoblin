package config

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Image     Image     `mapstructure:"image"`
	Port      string    `mapstructure:"port"`
	PublicDir string    `mapstructure:"public_dir"`
	RateLimit RateLimit `mapstructure:"rate_limit"`
	Secret    string    `mapstructure:"secret"`
}

type Image struct {
	AvifThroughVips bool              `mapstructure:"avif_through_vips"`
	CacheDir        string            `mapstructure:"cache_dir"`
	Directory       string            `mapstructure:"directory"`
	Formats         []string          `mapstructure:"formats"`
	Path            string            `mapstructure:"path"`
	Presets         map[string]string `mapstructure:"presets"`
}

type RateLimit struct {
	Limit int           `mapstructure:"limit"`
	Ttl   time.Duration `mapstructure:"ttl"`
}

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
