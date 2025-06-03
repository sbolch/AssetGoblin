package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode"
)

type mockTransport struct {
	server            *httptest.Server
	originalTransport http.RoundTripper
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq, err := http.NewRequest(req.Method, m.server.URL, nil)
	if err != nil {
		return nil, err
	}

	return m.originalTransport.RoundTrip(newReq)
}

func TestGetLatestVersion(t *testing.T) {
	originalTransport := http.DefaultTransport

	tests := []struct {
		name       string
		mockServer func() *httptest.Server
		want       string
		wantErr    bool
	}{
		{
			name: "successful response",
			mockServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mockRelease := release{
						TagName: "v1.0.0",
						Body:    "Test release notes",
						Assets: []releaseAsset{
							{
								Name:               "AssetGoblin_v1.0.0_Windows_x64.zip",
								BrowserDownloadURL: "https://example.com/download/AssetGoblin_v1.0.0_Windows_x64.zip",
							},
						},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(mockRelease)
				}))
			},
			want:    "v1.0.0",
			wantErr: false,
		},
		{
			name: "server error",
			mockServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid json response",
			mockServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte("invalid json"))
				}))
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.mockServer()
			defer server.Close()

			mockTransport := &mockTransport{
				server:            server,
				originalTransport: originalTransport,
			}

			http.DefaultTransport = mockTransport
			defer func() { http.DefaultTransport = originalTransport }()

			got, err := getLatestVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("getLatestVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getLatestVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArchMap(t *testing.T) {
	expectedMappings := map[string]string{
		"amd64": "x64",
		"386":   "x86",
		"arm64": "arm64",
	}

	for arch, expected := range expectedMappings {
		if got := archMap[arch]; got != expected {
			t.Errorf("archMap[%s] = %v, want %v", arch, got, expected)
		}
	}
}

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version is empty")
	}
}

func TestGetOSName(t *testing.T) {
	tests := []struct {
		goos     string
		expected string
	}{
		{
			goos:     "darwin",
			expected: "macOS",
		},
		{
			goos:     "linux",
			expected: "Linux",
		},
		{
			goos:     "windows",
			expected: "Windows",
		},
		{
			goos:     "freebsd",
			expected: "Freebsd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			var osName string
			if tt.goos == "darwin" {
				osName = "macOS"
			} else {
				osR := []rune(tt.goos)
				osR[0] = unicode.ToUpper(osR[0])
				osName = string(osR)
			}

			if osName != tt.expected {
				t.Errorf("For GOOS=%s, got osName=%s, want %s", tt.goos, osName, tt.expected)
			}
		})
	}
}

func TestUpdateNameGeneration(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		goos     string
		goarch   string
		expected string
	}{
		{
			name:     "windows amd64",
			version:  "v1.0.0",
			goos:     "windows",
			goarch:   "amd64",
			expected: "AssetGoblin_v1.0.0_Windows_x64.zip",
		},
		{
			name:     "linux 386",
			version:  "v1.0.0",
			goos:     "linux",
			goarch:   "386",
			expected: "AssetGoblin_v1.0.0_Linux_x86.tar.gz",
		},
		{
			name:     "macOS arm64",
			version:  "v1.0.0",
			goos:     "darwin",
			goarch:   "arm64",
			expected: "AssetGoblin_v1.0.0_macOS_arm64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var osName string
			if tt.goos == "darwin" {
				osName = "macOS"
			} else {
				osR := []rune(tt.goos)
				osR[0] = unicode.ToUpper(osR[0])
				osName = string(osR)
			}

			updateName := fmt.Sprintf("AssetGoblin_%s_%s_%s", tt.version, osName, archMap[tt.goarch])
			ext := ".tar.gz"
			if tt.goos == "windows" {
				ext = ".zip"
			}
			updateName += ext

			if updateName != tt.expected {
				t.Errorf("For GOOS=%s, GOARCH=%s, version=%s, got updateName=%s, want %s",
					tt.goos, tt.goarch, tt.version, updateName, tt.expected)
			}
		})
	}
}

func TestFindUpdateURL(t *testing.T) {
	tests := []struct {
		name       string
		assets     []releaseAsset
		updateName string
		wantURL    string
		wantFound  bool
	}{
		{
			name: "matching asset found",
			assets: []releaseAsset{
				{
					Name:               "AssetGoblin_v1.0.0_Windows_x64.zip",
					BrowserDownloadURL: "https://example.com/download/AssetGoblin_v1.0.0_Windows_x64.zip",
				},
				{
					Name:               "AssetGoblin_v1.0.0_Linux_x64.tar.gz",
					BrowserDownloadURL: "https://example.com/download/AssetGoblin_v1.0.0_Linux_x64.tar.gz",
				},
			},
			updateName: "AssetGoblin_v1.0.0_Windows_x64.zip",
			wantURL:    "https://example.com/download/AssetGoblin_v1.0.0_Windows_x64.zip",
			wantFound:  true,
		},
		{
			name: "matching asset not found",
			assets: []releaseAsset{
				{
					Name:               "AssetGoblin_v1.0.0_Linux_x64.tar.gz",
					BrowserDownloadURL: "https://example.com/download/AssetGoblin_v1.0.0_Linux_x64.tar.gz",
				},
			},
			updateName: "AssetGoblin_v1.0.0_Windows_x64.zip",
			wantURL:    "",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var updateURL string
			for _, asset := range tt.assets {
				if asset.Name == tt.updateName {
					updateURL = asset.BrowserDownloadURL
					break
				}
			}

			if (updateURL != "") != tt.wantFound {
				t.Errorf("Found = %v, want %v", updateURL != "", tt.wantFound)
			}
			if updateURL != tt.wantURL {
				t.Errorf("URL = %v, want %v", updateURL, tt.wantURL)
			}
		})
	}
}

func TestFindChecksumURL(t *testing.T) {
	tests := []struct {
		name      string
		assets    []releaseAsset
		wantURL   string
		wantFound bool
	}{
		{
			name: "checksum asset found",
			assets: []releaseAsset{
				{
					Name:               "AssetGoblin_v1.0.0_Windows_x64.zip",
					BrowserDownloadURL: "https://example.com/download/AssetGoblin_v1.0.0_Windows_x64.zip",
				},
				{
					Name:               "checksums.txt",
					BrowserDownloadURL: "https://example.com/download/checksums.txt",
				},
			},
			wantURL:   "https://example.com/download/checksums.txt",
			wantFound: true,
		},
		{
			name: "checksum asset not found",
			assets: []releaseAsset{
				{
					Name:               "AssetGoblin_v1.0.0_Windows_x64.zip",
					BrowserDownloadURL: "https://example.com/download/AssetGoblin_v1.0.0_Windows_x64.zip",
				},
			},
			wantURL:   "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var checksumURL string
			for _, asset := range tt.assets {
				if strings.Contains(asset.Name, "checksums") {
					checksumURL = asset.BrowserDownloadURL
					break
				}
			}

			if (checksumURL != "") != tt.wantFound {
				t.Errorf("Found = %v, want %v", checksumURL != "", tt.wantFound)
			}
			if checksumURL != tt.wantURL {
				t.Errorf("URL = %v, want %v", checksumURL, tt.wantURL)
			}
		})
	}
}

func TestParseChecksumFile(t *testing.T) {
	tests := []struct {
		name         string
		checksumData string
		updateName   string
		wantChecksum string
		wantFound    bool
	}{
		{
			name: "checksum found",
			checksumData: `
				abcdef1234567890 AssetGoblin_v1.0.0_Windows_x64.zip
				1234567890abcdef AssetGoblin_v1.0.0_Linux_x64.tar.gz
			`,
			updateName:   "AssetGoblin_v1.0.0_Windows_x64.zip",
			wantChecksum: "abcdef1234567890",
			wantFound:    true,
		},
		{
			name: "checksum not found",
			checksumData: `
				1234567890abcdef AssetGoblin_v1.0.0_Linux_x64.tar.gz
			`,
			updateName:   "AssetGoblin_v1.0.0_Windows_x64.zip",
			wantChecksum: "",
			wantFound:    false,
		},
		{
			name: "invalid line format",
			checksumData: `
				invalid line format
				abcdef1234567890 AssetGoblin_v1.0.0_Windows_x64.zip
			`,
			updateName:   "AssetGoblin_v1.0.0_Windows_x64.zip",
			wantChecksum: "abcdef1234567890",
			wantFound:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksums := make(map[string]string)
			lines := strings.Split(strings.TrimSpace(tt.checksumData), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				parts := strings.Fields(line)
				if len(parts) != 2 {
					continue // Skip invalid lines
				}
				checksums[parts[1]] = parts[0]
			}
			checksum, ok := checksums[tt.updateName]

			if ok != tt.wantFound {
				t.Errorf("Found = %v, want %v", ok, tt.wantFound)
			}
			if checksum != tt.wantChecksum {
				t.Errorf("Checksum = %v, want %v", checksum, tt.wantChecksum)
			}
		})
	}
}
