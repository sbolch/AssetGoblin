package image

import (
	"assetgoblin/config"
	"os"
)

type Service struct {
	Config *config.Image
}

func (s *Service) findImage(base string) (string, bool) {
	for _, ext := range s.Config.Formats {
		path := base + "." + ext
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return path, true
		}
	}
	return "", false
}

func (s *Service) isValidFormat(format string) bool {
	for _, ext := range s.Config.Formats {
		if format == "."+ext {
			return true
		}
	}
	return false
}
