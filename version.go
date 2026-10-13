// Package main provides version management functionality for the application.
// This file contains functions for checking, downloading, and installing updates.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
	"unicode"

	"assetgoblin/utils"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

// Version is the current version of the application.
// It is set to "development" by default and can be overridden during build.
var Version = "development"

// BuildTime is the build timestamp.
// It is set during build time.
var BuildTime = "unknown"

// GitCommit is the git commit hash.
// It is set during build time.
var GitCommit = "unknown"

var r *release

// archMap maps Go architecture identifiers to the architecture names used in release files.
var archMap = map[string]string{
	"amd64": "x64",
	"386":   "x86",
	"arm64": "arm64",
}

// release represents a GitHub release with its assets and metadata.
type release struct {
	Assets  []releaseAsset `json:"assets"`
	Body    string         `json:"body"`
	TagName string         `json:"tag_name"`
}

// releaseAsset represents a downloadable asset from a GitHub release.
type releaseAsset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
	Name               string `json:"name"`
}

// supportFilesDir returns the install location for non-binary files based on
// the current executable path and platform conventions.
func supportFilesDir(binaryPath string) string {
	binDir := filepath.Dir(binaryPath)

	switch runtime.GOOS {
	case "linux":
		if binDir == "/usr/bin" {
			return "/usr/share/assetgoblin"
		}
		if binDir == "/usr/local/bin" {
			return "/usr/local/share/assetgoblin"
		}
	case "darwin":
		if binDir == "/usr/local/bin" {
			return "/usr/local/share/assetgoblin"
		}
		if binDir == "/opt/homebrew/bin" {
			return "/opt/homebrew/share/assetgoblin"
		}
	case "windows":
		programFiles := os.Getenv("ProgramFiles")
		if programFiles == "" {
			programFiles = `C:\Program Files`
		}
		if strings.EqualFold(binDir, filepath.Join(programFiles, "AssetGoblin")) {
			return filepath.Join(programFiles, "AssetGoblin", "share")
		}
	}

	return binDir
}

// getLatestVersion fetches information about the latest release from GitHub
// and returns the tag name (version) of that release.
// It returns an error if the request fails or if the response cannot be parsed.
func getLatestVersion() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Get("https://api.github.com/repos/sbolch/AssetGoblin/releases/latest")
	if err != nil {
		return "", err
	}
	defer utils.CloseReader(res.Body)

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch releases: %s", res.Status)
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return "", err
	}

	return r.TagName, nil
}

// update checks for and installs the latest version of the application.
// It downloads the appropriate release for the current platform, verifies its checksum,
// extracts it, and replaces the current executable with the updated version.
// The function exits the application after completion, either with success or failure.
func update() {
	latest, err := getLatestVersion()
	if err != nil {
		slog.Error("Error getting the latest version", "error", err)
		os.Exit(1)
	}

	if latest == Version {
		printSuccess("Already up to date.")
		os.Exit(0)
	}

	lipgloss.Fprintln(os.Stdout, cliHeading.Render("Updating "+Version+" → "+latest))

	updateName := releaseArchiveName(latest, runtime.GOOS, runtime.GOARCH)

	updateDownloadURL, found := findUpdateURL(r.Assets, updateName)
	if !found {
		slog.Error("Failed to find update", "os", runtime.GOOS)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Get(updateDownloadURL)
	if err != nil {
		slog.Error("Failed to download update", "error", err)
		os.Exit(1)
	}
	defer utils.CloseReader(res.Body)

	if res.StatusCode != http.StatusOK {
		slog.Error("Failed to download update", "status", res.Status)
		os.Exit(1)
	}

	tmpFile, err := os.CreateTemp("", "AssetGoblin-update-*")
	if err != nil {
		slog.Error("Failed to create temporary file", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			slog.Warn("Failed to remove temp file", "error", err)
		}
	}()

	progress := utils.NewUpdateProgress(os.Stdout, res.ContentLength, term.IsTerminal(os.Stdout.Fd()) && os.Getenv("TERM") != "dumb", cliHeading)
	_, err = io.Copy(tmpFile, io.TeeReader(res.Body, progress))
	progress.Finish(err == nil)
	if err != nil {
		slog.Error("Failed to write temporary file", "error", err)
		os.Exit(1)
	}

	utils.CloseFile(tmpFile)

	lipgloss.Fprintln(os.Stdout, cliHeading.Render("Verifying checksum…"))

	checksumURL, found := findChecksumURL(r.Assets)
	if !found {
		slog.Error("Failed to verify update")
		os.Exit(1)
	}
	res, err = client.Get(checksumURL)
	if err != nil {
		slog.Error("Failed to download checksum file", "error", err)
		os.Exit(1)
	}
	defer utils.CloseReader(res.Body)

	if res.StatusCode != http.StatusOK {
		slog.Error("Failed to download checksum file", "status", res.Status)
		os.Exit(1)
	}

	var checksums map[string]string
	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Failed to read checksum file", "error", err)
		os.Exit(1)
	}
	checksums = parseChecksums(string(body))
	checksum, ok := checksums[updateName]
	if !ok {
		slog.Error("Failed to find checksum", "file", updateName)
		os.Exit(1)
	}

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		slog.Error("Failed to open archive", "error", err)
		os.Exit(1)
	}
	defer utils.CloseFile(file)

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		slog.Error("Failed to calculate checksum", "error", err)
		os.Exit(1)
	}

	calculatedChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	if calculatedChecksum != checksum {
		slog.Error("Checksum mismatch", "expected", checksum, "got", calculatedChecksum)
		os.Exit(1)
	}

	printSuccess("Checksum verified.")
	lipgloss.Fprintln(os.Stdout, cliHeading.Render("Extracting update…"))

	wd, _ := os.Getwd()
	currentExecutablePath, err := os.Executable()
	if err != nil {
		currentExecutablePath = filepath.Join(wd, "assetgoblin")
		if runtime.GOOS == "windows" {
			currentExecutablePath += ".exe"
		}
	}
	supportFilesPath := supportFilesDir(currentExecutablePath)

	switch runtime.GOOS {
	case "darwin", "linux":
		file, err := os.Open(tmpFile.Name())
		if err != nil {
			slog.Error("Failed to open archive", "error", err)
			os.Exit(1)
		}
		defer utils.CloseFile(file)

		gz, err := gzip.NewReader(file)
		if err != nil {
			slog.Error("Failed to open archive", "error", err)
			os.Exit(1)
		}
		defer gz.Close()

		tr := tar.NewReader(gz)
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				slog.Error("Failed to extract archive", "error", err)
				os.Exit(1)
			}

			if header.FileInfo().IsDir() {
				continue
			}

			filePath := filepath.Join(supportFilesPath, header.Name)
			if header.Name == "assetgoblin" {
				filePath = filepath.Join(wd, "assetgoblin_"+latest)
			}
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				slog.Error("Failed to create file directory", "error", err)
				os.Exit(1)
			}
			outFile, err := os.Create(filePath)
			if err != nil {
				slog.Error("Failed to create file", "error", err)
				os.Exit(1)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				utils.CloseFile(outFile)
				slog.Error("Failed to write file", "error", err)
				os.Exit(1)
			}
			utils.CloseFile(outFile)
		}
	case "windows":
		reader, err := zip.OpenReader(tmpFile.Name())
		if err != nil {
			slog.Error("Failed to open archive", "error", err)
			os.Exit(1)
		}
		defer reader.Close()

		for _, zf := range reader.File {
			if zf.FileInfo().IsDir() {
				continue
			}

			filePath := filepath.Join(supportFilesPath, zf.Name)
			if zf.Name == "assetgoblin.exe" {
				filePath = filepath.Join(wd, "assetgoblin_"+latest+".exe")
			}
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				slog.Error("Failed to create file directory", "error", err)
				os.Exit(1)
			}
			outFile, err := os.Create(filePath)
			if err != nil {
				slog.Error("Failed to create file", "error", err)
				os.Exit(1)
			}
			rc, err := zf.Open()
			if err != nil {
				utils.CloseFile(outFile)
				slog.Error("Failed to open file", "error", err)
				os.Exit(1)
			}
			if _, err := io.Copy(outFile, rc); err != nil {
				utils.CloseFile(outFile)
				utils.CloseReader(rc)
				slog.Error("Failed to write file", "error", err)
				os.Exit(1)
			}
			utils.CloseFile(outFile)
			utils.CloseReader(rc)
		}
	default:
		slog.Error("Unknown file extension", "file", updateName)
		os.Exit(1)
	}

	lipgloss.Fprintln(os.Stdout, cliHeading.Render("Installing update…"))
	currentExecutable := filepath.Join(wd, "assetgoblin")
	backupExecutable := filepath.Join(wd, "assetgoblin_"+Version)
	newExecutable := filepath.Join(wd, "assetgoblin_"+latest)
	if runtime.GOOS == "windows" {
		currentExecutable += ".exe"
		backupExecutable += ".exe"
		newExecutable += ".exe"
	}

	err = os.Rename(currentExecutable, backupExecutable)
	if err != nil {
		slog.Error("Failed to backup current executable", "error", err)
		os.Exit(1)
	}

	err = os.Rename(newExecutable, currentExecutable)
	if err != nil {
		slog.Error("Failed to replace current executable", "error", err)
		os.Exit(1)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(currentExecutable, 0755); err != nil {
			slog.Error("Failed to set permissions on new executable", "error", err)
			os.Exit(1)
		}
	}

	err = os.Remove(backupExecutable)
	if err != nil {
		slog.Warn("Failed to remove backup executable", "error", err)
	}

	printSuccess("Update finished successfully.")
	width := 80
	if columns, _, err := term.GetSize(os.Stdout.Fd()); err == nil && columns > 0 {
		width = min(columns, width)
	}
	printReleaseNotes(os.Stdout, r.Body, width)

	os.Exit(0)
}

// releaseArchiveName selects the platform naming convention used by releases.
func releaseArchiveName(version, goos, goarch string) string {
	osName := "macOS"
	if goos != "darwin" {
		chars := []rune(goos)
		if len(chars) > 0 {
			chars[0] = unicode.ToUpper(chars[0])
		}
		osName = string(chars)
	}
	extension := ".tar.gz"
	if goos == "windows" {
		extension = ".zip"
	}
	return fmt.Sprintf("AssetGoblin_%s_%s_%s%s", version, osName, archMap[goarch], extension)
}

// parseChecksums ignores malformed lines and indexes checksums by archive name.
func parseChecksums(data string) map[string]string {
	checksums := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			checksums[parts[1]] = parts[0]
		}
	}
	return checksums
}

// findUpdateURL selects the exact archive requested by the updater.
func findUpdateURL(assets []releaseAsset, name string) (string, bool) {
	index := slices.IndexFunc(assets, func(asset releaseAsset) bool { return asset.Name == name })
	if index < 0 {
		return "", false
	}
	return assets[index].BrowserDownloadURL, true
}

// findChecksumURL selects the checksum document using the release naming convention.
func findChecksumURL(assets []releaseAsset) (string, bool) {
	index := slices.IndexFunc(assets, func(asset releaseAsset) bool { return strings.Contains(asset.Name, "checksums") })
	if index < 0 {
		return "", false
	}
	return assets[index].BrowserDownloadURL, true
}
