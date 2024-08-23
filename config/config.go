package config

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Config struct {
	Port  string
	Image image
}

type image struct {
	Formats []string
	Presets map[string]string
}

type jsonConfig struct {
	Port  string    `json:"port"`
	Image jsonImage `json:"image"`
}

type jsonImage struct {
	Formats []string          `json:"formats"`
	Presets map[string]string `json:"presets"`
}

func Load() (Config, error) {
	var config Config

	file, err := os.Open("config.gob")
	if err != nil && os.IsNotExist(err) {
		jsonConfig, err := loadJson()
		if err != nil {
			return Config{}, err
		}

		config = Config{
			Port: jsonConfig.Port,
			Image: image{
				Formats: jsonConfig.Image.Formats,
				Presets: jsonConfig.Image.Presets,
			},
		}

		if err := saveGob(config); err != nil {
			return Config{}, err
		}

		return config, nil
	} else if err != nil {
		return Config{}, fmt.Errorf("unable to open config file: %w", err)
	}

	decoder := gob.NewDecoder(file)
	if err = decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("unable to decode config file: %w", err)
	}

	defer closeFile(file)

	return config, nil
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

func saveGob(config Config) error {
	file, err := os.Create("config.gob")
	if err != nil {
		log.Printf("Warning: %v\n", err)
		return nil
	}

	encoder := gob.NewEncoder(file)
	if err = encoder.Encode(config); err != nil {
		return fmt.Errorf("unable to encode config: %w", err)
	}

	defer closeFile(file)

	return nil
}

func closeFile(file *os.File) {
	if err := file.Close(); err != nil {
		log.Printf("Warning: %v\n", err)
	}
}
