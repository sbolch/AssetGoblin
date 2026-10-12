package image

import (
	"assetgoblin/config"
	"assetgoblin/utils"
	"encoding/json"
	"github.com/regen-network/gocuke"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestImageFeatures(t *testing.T) {
	gocuke.NewRunner(t, &imageSuite{}).Path("../features/image/*.feature").Before((*imageSuite).Before).After((*imageSuite).After).
		Step(`^preset "([^"]*)" has settings (.*)$`, (*imageSuite).Preset).
		Step(`^size "([^"]*)" is selected with fit "([^"]*)"$`, (*imageSuite).Size).
		Step(`^the resize geometry should be "([^"]*)" with sizing "([^"]*)"$`, (*imageSuite).Geometry).
		Step(`^the selected fit should be "([^"]*)"$`, (*imageSuite).Fit).
		Step(`^the selection should be a preset "([^"]*)"$`, (*imageSuite).IsPreset).
		Step(`^the transformation settings should match (.*)$`, (*imageSuite).Options).
		Step(`^supported source formats "([^"]*)"$`, (*imageSuite).Formats).
		Step(`^a source (file|directory) "([^"]*)"$`, (*imageSuite).Source).
		Step(`^the source basename "([^"]*)" is searched$`, (*imageSuite).Find).
		Step(`^the discovered filename should be "([^"]*)"$`, (*imageSuite).Found).
		Step(`^output format "([^"]*)" is checked$`, (*imageSuite).CheckFormat).
		Step(`^the format should be supported "([^"]*)"$`, (*imageSuite).Supported).
		Step(`^image path "([^"]*)" is resolved$`, (*imageSuite).Resolve).
		Step(`^it should resolve to "([^"]*)" under the working directory$`, (*imageSuite).Resolved).
		Step(`^an absolute image path is resolved$`, (*imageSuite).Absolute).
		Step(`^it should retain its absolute location$`, (*imageSuite).Unchanged).
		Run()
}

type imageSuite struct {
	gocuke.TestingT
	dir                          string
	service                      Service
	presets                      map[string]utils.ImagePreset
	geometry                     string
	parts                        sizeParts
	preset, supported, found     bool
	filename, resolved, absolute string
}

func (s *imageSuite) Before() {
	var err error
	s.dir, err = os.MkdirTemp("", "assetgoblin-image-features-")
	if err != nil {
		s.Fatal(err)
	}
	s.presets = map[string]utils.ImagePreset{}
	s.service = Service{Config: &config.Image{}}
}
func (s *imageSuite) After() { os.RemoveAll(s.dir) }
func (s *imageSuite) Preset(name, data string) {
	s.Helper()
	var preset utils.ImagePreset
	if err := json.Unmarshal([]byte(data), &preset); err != nil {
		s.Fatal(err)
	}
	s.presets[name] = preset
}
func (s *imageSuite) Size(input, fit string) {
	s.geometry, s.parts, s.preset = parseSize(input, fit, s.presets)
}
func (s *imageSuite) Geometry(geometry, sized string) {
	s.Helper()
	if s.geometry != geometry || s.parts.hasSize != (sized == "true") {
		s.Fatalf("expected geometry %q and sizing %s, got %q %+v", geometry, sized, s.geometry, s.parts)
	}
}
func (s *imageSuite) Fit(fit string) {
	s.Helper()
	if string(s.parts.fit) != fit {
		s.Fatalf("expected fit %q, got %q", fit, s.parts.fit)
	}
}
func (s *imageSuite) IsPreset(preset string) {
	s.Helper()
	if s.preset != (preset == "true") {
		s.Fatalf("expected preset %s", preset)
	}
}
func (s *imageSuite) Options(data string) {
	s.Helper()
	var expected utils.ImagePreset
	if err := json.Unmarshal([]byte(data), &expected); err != nil {
		s.Fatal(err)
	}
	actual := utils.ImagePreset{Fit: string(s.parts.fit), Width: s.parts.width, Height: s.parts.height, Rotate: s.parts.rotate, Flip: s.parts.flip, Crop: s.parts.crop, Brightness: s.parts.brightness, Contrast: s.parts.contrast, Gamma: s.parts.gamma, Filters: s.parts.filters}
	if expected.Fit == "" {
		expected.Fit = "contain"
	}
	if !reflect.DeepEqual(actual, expected) {
		s.Fatalf("changed preset settings: %+v, want %+v", actual, expected)
	}
}
func (s *imageSuite) Formats(formats string) {
	s.service.Config.Formats = nil
	if formats != "" {
		s.service.Config.Formats = strings.Split(formats, ",")
	}
}
func (s *imageSuite) Source(kind, name string) {
	s.Helper()
	path := filepath.Join(s.dir, name)
	var err error
	if kind == "directory" {
		err = os.Mkdir(path, 0755)
	} else {
		err = os.WriteFile(path, nil, 0600)
	}
	if err != nil {
		s.Fatal(err)
	}
}
func (s *imageSuite) Find(base string) {
	s.filename, s.found = s.service.findImage(filepath.Join(s.dir, base))
}
func (s *imageSuite) Found(filename string) {
	s.Helper()
	if (filename != "") != s.found || (s.found && filepath.Base(s.filename) != filename) {
		s.Fatalf("expected filename %q, got %q (found %v)", filename, s.filename, s.found)
	}
}
func (s *imageSuite) CheckFormat(format string) { s.supported = s.service.isValidFormat(format) }
func (s *imageSuite) Supported(value string) {
	s.Helper()
	expected, err := strconv.ParseBool(value)
	if err != nil {
		s.Fatal(err)
	}
	if s.supported != expected {
		s.Fatalf("expected supported %v", expected)
	}
}
func (s *imageSuite) Resolve(path string) { s.resolved = ensureAbsolute(path, s.dir) }
func (s *imageSuite) Resolved(path string) {
	s.Helper()
	if s.resolved != filepath.Join(s.dir, filepath.FromSlash(path)) || !filepath.IsAbs(s.resolved) {
		s.Fatalf("wrong resolved path %q", s.resolved)
	}
}
func (s *imageSuite) Absolute() {
	s.absolute = filepath.Join(s.dir, "absolute", "path")
	s.resolved = ensureAbsolute(s.absolute, filepath.Join(s.dir, "working"))
}
func (s *imageSuite) Unchanged() {
	s.Helper()
	if s.resolved != s.absolute {
		s.Fatalf("absolute path changed to %q", s.resolved)
	}
}
