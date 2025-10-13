// Package image provides functionality for processing and serving images.
// It supports resizing images to different presets and converting between formats.
package image

import (
	"assetgoblin/config"
	"os"
)

// Service handles image processing and serving operations.
// It uses the configuration provided to determine how to process and serve images.
type Service struct {
	Config *config.Image // Configuration for image processing
}

// findImage searches for an image file with any of the supported formats.
// It takes a base path without extension and tries to find a file by appending
// each of the supported extensions. Returns the full path of the found image and true
// if an image is found, or an empty string and false otherwise.
func (s *Service) findImage(base string) (string, bool) {
	for _, ext := range s.Config.Formats {
		path := base + "." + ext
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return path, true
		}
	}
	return "", false
}

// isValidFormat checks if the given format is supported by the service.
// The format should include the leading dot (e.g., ".jpg").
// Returns true if the format is supported, false otherwise.
func (s *Service) isValidFormat(format string) bool {
	for _, ext := range s.Config.Formats {
		if format == "."+ext {
			return true
		}
	}
	return false
}
