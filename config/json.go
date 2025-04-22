package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type jsonConfig struct {
	Image struct {
		AvifThroughVips bool              `json:"avif_through_vips"`
		CacheDir        string            `json:"cache_dir"`
		Directory       string            `json:"directory"`
		Formats         []string          `json:"formats"`
		Path            string            `json:"path"`
		Presets         map[string]string `json:"presets"`
	} `json:"image"`
	Port      string `json:"port"`
	PublicDir string `json:"public_dir"`
	RateLimit struct {
		Limit int      `json:"limit"`
		Ttl   Duration `json:"ttl"`
	} `json:"rate_limit"`
	Secret string `json:"secret"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	d.Duration = duration

	return nil
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
