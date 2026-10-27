package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	stdimage "image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/regen-network/gocuke"

	"assetgoblin/config"
	"assetgoblin/utils"
)

func TestImageFeatures(t *testing.T) {
	gocuke.NewRunner(t, &imageSuite{}).Path("../features/image/*.feature").Before((*imageSuite).Before).After((*imageSuite).After).
		Step(`^preset "([^"]*)" has settings (.*)$`, (*imageSuite).Preset).
		Step(`^size "([^"]*)" is selected with fit "([^"]*)"$`, (*imageSuite).Size).
		Step(`^the resize geometry should be "([^"]*)" with sizing "([^"]*)"$`, (*imageSuite).Geometry).
		Step(`^the selected fit should be "([^"]*)"$`, (*imageSuite).Fit).
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
		Step(`^a four-color image source$`, (*imageSuite).ColorSource).
		Step(`^image URL "([^"]*)" is requested$`, (*imageSuite).Request).
		Step(`^image URL "([^"]*)" is requested after cancellation$`, (*imageSuite).CanceledRequest).
		Step(`^the image response status should be (\d+)$`, (*imageSuite).Status).
		Step(`^the image should be (\d+) by (\d+) with first pixel "([^"]*)"$`, (*imageSuite).Pixels).
		Run()
}

type imageSuite struct {
	gocuke.TestingT
	dir                          string
	service                      Service
	presets                      map[string]utils.ImagePreset
	geometry                     string
	parts                        utils.ImagePreset
	supported, found             bool
	filename, resolved, absolute string
	response                     *httptest.ResponseRecorder
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
	s.geometry, s.parts = parseSize(input, fit, s.presets)
}
func (s *imageSuite) Geometry(geometry, sized string) {
	s.Helper()
	if s.geometry != geometry || (s.parts.Width > 0) != (sized == "true") {
		s.Fatalf("expected geometry %q and sizing %s, got %q %+v", geometry, sized, s.geometry, s.parts)
	}
}
func (s *imageSuite) Fit(fit string) {
	s.Helper()
	if s.parts.Fit != fit {
		s.Fatalf("expected fit %q, got %q", fit, s.parts.Fit)
	}
}
func (s *imageSuite) Options(data string) {
	s.Helper()
	var expected utils.ImagePreset
	if err := json.Unmarshal([]byte(data), &expected); err != nil {
		s.Fatal(err)
	}
	actual := s.parts
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

func (s *imageSuite) ColorSource() {
	s.Helper()
	tool := "convert"
	if runtime.GOOS == "windows" {
		tool = "magick"
	}
	if _, err := exec.LookPath(tool); err != nil {
		s.Skip(tool + " unavailable")
	}
	s.service.Config = &config.Image{Directory: s.dir, CacheDir: filepath.Join(s.dir, "cache"), Path: "/img/", Formats: []string{"png"}}
	source := stdimage.NewRGBA(stdimage.Rect(0, 0, 100, 80))
	colors := []color.RGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {255, 255, 0, 255}}
	for y := range 80 {
		for x := range 100 {
			source.SetRGBA(x, y, colors[(y/40)*2+x/50])
		}
	}
	file, err := os.Create(filepath.Join(s.dir, "colors.png"))
	if err != nil {
		s.Fatal(err)
	}
	encodeErr, closeErr := png.Encode(file, source), file.Close()
	if encodeErr != nil {
		s.Fatal(encodeErr)
	}
	if closeErr != nil {
		s.Fatal(closeErr)
	}
}
func (s *imageSuite) Request(path string) {
	s.response = httptest.NewRecorder()
	s.service.Serve(s.response, httptest.NewRequest(http.MethodGet, path, nil))
}
func (s *imageSuite) Pixels(width, height int64, pixel string) {
	s.Helper()
	if s.response.Code != http.StatusOK {
		s.Fatalf("status %d: %s", s.response.Code, s.response.Body.String())
	}
	img, err := png.Decode(bytes.NewReader(s.response.Body.Bytes()))
	if err != nil {
		s.Fatal(err)
	}
	if img.Bounds().Dx() != int(width) || img.Bounds().Dy() != int(height) {
		s.Fatalf("unexpected dimensions %v", img.Bounds())
	}
	r, g, b, _ := img.At(0, 0).RGBA()
	if pixel == "gray" {
		for y := range img.Bounds().Dy() {
			for x := range img.Bounds().Dx() {
				r, g, b, _ := img.At(x, y).RGBA()
				if r != g || g != b {
					s.Fatalf("pixel at %d,%d is colored", x, y)
				}
			}
		}
		return
	}
	actual := fmt.Sprintf("%d,%d,%d", r>>8, g>>8, b>>8)
	if pixel != "" && actual != pixel {
		s.Fatalf("first pixel %s, want %s", actual, pixel)
	}
}

func (s *imageSuite) CanceledRequest(path string) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.response = httptest.NewRecorder()
	s.service.Serve(s.response, httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil))
}
func (s *imageSuite) Status(status int64) {
	s.Helper()
	if s.response.Code != int(status) {
		s.Fatalf("status %d, want %d: %s", s.response.Code, status, s.response.Body.String())
	}
}
