package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type jsonConfig struct {
	Image     *jsonImage `json:"image"`
	Port      string     `json:"port"`
	PublicDir string     `json:"public_dir"`
}

type jsonImage struct {
	AvifThroughVips bool              `json:"avif_through_vips"`
	CacheDir        string            `json:"cache_dir"`
	Directory       string            `json:"directory"`
	Formats         []string          `json:"formats"`
	Path            string            `json:"path"`
	Presets         map[string]string `json:"presets"`
}

func loadJson() (jsonConfig, error) {
	data, err := os.ReadFile("config.json")
	if err != nil {
		return jsonConfig{}, fmt.Errorf("unable to read config file: %w", err)
	}

	var config jsonConfig

	if err = json.Unmarshal(data, &config); err != nil {
		return jsonConfig{}, fmt.Errorf("unable to unmarshal config file: %w", err)
	}

	return config, nil
}
