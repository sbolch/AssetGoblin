package main

import (
	"os"
	"path/filepath"

	"assetgoblin/plugin"
)

func loadPlugins(directory, pythonExecutable string) (*plugin.Manager, error) {
	directory, err := pluginDirectory(directory)
	if err != nil {
		return nil, err
	}
	return plugin.LoadWithPython(directory, pythonExecutable)
}

func pluginDirectory(directory string) (string, error) {
	if directory != "" {
		return filepath.Abs(directory)
	}
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executable), nil
}
