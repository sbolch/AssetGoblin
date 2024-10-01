package config

import (
	"encoding/gob"
	"fmt"
	"os"
)

type Config struct {
	Image     *image
	Port      string
	PublicDir string
}

type image struct {
	AvifThroughVips bool
	CacheDir        string
	Directory       string
	Formats         []string
	Path            string
	Presets         map[string]string
}

func (config *Config) Load() error {
	file, err := os.Open("config.gob")
	if err != nil && os.IsNotExist(err) {
		jsonConfig, err := loadJson()
		if err != nil {
			return err
		}

		config.Port = jsonConfig.Port
		config.PublicDir = jsonConfig.PublicDir
		config.Image = &image{
			AvifThroughVips: jsonConfig.Image.AvifThroughVips,
			CacheDir:        jsonConfig.Image.CacheDir,
			Directory:       jsonConfig.Image.Directory,
			Formats:         jsonConfig.Image.Formats,
			Path:            jsonConfig.Image.Path,
			Presets:         jsonConfig.Image.Presets,
		}

		if err := config.saveGob(); err != nil {
			return err
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("unable to open config file: %w", err)
	}
	defer closeFile(file)

	decoder := gob.NewDecoder(file)
	if err = decoder.Decode(config); err != nil {
		return fmt.Errorf("unable to decode config file: %w", err)
	}

	return nil
}
