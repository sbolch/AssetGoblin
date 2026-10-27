// Package image resizes, transforms, and caches images using vips, ImageMagick, and registered codecs.
package image

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"assetgoblin/config"
	"assetgoblin/utils"
)

// Service serves image transformations using the configured formats, presets, and codecs.
type Service struct {
	Config *config.Image
	Codecs map[string]Codec
}

// findImage tries configured extensions first, then registered codecs in sorted order.
func (s *Service) findImage(base string) (string, bool) {
	formats := slices.Clone(s.Config.Formats)
	extra := []string{}
	for name := range s.Codecs {
		extra = append(extra, name)
	}
	slices.Sort(extra)
	formats = append(formats, extra...)
	for _, ext := range formats {
		path := base + "." + ext
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func (s *Service) isValidFormat(format string) bool {
	return slices.Contains(s.Config.Formats, strings.TrimPrefix(format, "."))
}

// parseSize resolves a named preset or positive dimensions. An empty geometry is invalid.
func parseSize(size, queryFit string, presets map[string]utils.ImagePreset) (string, utils.ImagePreset) {
	preset, named := presets[size]
	if !named {
		width, height, dimensions := strings.Cut(size, "x")
		var err error
		preset.Width, err = strconv.Atoi(width)
		if err != nil {
			return "", utils.ImagePreset{}
		}
		if dimensions {
			preset.Height, err = strconv.Atoi(height)
			if err != nil || preset.Height <= 0 {
				return "", utils.ImagePreset{}
			}
		}
	}
	if preset.Width <= 0 {
		return "", utils.ImagePreset{}
	}
	preset.Filters = slices.Clone(preset.Filters)
	if preset.Fit == "cover" || (queryFit == "cover" && (named || preset.Height > 0)) {
		preset.Fit = "cover"
	} else {
		preset.Fit = "contain"
	}
	geometry := strconv.Itoa(preset.Width)
	if preset.Height > 0 {
		geometry += "x" + strconv.Itoa(preset.Height)
	}
	return geometry, preset
}

// buildVipsCommand uses vips for resizing. ImageMagick handles the full transform set.
// A nil command selects ImageMagick directly instead of attempting an invalid pipeline.
func buildVipsCommand(ctx context.Context, input, output string, parts utils.ImagePreset) *exec.Cmd {
	if parts.Rotate != 0 || parts.Flip != "" || parts.Crop != "" || parts.Brightness != 0 || parts.Contrast != 0 ||
		(parts.Gamma != 0 && parts.Gamma != 1) || len(parts.Filters) > 0 {
		return nil
	}
	args := []string{"thumbnail", input, output, strconv.Itoa(parts.Width)}
	if parts.Height > 0 {
		args = append(args, "--height", strconv.Itoa(parts.Height))
		if parts.Fit == "cover" {
			args = append(args, "--crop", "centre")
		}
	}
	return exec.CommandContext(ctx, "vips", args...)
}

// buildConvertCommand builds an ImageMagick convert command for image processing.
func buildConvertCommand(ctx context.Context, input, output, resizeOption string, parts utils.ImagePreset) *exec.Cmd {
	args := []string{input}

	if parts.Fit == "cover" && parts.Height > 0 {
		gravity := map[string]string{"top-left": "NorthWest", "top": "North", "top-right": "NorthEast", "left": "West", "center": "Center", "right": "East", "bottom-left": "SouthWest", "bottom": "South", "bottom-right": "SouthEast"}[parts.Crop]
		if gravity == "" {
			gravity = "Center"
		}
		args = append(args, "-resize", resizeOption+"^", "-gravity", gravity, "-extent", resizeOption)
	} else {
		args = append(args, "-resize", resizeOption)
	}

	if parts.Brightness != 0 || parts.Contrast != 0 {
		args = append(args, "-brightness-contrast", fmt.Sprintf("%.2fx%.2f", parts.Brightness, parts.Contrast))
	}

	if parts.Gamma > 0 && parts.Gamma != 1.0 {
		args = append(args, "-gamma", fmt.Sprintf("%.2f", parts.Gamma))
	}

	if parts.Rotate > 0 {
		args = append(args, "-rotate", strconv.Itoa(parts.Rotate))
	}

	if parts.Flip == "horizontal" || parts.Flip == "both" {
		args = append(args, "-flop")
	}
	if parts.Flip == "vertical" || parts.Flip == "both" {
		args = append(args, "-flip")
	}

	for _, filter := range parts.Filters {
		switch filter {
		case "grayscale":
			args = append(args, "-colorspace", "Gray")
		case "sepia":
			args = append(args, "-sepia-tone", "0.8")
		case "blur":
			args = append(args, "-blur", "0x3")
		case "sharpen":
			args = append(args, "-unsharp", "0x1")
		case "negate", "invert":
			args = append(args, "-negate")
		case "normalize":
			args = append(args, "-normalize")
		case "equalize":
			args = append(args, "-equalize")
		case "contrast":
			args = append(args, "-contrast")
		case "edge":
			args = append(args, "-edge", "1")
		case "emboss":
			args = append(args, "-emboss", "1")
		case "charcoal":
			args = append(args, "-charcoal", "1")
		case "solarize":
			args = append(args, "-solarize", "50")
		case "paint":
			args = append(args, "-paint", "3")
		case "oil":
			args = append(args, "-paint", "5")
		case "sketch":
			args = append(args, "-sketch", "1")
		case "vignette":
			args = append(args, "-vignette", "10x5")
		}
	}

	args = append(args, output)

	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "magick", append([]string{"convert"}, args...)...)
	}
	return exec.CommandContext(ctx, "convert", args...)
}

// ensureAbsolute converts a relative path to an absolute path using the working directory.
func ensureAbsolute(path, wd string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(wd, path)
}
