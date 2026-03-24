package config

import (
	"encoding/gob"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"assetgoblin/utils"
)

// GobFilePath returns the full path to the cached gob config file.
func GobFilePath() string {
	return filepath.Join(defaultCacheDir(), "config.gob")
}

// RemoveGobFile deletes the cached gob config file if it exists.
func RemoveGobFile() error {
	if err := os.Remove(GobFilePath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to remove gob file: %w", err)
	}

	return nil
}

// saveGob serializes the Config struct to a gob file for faster loading in the future.
// It returns an error if the file cannot be created or if the encoding fails.
// If the file cannot be created, it logs a warning and returns nil to allow the application to continue.
func (config *Config) saveGob() error {
	if err := os.MkdirAll(defaultCacheDir(), 0755); err != nil {
		slog.Warn("Failed to create cache directory", "error", err)
		return nil
	}

	file, err := os.Create(GobFilePath())
	if err != nil {
		slog.Warn("Failed to create gob file", "error", err)
		return nil
	}
	defer utils.CloseFile(file)

	encoder := gob.NewEncoder(file)
	if err = encoder.Encode(config); err != nil {
		return fmt.Errorf("unable to encode config: %w", err)
	}

	return nil
}

// loadGob deserializes the Config struct from a gob file.
// It returns an error if the file cannot be opened or if the decoding fails.
func (config *Config) loadGob() error {
	file, err := os.Open(GobFilePath())
	if err != nil {
		return fmt.Errorf("unable to open config file: %w", err)
	}
	defer utils.CloseFile(file)

	decoder := gob.NewDecoder(file)
	if err = decoder.Decode(config); err != nil {
		return fmt.Errorf("unable to decode config file: %w", err)
	}

	return nil
}

// normalizePresets normalizes presets by setting default Fit value and validates all preset fields.
func (config *Config) normalizePresets() error {
	if config.Image.Presets == nil {
		return nil
	}

	validRotations := map[int]bool{0: true, 90: true, 180: true, 270: true}
	validFlips := map[string]bool{"": true, "horizontal": true, "vertical": true, "both": true}
	validCrops := map[string]bool{
		"":             true,
		"top-left":     true,
		"top":          true,
		"top-right":    true,
		"left":         true,
		"center":       true,
		"right":        true,
		"bottom-left":  true,
		"bottom":       true,
		"bottom-right": true,
	}
	validFilters := map[string]bool{
		"":          true,
		"grayscale": true,
		"sepia":     true,
		"blur":      true,
		"sharpen":   true,
		"negate":    true,
		"edge":      true,
		"emboss":    true,
		"charcoal":  true,
		"solarize":  true,
		"normalize": true,
		"equalize":  true,
		"contrast":  true,
		"paint":     true,
		"oil":       true,
		"sketch":    true,
		"vignette":  true,
	}

	normalized := make(map[string]utils.ImagePreset)
	for name, p := range config.Image.Presets {
		if p.Width <= 0 {
			return fmt.Errorf("preset %q: width is required and must be greater than 0", name)
		}
		if p.Fit == "" {
			p.Fit = "contain"
		}
		if !validRotations[p.Rotate] {
			return fmt.Errorf("preset %q: rotate must be 0, 90, 180, or 270", name)
		}
		if !validFlips[p.Flip] {
			return fmt.Errorf("preset %q: flip must be empty, horizontal, vertical, or both", name)
		}
		if !validCrops[p.Crop] {
			return fmt.Errorf("preset %q: invalid crop region", name)
		}
		if p.Brightness < -100 || p.Brightness > 100 {
			return fmt.Errorf("preset %q: brightness must be between -100 and 100", name)
		}
		if p.Contrast < -100 || p.Contrast > 100 {
			return fmt.Errorf("preset %q: contrast must be between -100 and 100", name)
		}
		if p.Gamma < 0.1 || p.Gamma > 10.0 {
			return fmt.Errorf("preset %q: gamma must be between 0.1 and 10.0", name)
		}
		for _, f := range p.Filters {
			if !validFilters[f] {
				return fmt.Errorf("preset %q: invalid filter %q", name, f)
			}
		}
		normalized[name] = p
	}
	config.Image.Presets = normalized
	return nil
}
