package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

var Version = "development"
var r *release
var archMap = map[string]string{
	"amd64": "x86_64",
	"386":   "i386",
	"arm64": "arm64",
	"arm":   "arm",
}

type release struct {
	Assets  []releaseAsset `json:"assets"`
	Body    string         `json:"body"`
	TagName string         `json:"tag_name"`
}

type releaseAsset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
	Name               string `json:"name"`
}

func getLatestVersion() (string, error) {
	res, err := http.Get("https://api.github.com/repos/sbolch/AssetGoblin/releases/latest")
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Printf("Warning: %v\n", err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch releases: %s", res.Status)
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return "", err
	}

	return r.TagName, nil
}

func update() {
	log.SetFlags(0)

	latest, err := getLatestVersion()
	if err != nil {
		log.Fatalf("Error getting the latest version: %v", err)
	}

	if latest == Version {
		log.Println("Already up to date.")
		os.Exit(0)
	}

	log.Printf("Updating from %s to %s\n", Version, latest)

	// Get the name of the update for the current OS
	osR := []rune(runtime.GOOS)
	osR[0] = unicode.ToUpper(osR[0])
	osName := string(osR)

	updateName := fmt.Sprintf("AssetGoblin_%s_%s", osName, archMap[runtime.GOARCH])
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	updateName += ext

	// Find the URL of the update
	var updateURL string
	for _, asset := range r.Assets {
		if asset.Name == updateName {
			updateURL = asset.BrowserDownloadURL
			break
		}
	}
	if updateURL == "" {
		log.Fatalf("Failed to find update for %s.", osName)
	}

	log.Println("Downloading updated archive...")

	// Download the update
	res, err := http.Get(updateURL)
	if err != nil {
		log.Fatalf("Failed to download update: %v", err)
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Printf("Warning: %v\n", err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		log.Fatalf("Failed to download update: %s", res.Status)
	}

	tmpFile, err := os.CreateTemp("", "AssetGoblin-update-*")
	if err != nil {
		log.Fatalf("Failed to create temporary file: %v", err)
	}
	defer func(name string) {
		if err := os.Remove(name); err != nil {
			log.Printf("Warning: %v\n", err)
		}
	}(tmpFile.Name())

	_, err = io.Copy(tmpFile, res.Body)
	if err != nil {
		log.Fatalf("Failed to write temporary file: %v", err)
	}

	err = tmpFile.Close()
	if err != nil {
		log.Fatalf("Failed to close temporary file: %v", err)
	}

	// Download the checksum file
	log.Println("Downloading checksum file...")

	var checksumURL string
	for _, asset := range r.Assets {
		if strings.Contains(asset.Name, "checksums") {
			checksumURL = asset.BrowserDownloadURL
			break
		}
	}
	if checksumURL == "" {
		log.Fatal("Failed to verify update.")
	}

	res, err = http.Get(checksumURL)
	if err != nil {
		log.Fatalf("Failed to download checksum file: %v", err)
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Printf("Warning: %v\n", err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		log.Fatalf("Failed to download checksum file: %s", res.Status)
	}

	checksums := make(map[string]string)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Failed to read checksum file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue // Skip invalid lines
		}
		checksums[parts[1]] = parts[0]
	}
	checksum, ok := checksums[updateName]
	if !ok {
		log.Fatalf("Failed to find checksum for %s.", updateName)
	}

	// Verify checksum
	file, err := os.Open(tmpFile.Name())
	if err != nil {
		log.Fatalf("Failed to open archive: %v", err)
	}
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			log.Printf("Warning: %v\n", err)
		}
	}(file)

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		log.Fatalf("Failed to calculate checksum: %v", err)
	}

	calculatedChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	if calculatedChecksum != checksum {
		log.Fatalf("Checksum mismatch: expected %s, got %s", checksum, calculatedChecksum)
	}

	log.Println("Checksum verified.")

	wd, _ := os.Getwd()

	// Extract the update
	switch runtime.GOOS {
	case "darwin", "linux":
		file, err := os.Open(tmpFile.Name())
		if err != nil {
			log.Fatalf("Failed to open archive: %v", err)
		}
		defer func(file *os.File) {
			if err := file.Close(); err != nil {
				log.Printf("Warning: %v\n", err)
			}
		}(file)

		gz, err := gzip.NewReader(file)
		if err != nil {
			log.Fatalf("Failed to open archive: %v", err)
		}
		defer func(gz *gzip.Reader) {
			if err := gz.Close(); err != nil {
				log.Printf("Warning: %v\n", err)
			}
		}(gz)

		tr := tar.NewReader(gz)
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("Failed to extract archive: %v", err)
			}

			filePath := filepath.Join(wd, header.Name)
			if header.Name == "AssetGoblin" {
				filePath = filepath.Join(wd, "AssetGoblin_"+latest)
			}
			outFile, err := os.Create(filePath)
			if err != nil {
				log.Fatalf("Failed to create file: %v", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				if err := outFile.Close(); err != nil {
					log.Printf("Warning: %v\n", err)
				}
				log.Fatalf("Failed to write file: %v", err)
			}
			err = outFile.Close()
			if err != nil {
				log.Printf("Warning: %v\n", err)
			}
		}
	case "windows":
		reader, err := zip.OpenReader(tmpFile.Name())
		if err != nil {
			log.Fatalf("Failed to open archive: %v", err)
		}
		defer func(reader *zip.ReadCloser) {
			if err := reader.Close(); err != nil {
				log.Printf("Warning: %v\n", err)
			}
		}(reader)

		for _, file := range reader.File {
			filePath := filepath.Join(wd, file.Name)
			if file.Name == "AssetGoblin.exe" {
				filePath = filepath.Join(wd, "AssetGoblin_"+latest+".exe")
			}
			outFile, err := os.Create(filePath)
			if err != nil {
				log.Fatalf("Failed to create file: %v", err)
			}
			rc, err := file.Open()
			if err != nil {
				if err := outFile.Close(); err != nil {
					log.Printf("Warning: %v\n", err)
				}
				log.Fatalf("Failed to open file: %v", err)
			}
			if _, err := io.Copy(outFile, rc); err != nil {
				err := outFile.Close()
				if err != nil {
					log.Printf("Warning: %v\n", err)
				}
				err = rc.Close()
				if err != nil {
					log.Printf("Warning: %v\n", err)
				}
				log.Fatalf("Failed to write file: %v", err)
			}
			err = outFile.Close()
			if err != nil {
				log.Printf("Warning: %v\n", err)
			}
			err = rc.Close()
			if err != nil {
				log.Printf("Warning: %v\n", err)
			}
		}
	default:
		log.Fatalf("Unknown file extension: %s", ext)
	}

	// Replace the current executable with the new one
	currentExecutable := filepath.Join(wd, "AssetGoblin")
	backupExecutable := filepath.Join(wd, "AssetGoblin_"+Version)
	newExecutable := filepath.Join(wd, "AssetGoblin_"+latest)
	if runtime.GOOS == "windows" {
		currentExecutable += ".exe"
		backupExecutable += ".exe"
		newExecutable += ".exe"
	}

	err = os.Rename(currentExecutable, backupExecutable)
	if err != nil {
		log.Fatalf("Failed to backup current executable: %v", err)
	}

	err = os.Rename(newExecutable, currentExecutable)
	if err != nil {
		log.Fatalf("Failed to replace current executable: %v", err)
	}

	// Set the permissions on the new executable
	if runtime.GOOS != "windows" {
		if err := os.Chmod(currentExecutable, 0755); err != nil {
			log.Fatalf("Failed to set permissions on new executable: %v", err)
		}
	}

	// Remove the old executable
	err = os.Remove(backupExecutable)
	if err != nil {
		log.Printf("Warning: %v\n", err)
	}

	log.Println("Update finished successfully.")

	// Print the release notes
	log.Println(r.Body)

	os.Exit(0)
}
