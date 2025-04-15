package image

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func (s *Service) Serve(res http.ResponseWriter, req *http.Request) {
	log.Println(req.Method, req.URL.Path, req.RemoteAddr, req.UserAgent())

	wd, _ := os.Getwd()

	splitPath := strings.Split(req.URL.Path, "/")
	if len(splitPath) < 4 {
		http.NotFound(res, req)
		return
	}

	presetName := splitPath[2]
	path := strings.Join(splitPath[3:], "/")

	requestedPath := filepath.Join(wd, s.Config.Directory, path)
	requestedExt := strings.ToLower(filepath.Ext(requestedPath))

	// Check if the requested format is valid
	if !s.isValidFormat(requestedExt) {
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

	finalPath := filepath.Join(
		wd,
		s.Config.CacheDir,
		strings.TrimPrefix(strings.TrimSuffix(foundPath, filepath.Ext(foundPath)), filepath.Join(wd, s.Config.Directory)),
		presetName+requestedExt,
	)

	// If cached version doesn't exist, generate, cache, and serve it
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(finalPath), os.ModePerm); err != nil {
			log.Println("Error while creating cache:", err)
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
				log.Println("Error while converting image:", err)
				http.Error(res, "Error while converting image", http.StatusInternalServerError)
				return
			}
		}
	}

	http.ServeFile(res, req, finalPath)
}
