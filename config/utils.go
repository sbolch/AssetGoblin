// Package config provides functionality for loading and managing application configuration.
package config

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// closeFile safely closes a file and logs any errors that occur.
func closeFile(file *os.File) {
	if err := file.Close(); err != nil {
		log.Printf("Warning: %v\n", err)
	}
}

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
		log.Printf("Warning: %v\n", err)
		return nil
	}

	file, err := os.Create(GobFilePath())
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

// loadGob deserializes the Config struct from a gob file.
// It returns an error if the file cannot be opened or if the decoding fails.
func (config *Config) loadGob() error {
	file, err := os.Open(GobFilePath())
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
