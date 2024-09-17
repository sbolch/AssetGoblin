package image

import "os"

func findImage(base string) (string, bool) {
	for _, ext := range conf.Image.Formats {
		path := base + "." + ext
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return path, true
		}
	}
	return "", false
}

func isValidFormat(format string) bool {
	for _, ext := range conf.Image.Formats {
		if format == "."+ext {
			return true
		}
	}
	return false
}
