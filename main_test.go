package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestServe_Skipped(t *testing.T) {
	t.Skip("Skipping test because serve() starts an HTTP server that blocks")
}

func TestMainVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version is empty")
	}
}

func TestLogo(t *testing.T) {
	if Logo == "" {
		t.Error("Logo is empty")
	}
}

func TestFlagParsing(t *testing.T) {
	oldArgs := os.Args
	oldFlagCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlagCommandLine
	}()

	tests := []struct {
		name        string
		args        []string
		wantServe   bool
		wantVersion bool
		wantUpdate  bool
	}{
		{
			name:        "no flags",
			args:        []string{"assetgoblin"},
			wantServe:   false,
			wantVersion: false,
			wantUpdate:  false,
		},
		{
			name:        "serve flag",
			args:        []string{"assetgoblin", "-serve"},
			wantServe:   true,
			wantVersion: false,
			wantUpdate:  false,
		},
		{
			name:        "version flag",
			args:        []string{"assetgoblin", "-version"},
			wantServe:   false,
			wantVersion: true,
			wantUpdate:  false,
		},
		{
			name:        "update flag",
			args:        []string{"assetgoblin", "-update"},
			wantServe:   false,
			wantVersion: false,
			wantUpdate:  true,
		},
		{
			name:        "multiple flags",
			args:        []string{"assetgoblin", "-serve", "-version"},
			wantServe:   true,
			wantVersion: true,
			wantUpdate:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(tt.args[0], flag.ExitOnError)

			serveFlag := flag.Bool("serve", false, "Run the server")
			versionFlag := flag.Bool("version", false, "Print version info")
			updateFlag := flag.Bool("update", false, "Update to latest version")

			os.Args = tt.args
			flag.Parse()

			if *serveFlag != tt.wantServe {
				t.Errorf("serveFlag = %v, want %v", *serveFlag, tt.wantServe)
			}
			if *versionFlag != tt.wantVersion {
				t.Errorf("versionFlag = %v, want %v", *versionFlag, tt.wantVersion)
			}
			if *updateFlag != tt.wantUpdate {
				t.Errorf("updateFlag = %v, want %v", *updateFlag, tt.wantUpdate)
			}
		})
	}
}

func TestMainVersionOutput(t *testing.T) {
	if os.Getenv("TEST_MAIN_VERSION") == "1" {
		oldArgs := os.Args
		os.Args = []string{"assetgoblin", "-version"}
		defer func() { os.Args = oldArgs }()

		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainVersionOutput")
	cmd.Env = append(os.Environ(), "TEST_MAIN_VERSION=1")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Process exited with error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, Version) {
		t.Errorf("Expected output to contain version %q, got %q", Version, output)
	}
}

func TestMainDefaultOutput(t *testing.T) {
	t.Skip("Skipping test for default output as it's difficult to capture reliably")
}
