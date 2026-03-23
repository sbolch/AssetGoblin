// Package utils provides common utility functions.
package utils

import (
	"io"
	"log/slog"
	"os"
)

// CloseFile safely closes an *os.File and logs any errors that occur.
func CloseFile(file *os.File) {
	if err := file.Close(); err != nil {
		slog.Warn("Failed to close file", "error", err)
	}
}

// CloseReader safely closes an io.ReadCloser and logs any errors that occur.
func CloseReader(r io.ReadCloser) {
	if err := r.Close(); err != nil {
		slog.Warn("Failed to close reader", "error", err)
	}
}
