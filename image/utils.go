// Package image provides functionality for processing and serving images.
// It supports resizing images to different presets and converting between formats.
package image

import (
	"assetgoblin/config"
	"assetgoblin/utils"
	"fmt"
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
	width      int
	height     int
	fit        FitMode
	hasSize    bool
	rotate     int
	flip       string
	crop       string
	brightness float64
	contrast   float64
	gamma      float64
	filters    []string
}

// parseSize parses a preset name, size string (e.g., "640" or "640x480"), or direct dimensions.
// It checks presets first, then falls back to direct size parsing.
// Returns resize option string, size parts (including transforms), and whether it's a preset.
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
			width:      width,
			height:     height,
			fit:        fit,
			hasSize:    width > 0,
			rotate:     preset.Rotate,
			flip:       preset.Flip,
			crop:       preset.Crop,
			brightness: preset.Brightness,
			contrast:   preset.Contrast,
			gamma:      preset.Gamma,
			filters:    preset.Filters,
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

// buildVipsCommand builds a libvips command for image processing.
func buildVipsCommand(input, output string, resizeOption string, parts sizeParts) *exec.Cmd {
	if !parts.hasSize && parts.rotate == 0 && parts.flip == "" && len(parts.filters) == 0 {
		return exec.Command("vips", "copy", input, output)
	}

	hasTransforms := parts.rotate > 0 || parts.flip != "" || len(parts.filters) > 0

	if parts.hasSize {
		if parts.fit == FitModeCover {
			if hasTransforms {
				tmp := strings.TrimSuffix(output, filepath.Ext(output)) + "_tmp." + filepath.Ext(output)
				cmd := exec.Command("vips", "cover", input, tmp, strconv.Itoa(parts.width), strconv.Itoa(parts.height))
				cmd = addVipsTransforms(cmd, tmp, output, parts)
				return cmd
			}
			return exec.Command("vips", "cover", input, output, strconv.Itoa(parts.width), strconv.Itoa(parts.height))
		}
		if hasTransforms {
			tmp := strings.TrimSuffix(output, filepath.Ext(output)) + "_tmp." + filepath.Ext(output)
			cmd := exec.Command("vips", "thumbnail", input, tmp, resizeOption)
			cmd = addVipsTransforms(cmd, tmp, output, parts)
			return cmd
		}
		return exec.Command("vips", "thumbnail", input, output, resizeOption)
	}

	if hasTransforms {
		tmp := strings.TrimSuffix(output, filepath.Ext(output)) + "_tmp." + filepath.Ext(output)
		cmd := exec.Command("vips", "copy", input, tmp)
		cmd = addVipsTransforms(cmd, tmp, output, parts)
		return cmd
	}

	return exec.Command("vips", "copy", input, output)
}

// addVipsTransforms adds image transformation operations to a vips command.
func addVipsTransforms(cmd *exec.Cmd, input, output string, parts sizeParts) *exec.Cmd {
	// Brightness/Contrast adjustments
	if parts.brightness != 0 || parts.contrast != 0 {
		tmp := strings.TrimSuffix(output, filepath.Ext(output)) + "_bc." + filepath.Ext(output)
		// Use VIPS math operations for brightness/contrast
		// brightness: scale factor (1 = no change), offset
		// contrast: multiply factor
		brightnessFactor := 1.0 + parts.brightness/100.0
		contrastFactor := 1.0 + parts.contrast/100.0
		cmd.Args = append(cmd.Args, "&&", "vips", "multiply", input, tmp,
			"--coef", fmt.Sprintf("%.2f", brightnessFactor*contrastFactor))
		input = tmp
	}

	if parts.gamma > 0 && parts.gamma != 1.0 {
		tmp := strings.TrimSuffix(output, filepath.Ext(output)) + "_g." + filepath.Ext(output)
		cmd.Args = append(cmd.Args, "&&", "vips", "gamma", input, tmp,
			"--exponent", fmt.Sprintf("%.2f", 1.0/parts.gamma))
		input = tmp
	}

	if parts.rotate > 0 {
		tmp := strings.TrimSuffix(output, filepath.Ext(output)) + "_r." + filepath.Ext(output)
		cmd.Args = append(cmd.Args, "&&", "vips", "rotate", input, tmp, "--angle", strconv.Itoa(parts.rotate))
		input = tmp
	}

	if parts.flip == "horizontal" {
		tmp := strings.TrimSuffix(output, filepath.Ext(output)) + "_f." + filepath.Ext(output)
		cmd.Args = append(cmd.Args, "&&", "vips", "flop", input, tmp)
		input = tmp
	} else if parts.flip == "vertical" {
		tmp := strings.TrimSuffix(output, filepath.Ext(output)) + "_f." + filepath.Ext(output)
		cmd.Args = append(cmd.Args, "&&", "vips", "flip", input, tmp)
		input = tmp
	}

	for _, filter := range parts.filters {
		tmp := strings.TrimSuffix(output, filepath.Ext(output)) + "_" + filter + "." + filepath.Ext(output)
		switch filter {
		case "grayscale":
			cmd.Args = append(cmd.Args, "&&", "vips", "grayscale", input, tmp)
		case "sepia":
			cmd.Args = append(cmd.Args, "&&", "vips", "sRGB2grey", input, tmp)
		case "blur":
			cmd.Args = append(cmd.Args, "&&", "vips", "blur", input, tmp, "3")
		case "sharpen":
			cmd.Args = append(cmd.Args, "&&", "vips", "sharpen", input, tmp)
		case "negate", "invert":
			cmd.Args = append(cmd.Args, "&&", "vips", "invert", input, tmp)
		case "normalize":
			cmd.Args = append(cmd.Args, "&&", "vips", "normalise", input, tmp)
		case "equalize":
			cmd.Args = append(cmd.Args, "&&", "vips", "histeq", input, tmp)
		case "contrast":
			cmd.Args = append(cmd.Args, "&&", "vips", "contrast", input, tmp, "1")
		case "edge":
			cmd.Args = append(cmd.Args, "&&", "vips", "canny", input, tmp)
		case "emboss":
			cmd.Args = append(cmd.Args, "&&", "vips", "freud", input, tmp)
		case "charcoal":
			cmd.Args = append(cmd.Args, "&&", "vips", "conv", input, tmp, "0,-1,0,-1,5,-1,0,-1,0")
		case "solarize":
			cmd.Args = append(cmd.Args, "&&", "vips", "math2", input, tmp, "sin", "--factor", "128")
		case "paint":
			cmd.Args = append(cmd.Args, "&&", "vips", "median", input, tmp, "3")
		case "oil":
			cmd.Args = append(cmd.Args, "&&", "vips", "median", input, tmp, "5")
		case "sketch":
			cmd.Args = append(cmd.Args, "&&", "vips", "sobel", input, tmp)
		case "vignette":
			cmd.Args = append(cmd.Args, "&&", "vips", "radial", input, tmp, "--scale", "0.5")
		}
		input = tmp
	}

	if input != output {
		cmd.Args = append(cmd.Args, "&&", "vips", "copy", input, output)
	}

	return cmd
}

// buildConvertCommand builds an ImageMagick convert command for image processing.
func buildConvertCommand(prefix, input, output string, resizeOption string, parts sizeParts) *exec.Cmd {
	args := []string{}

	if parts.hasSize {
		if parts.fit == FitModeCover {
			args = append(args, input, "-resize", resizeOption+"^", "-gravity", "center", "-extent", resizeOption)
		} else {
			args = append(args, input, "-resize", resizeOption)
		}
	} else {
		args = append(args, input)
	}

	if parts.brightness != 0 || parts.contrast != 0 {
		args = append(args, "-brightness-contrast", fmt.Sprintf("%.2f", parts.brightness), fmt.Sprintf("%.2f", parts.contrast))
	}

	if parts.gamma > 0 && parts.gamma != 1.0 {
		args = append(args, "-gamma", fmt.Sprintf("%.2f", parts.gamma))
	}

	if parts.rotate > 0 {
		args = append(args, "-rotate", strconv.Itoa(parts.rotate))
	}

	if parts.flip == "horizontal" {
		args = append(args, "-flop")
	} else if parts.flip == "vertical" {
		args = append(args, "-flip")
	}

	for _, filter := range parts.filters {
		switch filter {
		case "grayscale":
			args = append(args, "-grayscale")
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

	return exec.Command(prefix+"convert", args...)
}

// ensureAbsolute converts a relative path to an absolute path using the working directory.
func ensureAbsolute(path, wd string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(wd, path)
}
