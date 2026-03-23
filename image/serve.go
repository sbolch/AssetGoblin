// Package image provides functionality for processing and serving images.
// It supports resizing images to different presets and converting between formats.
package image

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

type FitMode string

const (
	FitModeContain FitMode = "contain"
	FitModeCover   FitMode = "cover"
)

// Serve handles HTTP requests for images, processing them according to the requested preset.
// It extracts the preset name and image path from the URL, finds the image file,
// resizes it according to the preset, caches the result, and serves it to the client.
// If the image is already cached, it serves the cached version directly.
// The URL format should be: /[base_path]/[preset_name]/[image_path]
// Alternatively, direct dimensions can be used: /[base_path]/[width]/[image_path] or /[base_path]/[width]x[height]/[image_path]
// When width x height is given, use fit=cover or fit=contain query parameter to determine resizing behavior.
func (s *Service) Serve(res http.ResponseWriter, req *http.Request) {
	slog.Info("Request received", "method", req.Method, "path", req.URL.Path, "remote", req.RemoteAddr, "user-agent", req.UserAgent())

	wd, _ := os.Getwd()

	imageDir := ensureAbsolute(s.Config.Directory, wd)
	cacheDir := ensureAbsolute(s.Config.CacheDir, wd)

	splitPath := strings.Split(req.URL.Path, "/")
	if len(splitPath) < 4 {
		http.NotFound(res, req)
		return
	}

	presetOrDim := splitPath[2]
	path := strings.Join(splitPath[3:], "/")

	requestedPath := filepath.Join(imageDir, path)
	requestedExt := strings.ToLower(filepath.Ext(requestedPath))

	if !slices.Contains(s.Config.Formats, strings.TrimPrefix(requestedExt, ".")) {
		http.Error(res, "Unsupported format: "+requestedExt, http.StatusBadRequest)
		return
	}

	queryFit := req.URL.Query().Get("fit")
	resizeOption, sizeParts, isPreset := parseSize(presetOrDim, queryFit, s.Config.Presets)
	if resizeOption == "" && !isPreset {
		http.Error(res, "Unsupported preset or size: "+presetOrDim, http.StatusBadRequest)
		return
	}

	foundPath, found := s.findImage(strings.TrimSuffix(requestedPath, requestedExt))
	if !found {
		http.NotFound(res, req)
		return
	}

	sourceBasePath := strings.TrimSuffix(foundPath, filepath.Ext(foundPath))
	relSourcePath, err := filepath.Rel(imageDir, sourceBasePath)
	if err != nil {
		slog.Error("Error while resolving source path", "error", err)
		http.Error(res, "Error while resolving source path", http.StatusInternalServerError)
		return
	}

	cacheKey := presetOrDim
	if sizeParts.hasSize {
		cacheKey = presetOrDim + "_" + string(sizeParts.fit)
	}

	finalPath := filepath.Join(cacheDir, relSourcePath, cacheKey+requestedExt)

	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
			slog.Error("Error while creating cache", "error", err)
			http.Error(res, "Error while creating cache", http.StatusInternalServerError)
			return
		}

		vipsEncoded := false
		if requestedExt != ".avif" || s.Config.AvifThroughVips {
			cmd := buildVipsCommand(foundPath, finalPath, resizeOption, sizeParts)
			if err = cmd.Run(); err == nil {
				vipsEncoded = true
			}
		}
		if !vipsEncoded {
			prefix := ""
			if runtime.GOOS == "windows" {
				prefix = "magick "
			}
			cmd := buildConvertCommand(prefix, foundPath, finalPath, resizeOption, sizeParts)
			if err = cmd.Run(); err != nil {
				slog.Error("Error while converting image", "error", err)
				http.Error(res, "Error while converting image", http.StatusInternalServerError)
				return
			}
		}
	}

	http.ServeFile(res, req, finalPath)
}
