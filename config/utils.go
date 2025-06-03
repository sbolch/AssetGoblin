package config

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

func closeFile(file *os.File) {
	if err := file.Close(); err != nil {
		log.Printf("Warning: %v\n", err)
	}
}

func (config *Config) saveGob() error {
	file, err := os.Create("config.gob")
	if err != nil {
		log.Printf("Warning: %v\n", err)
		return nil
	}
	defer closeFile(file)

	encoder := gob.NewEncoder(file)
	if err = encoder.Encode(config); err != nil {
		return fmt.Errorf("unable to encode config: %w", err)
	}

	return nil
}

func (config *Config) loadGob() error {
	file, err := os.Open("config.gob")
	if err != nil {
		return fmt.Errorf("unable to open config file: %w", err)
	}
	defer closeFile(file)

	decoder := gob.NewDecoder(file)
	if err = decoder.Decode(config); err != nil {
		return fmt.Errorf("unable to decode config file: %w", err)
	}

	return nil
}
