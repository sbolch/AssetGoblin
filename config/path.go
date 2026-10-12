package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const appDirName = "assetgoblin"

// searchConfigPaths returns ordered directories to search for config files.
// The current working directory is checked first, then user-level and system-level paths.
func searchConfigPaths() []string {
	paths := []string{"."}

	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		paths = append(paths, filepath.Join(userConfigDir, appDirName))
	}

	switch runtime.GOOS {
	case "linux":
		paths = append(paths, filepath.Join("/etc", appDirName))
	case "windows":
		programData := os.Getenv("ProgramData")
		if programData != "" {
			paths = append(paths, filepath.Join(programData, appDirName))
		}
	case "darwin":
		paths = append(paths, filepath.Join("/Library/Application Support", appDirName))
	}

	return paths
}

// defaultCacheDir returns the OS-specific cache directory for AssetGoblin.
// It prefers the per-user cache location and falls back to system paths.
func defaultCacheDir() string {
	userCacheDir, err := os.UserCacheDir()
	if err == nil {
		return filepath.Join(userCacheDir, appDirName)
	}

	switch runtime.GOOS {
	case "linux":
		return filepath.Join("/var/cache", appDirName)
	case "windows":
		programData := os.Getenv("ProgramData")
		if programData != "" {
			return filepath.Join(programData, appDirName, "cache")
		}
	case "darwin":
		return filepath.Join("/Library/Caches", appDirName)
	}

	return filepath.Join(".", "cache")
}
