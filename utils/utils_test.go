package utils

import (
	"os"
	"testing"
)

// TestCloseFile verifies that CloseFile closes file handles.
func TestCloseFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-close-file-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	CloseFile(tmpFile)

	_, err = tmpFile.Write([]byte("test"))
	if err == nil {
		t.Errorf("Expected error writing to closed file, got nil")
	}
}

// TestCloseReader verifies that CloseReader closes readers.
func TestCloseReader(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-close-reader-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.Close()

	CloseReader(tmpFile)

	_, err = tmpFile.Write([]byte("test"))
	if err == nil {
		t.Errorf("Expected error writing to closed file, got nil")
	}
}
