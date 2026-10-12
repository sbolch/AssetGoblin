package main

import (
	"assetgoblin/plugin"
	"os"
	"path/filepath"
)

// pluginDirectory defaults to the executable's directory, never the working directory.
func pluginDirectory(override string) (string, error) {
	if override != "" {
		return filepath.Abs(override)
	}
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executable), nil
}

func loadPlugins(directory string) (*plugin.Manager, error) {
	directory, err := pluginDirectory(directory)
	if err != nil {
		return nil, err
	}
	return plugin.Load(directory)
}
