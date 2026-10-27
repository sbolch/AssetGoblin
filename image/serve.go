package image

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Serve resolves a preset or dimensions, applies validated query transformations,
// and serves the cached image. Registered codecs bridge plugin formats through PNG.
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
	if outputCodec == nil && !s.isValidFormat(requestedExt) {
		http.Error(res, "Unsupported format: "+requestedExt, http.StatusBadRequest)
		return
	}

	query := req.URL.Query()
	queryFit := query.Get("fit")
	queryRotate := query.Get("rotate")
	queryFlip := query.Get("flip")
	queryCrop := query.Get("crop")

	resizeOption, parts := parseSize(presetOrSizes, queryFit, s.Config.Presets)
	if resizeOption == "" {
		http.Error(res, "Unsupported preset or size: "+presetOrSizes, http.StatusBadRequest)
		return
	}

	if queryRotate != "" {
		r, err := strconv.Atoi(queryRotate)
		if err == nil && (r == 0 || r == 90 || r == 180 || r == 270) {
			parts.Rotate = r
		}
	}
	if queryFlip != "" {
		if queryFlip == "horizontal" || queryFlip == "vertical" || queryFlip == "both" {
			parts.Flip = queryFlip
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
			parts.Crop = queryCrop
		}
	}
	queryBrightness := query.Get("brightness")
	queryContrast := query.Get("contrast")
	queryGamma := query.Get("gamma")
	filterStr := query.Get("filter")

	if b, err := strconv.ParseFloat(queryBrightness, 64); err == nil && b >= -100 && b <= 100 {
		parts.Brightness = b
	}
	if c, err := strconv.ParseFloat(queryContrast, 64); err == nil && c >= -100 && c <= 100 {
		parts.Contrast = c
	}
	if g, err := strconv.ParseFloat(queryGamma, 64); err == nil && g >= 0.1 && g <= 10.0 {
		parts.Gamma = g
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
				parts.Filters = append(parts.Filters, f)
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

	cacheKey := presetOrSizes + "_" + parts.Fit
	if parts.Rotate > 0 {
		cacheKey += "_r" + strconv.Itoa(parts.Rotate)
	}
	if parts.Flip != "" {
		cacheKey += "_f" + parts.Flip
	}
	if parts.Brightness != 0 {
		cacheKey += "_b" + strconv.FormatFloat(parts.Brightness, 'f', -1, 64)
	}
	if parts.Contrast != 0 {
		cacheKey += "_c" + strconv.FormatFloat(parts.Contrast, 'f', -1, 64)
	}
	if parts.Gamma > 0 && parts.Gamma != 1.0 {
		cacheKey += "_g" + strconv.FormatFloat(parts.Gamma, 'f', -1, 64)
	}
	if len(parts.Filters) > 0 {
		cacheKey += "_" + strings.Join(parts.Filters, ",")
	}
	if parts.Crop != "" {
		cacheKey += "_crop" + parts.Crop
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
			cmd := buildVipsCommand(req.Context(), foundPath, finalPath, parts)
			if cmd != nil && cmd.Run() == nil {
				vipsEncoded = true
			}
		}
		if !vipsEncoded {
			cmd := buildConvertCommand(req.Context(), foundPath, finalPath, resizeOption, parts)
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
