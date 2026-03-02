package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchConfigPaths(t *testing.T) {
	isolateConfigAndCacheEnv(t)

	paths := searchConfigPaths()
	if len(paths) < 2 {
		t.Fatalf("Expected at least 2 config search paths, got %d", len(paths))
	}

	if paths[0] != "." {
		t.Fatalf("Expected current directory as first config path, got %s", paths[0])
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("Failed to resolve user config dir: %v", err)
	}
	wantUserConfigPath := filepath.Join(userConfigDir, appDirName)

	foundUserConfigPath := false
	for _, configPath := range paths {
		if configPath == wantUserConfigPath {
			foundUserConfigPath = true
			break
		}
	}
	if !foundUserConfigPath {
		t.Fatalf("Expected %q in config search paths, got %v", wantUserConfigPath, paths)
	}
}

func TestDefaultCacheDir(t *testing.T) {
	isolateConfigAndCacheEnv(t)

	cacheDir := defaultCacheDir()
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("Failed to resolve user cache dir: %v", err)
	}
	wantCacheDir := filepath.Join(userCacheDir, appDirName)
	if cacheDir != wantCacheDir {
		t.Fatalf("Expected default cache dir %q, got %q", wantCacheDir, cacheDir)
	}
}
