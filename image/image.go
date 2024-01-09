package image

import (
	"asset-manager/settings"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type config struct {
	Formats []string          `json:"image_formats"`
	Presets map[string]string `json:"image_presets"`
}

var options config

func init() {
	settings.Load(&options)
}

func Serve(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())

	wd, _ := os.Getwd()

	splitPath := strings.Split(r.URL.Path, "/")
	if len(splitPath) < 4 {
		http.NotFound(w, r)
		return
	}

	presetName := splitPath[2]
	path := strings.Join(splitPath[3:], "/")

	requestedPath := wd + "/assets/img/" + path
	requestedExt := strings.ToLower(filepath.Ext(requestedPath))

	// Check if the requested format is valid
	if !isValidFormat(requestedExt) {
		http.Error(w, "Unsupported format: "+requestedExt, http.StatusBadRequest)
		return
	}

	presetValue, hasPreset := options.Presets[presetName]
	if !hasPreset {
		http.Error(w, "Unsupported preset: "+presetName, http.StatusBadRequest)
		return
	}

	// Search for the file with any of the supported formats
	foundPath, found := findImage(strings.TrimSuffix(requestedPath, requestedExt))
	if !found {
		http.NotFound(w, r)
		return
	}

	finalPath := wd + "/cache" + strings.TrimPrefix(strings.TrimSuffix(foundPath, filepath.Ext(foundPath)), wd+"/assets") + "/" + presetName + requestedExt

	// If cached version doesn't exist, generate, cache, and serve it
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Dir(finalPath), os.ModePerm)
		if err != nil {
			http.Error(w, "Error while converting image #1", http.StatusInternalServerError)
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
				http.Error(w, "Error while converting image #2", http.StatusInternalServerError)
				return
			}
		}
	}

	http.ServeFile(w, r, finalPath)
}

func isValidFormat(f string) bool {
	for _, format := range options.Formats {
		if f == "."+format {
			return true
		}
	}
	return false
}

func findImage(base string) (string, bool) {
	for _, format := range options.Formats {
		path := base + "." + format
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return path, true
		}
	}
	return "", false
}
