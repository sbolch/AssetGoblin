// Package image provides functionality for processing and serving images.
// It supports resizing images to different presets and converting between formats.
package image

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

type FitMode string

const (
	FitModeContain FitMode = "contain"
	FitModeCover   FitMode = "cover"
)

// Serve handles HTTP requests for images, processing them according to the requested preset.
// It extracts the preset name and image path from the URL, finds the image file,
// resizes it according to the preset with optional transforms (rotate, flip, brightness, contrast, gamma, filters),
// caches the result, and serves it to the client.
// If the image is already cached, it serves the cached version directly.
// The URL format should be: /[base_path]/[preset_name]/[image_path]
// Alternatively, direct dimensions can be used: /[base_path]/[width]/[image_path] or /[base_path]/[width]x[height]/[image_path]
// Query parameters: fit, rotate, flip, crop, brightness, contrast, gamma, filter
func (s *Service) Serve(res http.ResponseWriter, req *http.Request) {
	slog.Info("Request received", "method", req.Method, "path", req.URL.Path, "remote", req.RemoteAddr, "user-agent", req.UserAgent())

	wd, _ := os.Getwd()

	imageDir := ensureAbsolute(s.Config.Directory, wd)
	cacheDir := ensureAbsolute(s.Config.CacheDir, wd)

	prefix := s.Config.Path
	if prefix == "" {
		prefix = "/img/"
	}
	relative, ok := strings.CutPrefix(req.URL.Path, prefix)
	presetOrSizes, path, hasPath := strings.Cut(relative, "/")
	if !ok || !hasPath || presetOrSizes == "" || !filepath.IsLocal(path) {
		http.NotFound(res, req)
		return
	}
	requestedPath := filepath.Join(imageDir, path)

	requestedExt := strings.ToLower(filepath.Ext(requestedPath))

	outputCodec := s.Codecs[strings.TrimPrefix(requestedExt, ".")]
	if outputCodec == nil && !slices.Contains(s.Config.Formats, strings.TrimPrefix(requestedExt, ".")) {
		http.Error(res, "Unsupported format: "+requestedExt, http.StatusBadRequest)
		return
	}

	query := req.URL.Query()
	queryFit := query.Get("fit")
	queryRotate := query.Get("rotate")
	queryFlip := query.Get("flip")
	queryCrop := query.Get("crop")

	resizeOption, sizeParts, isPreset := parseSize(presetOrSizes, queryFit, s.Config.Presets)
	if resizeOption == "" && !isPreset {
		http.Error(res, "Unsupported preset or size: "+presetOrSizes, http.StatusBadRequest)
		return
	}

	if queryRotate != "" {
		r, err := strconv.Atoi(queryRotate)
		if err == nil && (r == 0 || r == 90 || r == 180 || r == 270) {
			sizeParts.rotate = r
		}
	}
	if queryFlip != "" {
		if queryFlip == "horizontal" || queryFlip == "vertical" || queryFlip == "both" {
			sizeParts.flip = queryFlip
		}
	}
	if queryCrop != "" {
		validCrops := map[string]bool{
			"top-left":     true,
			"top":          true,
			"top-right":    true,
			"left":         true,
			"center":       true,
			"right":        true,
			"bottom-left":  true,
			"bottom":       true,
			"bottom-right": true,
		}
		if validCrops[queryCrop] {
			sizeParts.crop = queryCrop
		}
	}
	queryBrightness := query.Get("brightness")
	queryContrast := query.Get("contrast")
	queryGamma := query.Get("gamma")
	filterStr := query.Get("filter")

	if b, err := strconv.ParseFloat(queryBrightness, 64); err == nil && b >= -100 && b <= 100 {
		sizeParts.brightness = b
	}
	if c, err := strconv.ParseFloat(queryContrast, 64); err == nil && c >= -100 && c <= 100 {
		sizeParts.contrast = c
	}
	if g, err := strconv.ParseFloat(queryGamma, 64); err == nil && g >= 0.1 && g <= 10.0 {
		sizeParts.gamma = g
	}

	if filterStr != "" {
		queryFilters := strings.Split(filterStr, ",")
		validFilters := map[string]bool{
			"grayscale": true,
			"sepia":     true,
			"blur":      true,
			"sharpen":   true,
			"negate":    true,
			"invert":    true,
			"normalize": true,
			"equalize":  true,
			"contrast":  true,
			"edge":      true,
			"emboss":    true,
			"charcoal":  true,
			"solarize":  true,
			"paint":     true,
			"oil":       true,
			"sketch":    true,
			"vignette":  true,
		}
		for _, f := range queryFilters {
			f = strings.TrimSpace(f)
			if validFilters[f] {
				sizeParts.filters = append(sizeParts.filters, f)
			}
		}
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

	inputCodec := s.Codecs[strings.TrimPrefix(strings.ToLower(filepath.Ext(foundPath)), ".")]
	revision := ""
	if inputCodec != nil {
		relativeSource, err := filepath.Rel(imageDir, foundPath)
		if err != nil {
			http.NotFound(res, req)
			return
		}
		root, err := os.OpenRoot(imageDir)
		if err != nil {
			http.NotFound(res, req)
			return
		}
		defer root.Close()
		file, err := root.Open(relativeSource)
		if err != nil {
			http.NotFound(res, req)
			return
		}
		decoded, err := inputCodec.Decode(req.Context(), file, foundPath)
		file.Close()
		if err != nil {
			slog.Error("Image decode failed", "error", err)
			http.Error(res, "Image decode failed", 500)
			return
		}
		revision += inputCodec.Revision()
		foundPath = decoded
	}
	if outputCodec != nil {
		revision += outputCodec.Revision()
	}

	cacheKey := presetOrSizes
	if sizeParts.hasSize {
		cacheKey = presetOrSizes + "_" + string(sizeParts.fit)
	}
	if sizeParts.rotate > 0 {
		cacheKey += "_r" + strconv.Itoa(sizeParts.rotate)
	}
	if sizeParts.flip != "" {
		cacheKey += "_f" + sizeParts.flip
	}
	if sizeParts.brightness != 0 {
		cacheKey += "_b" + strconv.FormatFloat(sizeParts.brightness, 'f', -1, 64)
	}
	if sizeParts.contrast != 0 {
		cacheKey += "_c" + strconv.FormatFloat(sizeParts.contrast, 'f', -1, 64)
	}
	if sizeParts.gamma > 0 && sizeParts.gamma != 1.0 {
		cacheKey += "_g" + strconv.FormatFloat(sizeParts.gamma, 'f', -1, 64)
	}
	if len(sizeParts.filters) > 0 {
		cacheKey += "_" + strings.Join(sizeParts.filters, ",")
	}
	if sizeParts.crop != "" {
		cacheKey += "_crop" + sizeParts.crop
	}

	processingExt := requestedExt
	if outputCodec != nil {
		processingExt = ".png"
	}
	if revision != "" {
		info, err := os.Stat(foundPath)
		if err != nil {
			http.NotFound(res, req)
			return
		}
		cacheKey += fmt.Sprintf("_%x", sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d:%d", requestedExt, revision, info.Size(), info.ModTime().UnixNano()))))
	}
	finalPath := filepath.Join(cacheDir, relSourcePath, cacheKey+processingExt)

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

	if outputCodec != nil {
		file, err := os.Open(finalPath)
		if err != nil {
			http.Error(res, "Image cache unavailable", 500)
			return
		}
		encoded, err := outputCodec.Encode(req.Context(), file, finalPath)
		file.Close()
		if err != nil {
			slog.Error("Image encode failed", "error", err)
			http.Error(res, "Image encode failed", 500)
			return
		}
		finalPath = encoded
		res.Header().Set("Content-Type", outputCodec.MIMEType())
	}
	http.ServeFile(res, req, finalPath)
}
