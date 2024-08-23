package image

import (
	"asset-manager/config"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var conf config.Config

func init() {
	var err error
	conf, err = config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
}

func Serve(res http.ResponseWriter, req *http.Request) {
	log.Println(req.Method, req.URL.Path, req.RemoteAddr, req.UserAgent())

	wd, _ := os.Getwd()

	splitPath := strings.Split(req.URL.Path, "/")
	if len(splitPath) < 4 {
		http.NotFound(res, req)
		return
	}

	presetName := splitPath[2]
	path := strings.Join(splitPath[3:], "/")

	requestedPath := wd + "/assets/img/" + path
	requestedExt := strings.ToLower(filepath.Ext(requestedPath))

	// Check if the requested format is valid
	if !isValidFormat(requestedExt) {
		http.Error(res, "Unsupported format: "+requestedExt, http.StatusBadRequest)
		return
	}

	presetValue, hasPreset := conf.Image.Presets[presetName]
	if !hasPreset {
		http.Error(res, "Unsupported preset: "+presetName, http.StatusBadRequest)
		return
	}

	// Search for the file with any of the supported formats
	foundPath, found := findImage(strings.TrimSuffix(requestedPath, requestedExt))
	if !found {
		http.NotFound(res, req)
		return
	}

	finalPath := wd + "/cache" + strings.TrimPrefix(strings.TrimSuffix(foundPath, filepath.Ext(foundPath)), wd+"/assets") + "/" + presetName + requestedExt

	// If cached version doesn't exist, generate, cache, and serve it
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(finalPath), os.ModePerm); err != nil {
			http.Error(res, "Error while converting image #1", http.StatusInternalServerError)
			return
		}
		vipsEncoded := false
		// vips avif encoding is really slow atm
		if requestedExt != ".avif" {
			cmd := exec.Command("vips", "thumbnail", foundPath, finalPath, presetValue)
			if err = cmd.Run(); err == nil {
				vipsEncoded = true
			}
		}
		if !vipsEncoded {
			cmd := exec.Command("convert", foundPath, "-resize", presetValue, finalPath)
			if err = cmd.Run(); err != nil {
				http.Error(res, "Error while converting image #2", http.StatusInternalServerError)
				return
			}
		}
	}

	http.ServeFile(res, req, finalPath)
}

func isValidFormat(format string) bool {
	for _, ext := range conf.Image.Formats {
		if format == "."+ext {
			return true
		}
	}
	return false
}

func findImage(base string) (string, bool) {
	for _, ext := range conf.Image.Formats {
		path := base + "." + ext
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return path, true
		}
	}
	return "", false
}
