package config

import (
	"assetgoblin/utils"
	"github.com/regen-network/gocuke"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestConfigurationFeatures(t *testing.T) {
	feature, err := filepath.Abs("../features/config/configuration.feature")
	if err != nil {
		t.Fatal(err)
	}
	isolateConfigAndCacheEnv(t)
	t.Chdir(t.TempDir())
	gocuke.NewRunner(t, &configurationSuite{}).Path(feature).Before((*configurationSuite).Before).
		Step(`^the configuration file contains (.*)$`, (*configurationSuite).File).
		Step(`^a complete custom configuration$`, (*configurationSuite).Custom).
		Step(`^configuration is loaded$`, (*configurationSuite).Load).
		Step(`^the default configuration should be used$`, (*configurationSuite).Defaults).
		Step(`^configuration should be marked as loaded from (disk|gob)$`, (*configurationSuite).Origin).
		Step(`^the configured port should be "([^"]*)"$`, (*configurationSuite).Port).
		Step(`^the public directory should be "([^"]*)"$`, (*configurationSuite).PublicDirectory).
		Step(`^the configuration source should be the configuration file$`, (*configurationSuite).Source).
		Step(`^configuration is saved to the gob cache$`, (*configurationSuite).Save).
		Step(`^configuration is read directly from the gob cache$`, (*configurationSuite).Read).
		Step(`^every configuration field should match the saved configuration$`, (*configurationSuite).RoundTrip).
		Step(`^the gob cache is removed$`, (*configurationSuite).Remove).
		Step(`^the gob cache file should (exist|not exist)$`, (*configurationSuite).CacheExists).
		Step(`^configuration loading should fail$`, (*configurationSuite).Failure).
		Step(`^configuration lookup paths are resolved$`, (*configurationSuite).Lookup).
		Step(`^the working directory should be searched first$`, (*configurationSuite).WorkingFirst).
		Step(`^the user configuration directory should be searched$`, (*configurationSuite).UserLookup).
		Step(`^the default cache directory is resolved$`, (*configurationSuite).CacheDirectory).
		Step(`^it should be the application directory under the OS user cache$`, (*configurationSuite).UserCache).
		Run()
}

type configurationSuite struct {
	gocuke.TestingT
	cfg, original Config
	err           error
	paths         []string
	cache         string
}

func (s *configurationSuite) Before() {
	s.cfg = Config{}
	s.original = Config{}
	s.err = nil
	s.paths = nil
	s.cache = ""
	if err := RemoveGobFile(); err != nil {
		s.Fatal(err)
	}
	os.Remove("config.json")
}
func (s *configurationSuite) File(content string) {
	s.Helper()
	if err := os.WriteFile("config.json", []byte(content), 0600); err != nil {
		s.Fatal(err)
	}
}
func (s *configurationSuite) Custom() {
	s.cfg = Config{Port: "9090", PublicDir: "custom-public", Secret: "test-secret", RateLimit: RateLimit{Limit: 100, Ttl: 30 * time.Second}, Image: Image{Path: "/custom-img/", Directory: "custom-assets/img", CacheDir: "custom-cache", AvifThroughVips: true, Formats: []string{"jpeg", "png"}, Presets: map[string]utils.ImagePreset{"custom": {Width: 800, Height: 600, Fit: "contain", Rotate: 90, Flip: "horizontal", Crop: "center", Brightness: 10, Contrast: -25, Gamma: 1.5, Filters: []string{"grayscale", "blur"}}}}, UsedConfigFile: "saved-config.json"}
	s.original = s.cfg
}
func (s *configurationSuite) Load() { s.err = s.cfg.Load() }
func (s *configurationSuite) Defaults() {
	s.Helper()
	want := Config{Port: "8080", PublicDir: "public", RateLimit: RateLimit{Ttl: time.Minute}, Image: Image{Formats: []string{"avif", "jpeg", "jpg", "png", "tiff", "webp"}, Presets: map[string]utils.ImagePreset{}, Path: "/img/", Directory: "assets/img", CacheDir: filepath.Join(defaultCacheDir(), "img")}, UsedConfigFile: "none (defaults only)"}
	if s.err != nil || !reflect.DeepEqual(s.cfg, want) {
		s.Fatalf("default config mismatch: %+v (error %v)", s.cfg, s.err)
	}
}
func (s *configurationSuite) Origin(origin string) {
	s.Helper()
	if s.err != nil {
		s.Fatal(s.err)
	}
	if s.cfg.LoadedFromGob != (origin == "gob") {
		s.Fatalf("expected %s configuration", origin)
	}
}
func (s *configurationSuite) Port(port string) {
	s.Helper()
	if s.err != nil || s.cfg.Port != port {
		s.Fatalf("expected port %s, got %s (%v)", port, s.cfg.Port, s.err)
	}
}
func (s *configurationSuite) PublicDirectory(dir string) {
	s.Helper()
	if s.cfg.PublicDir != dir {
		s.Fatalf("expected public dir %s, got %s", dir, s.cfg.PublicDir)
	}
}
func (s *configurationSuite) Source() {
	s.Helper()
	if filepath.Base(s.cfg.UsedConfigFile) != "config.json" {
		s.Fatalf("unexpected config source %q", s.cfg.UsedConfigFile)
	}
}
func (s *configurationSuite) Save() {
	s.Helper()
	s.original = s.cfg
	if err := s.cfg.saveGob(); err != nil {
		s.Fatal(err)
	}
}
func (s *configurationSuite) Read() { s.cfg = Config{}; s.err = s.cfg.loadGob() }
func (s *configurationSuite) RoundTrip() {
	s.Helper()
	actual := s.cfg
	actual.LoadedFromGob = s.original.LoadedFromGob
	if s.err != nil || !reflect.DeepEqual(actual, s.original) {
		s.Fatalf("gob round trip changed config: %+v (%v)", s.cfg, s.err)
	}
}
func (s *configurationSuite) Remove() {
	s.Helper()
	if err := RemoveGobFile(); err != nil {
		s.Fatal(err)
	}
}
func (s *configurationSuite) CacheExists(expected string) {
	s.Helper()
	_, err := os.Stat(GobFilePath())
	if (err == nil) != (expected == "exist") {
		s.Fatalf("cache should %s: %v", expected, err)
	}
}
func (s *configurationSuite) Failure() {
	s.Helper()
	if s.err == nil {
		s.Fatal("expected configuration error")
	}
}
func (s *configurationSuite) Lookup() { s.paths = searchConfigPaths() }
func (s *configurationSuite) WorkingFirst() {
	s.Helper()
	if len(s.paths) < 2 || s.paths[0] != "." {
		s.Fatalf("wrong config search order %v", s.paths)
	}
}
func (s *configurationSuite) UserLookup() {
	s.Helper()
	dir, err := os.UserConfigDir()
	if err != nil {
		s.Fatal(err)
	}
	if !slices.Contains(s.paths, filepath.Join(dir, appDirName)) {
		s.Fatalf("user config directory missing: %v", s.paths)
	}
}
func (s *configurationSuite) CacheDirectory() { s.cache = defaultCacheDir() }
func (s *configurationSuite) UserCache() {
	s.Helper()
	dir, err := os.UserCacheDir()
	if err != nil {
		s.Fatal(err)
	}
	if s.cache != filepath.Join(dir, appDirName) {
		s.Fatalf("wrong user cache location %q", s.cache)
	}
}
