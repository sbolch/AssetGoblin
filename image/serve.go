// Package image provides functionality for processing and serving images.
// It supports resizing images to different presets and converting between formats.
package image

import (
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Serve handles HTTP requests for images, processing them according to the requested preset.
// It extracts the preset name and image path from the URL, finds the image file,
// resizes it according to the preset, caches the result, and serves it to the client.
// If the image is already cached, it serves the cached version directly.
// The URL format should be: /[base_path]/[preset_name]/[image_path]
func (s *Service) Serve(res http.ResponseWriter, req *http.Request) {
	slog.Info("Request received", "method", req.Method, "path", req.URL.Path, "remote", req.RemoteAddr, "user-agent", req.UserAgent())

	wd, _ := os.Getwd()

	imageDir := s.Config.Directory
	if !filepath.IsAbs(imageDir) {
		imageDir = filepath.Join(wd, imageDir)
	}

	cacheDir := s.Config.CacheDir
	if !filepath.IsAbs(cacheDir) {
		cacheDir = filepath.Join(wd, cacheDir)
	}

	splitPath := strings.Split(req.URL.Path, "/")
	if len(splitPath) < 4 {
		http.NotFound(res, req)
		return
	}

	presetName := splitPath[2]
	path := strings.Join(splitPath[3:], "/")

	requestedPath := filepath.Join(imageDir, path)
	requestedExt := strings.ToLower(filepath.Ext(requestedPath))

	if !slices.Contains(s.Config.Formats, strings.TrimPrefix(requestedExt, ".")) {
		http.Error(res, "Unsupported format: "+requestedExt, http.StatusBadRequest)
		return
	}

	presetValue, hasPreset := s.Config.Presets[presetName]
	if !hasPreset {
		http.Error(res, "Unsupported preset: "+presetName, http.StatusBadRequest)
		return
	}

	// Search for the file with any of the supported formats
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

	finalPath := filepath.Join(cacheDir, relSourcePath, presetName+requestedExt)

	// If cached version doesn't exist, generate, cache, and serve it
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
			slog.Error("Error while creating cache", "error", err)
			http.Error(res, "Error while creating cache", http.StatusInternalServerError)
			return
		}

		vipsEncoded := false
		if requestedExt != ".avif" || s.Config.AvifThroughVips {
			cmd := exec.Command("vips", "thumbnail", foundPath, finalPath, presetValue)
			if err = cmd.Run(); err == nil {
				vipsEncoded = true
			}
		}
		if !vipsEncoded {
			prefix := ""
			if runtime.GOOS == "windows" {
				prefix = "magick "
			}
			cmd := exec.Command(prefix+"convert", foundPath, "-resize", presetValue, finalPath)
			if err = cmd.Run(); err != nil {
				slog.Error("Error while converting image", "error", err)
				http.Error(res, "Error while converting image", http.StatusInternalServerError)
				return
			}
		}
	}

	http.ServeFile(res, req, finalPath)
}
