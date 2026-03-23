// Package image provides functionality for processing and serving images.
// It supports resizing images to different presets and converting between formats.
package image

import (
	"assetgoblin/config"
	"assetgoblin/utils"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Service handles image processing and serving operations.
// It uses the configuration provided to determine how to process and serve images.
type Service struct {
	Config *config.Image
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
// Returns true if the format is supported, false otherwise.
func (s *Service) isValidFormat(format string) bool {
	return slices.Contains(s.Config.Formats, strings.TrimPrefix(format, "."))
}

type sizeParts struct {
	width   int
	height  int
	fit     FitMode
	hasSize bool
}

// parseSize parses a size string (e.g., "640" or "640x480"), preset name, or direct dimensions with fit mode query parameter.
// It checks presets first, then falls back to direct size parsing.
// Returns resize option string, size parts, and whether it's a preset.
func parseSize(presetOrSize, queryFit string, presets map[string]utils.ImagePreset) (string, sizeParts, bool) {
	preset, isPreset := presets[presetOrSize]
	if isPreset {
		fit := FitModeContain
		if preset.Fit == "cover" || queryFit == "cover" {
			fit = FitModeCover
		}

		width := preset.Width
		height := preset.Height

		resizeOption := strconv.Itoa(width)
		if height > 0 {
			resizeOption = strconv.Itoa(width) + "x" + strconv.Itoa(height)
		}

		parts := sizeParts{
			width:   width,
			height:  height,
			fit:     fit,
			hasSize: width > 0,
		}
		return resizeOption, parts, true
	}

	if strings.Contains(presetOrSize, "x") {
		parts := strings.Split(presetOrSize, "x")
		if len(parts) != 2 {
			return "", sizeParts{}, false
		}
		width, err := strconv.Atoi(parts[0])
		if err != nil {
			return "", sizeParts{}, false
		}
		height, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", sizeParts{}, false
		}
		if width <= 0 || height <= 0 {
			return "", sizeParts{}, false
		}

		fit := FitModeContain
		if queryFit == "cover" {
			fit = FitModeCover
		}

		return strconv.Itoa(width) + "x" + strconv.Itoa(height), sizeParts{
			width:   width,
			height:  height,
			fit:     fit,
			hasSize: true,
		}, false
	}

	width, err := strconv.Atoi(presetOrSize)
	if err != nil || width <= 0 {
		return "", sizeParts{}, false
	}

	return strconv.Itoa(width), sizeParts{width: width, hasSize: false}, false
}

func buildVipsCommand(input, output string, resizeOption string, parts sizeParts) *exec.Cmd {
	if !parts.hasSize {
		return exec.Command("vips", "thumbnail", input, output, resizeOption)
	}

	if parts.fit == FitModeCover {
		return exec.Command("vips", "cover", input, output, strconv.Itoa(parts.width), strconv.Itoa(parts.height))
	}

	return exec.Command("vips", "thumbnail", input, output, resizeOption)
}

func buildConvertCommand(prefix, input, output string, resizeOption string, parts sizeParts) *exec.Cmd {
	if !parts.hasSize {
		return exec.Command(prefix+"convert", input, "-resize", resizeOption, output)
	}

	if parts.fit == FitModeCover {
		return exec.Command(prefix+"convert", input, "-resize", resizeOption+"^", "-gravity", "center", "-extent", resizeOption, output)
	}

	return exec.Command(prefix+"convert", input, "-resize", resizeOption, output)
}

func ensureAbsolute(path, wd string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(wd, path)
}
