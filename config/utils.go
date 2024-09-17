package config

import (
	"encoding/gob"
	"fmt"
	"log"
)
import "os"

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

	encoder := gob.NewEncoder(file)
	if err = encoder.Encode(config); err != nil {
		return fmt.Errorf("unable to encode config: %w", err)
	}

	defer closeFile(file)

	return nil
}
