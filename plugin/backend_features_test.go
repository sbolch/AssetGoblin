package plugin

import (
	"context"
	"testing"

	"github.com/regen-network/gocuke"
)

// optionalBackend exercises the runtime boundary without depending on an interpreter.
type optionalBackend struct {
	extension     string
	invalid       bool
	loads, closes int
}

func (b *optionalBackend) Extension() string { return b.extension }
func (b *optionalBackend) Load(ctx context.Context, filename string, catalog map[string]string) (any, []string, string, func(), error) {
	b.loads++
	value := map[string]any{"api_version": float64(1), "routes": []any{map[string]any{"path": "/optional/", "handle": Callback(func(context.Context, map[string]any) (any, error) { return map[string]any{"body": "optional"}, nil })}}}
	if b.invalid {
		value["api_version"] = float64(2)
	}
	return value, nil, "test-revision", func() { b.closes++ }, nil
}

type backendSuite struct {
	gocuke.TestingT
	pluginSuite
	backend         *optionalBackend
	original        map[string]Backend
	registrationErr error
}

func TestBackendFeatures(t *testing.T) {
	gocuke.NewRunner(t, &backendSuite{}).Path("../features/backends/*.feature").
		Step(`^an optional runtime plugin$`, (*backendSuite).Plugin).
		Step(`^its runtime is registered$`, (*backendSuite).Register).
		Step(`^its runtime is registered again$`, (*backendSuite).RegisterAgain).
		Step(`^backend registration should fail$`, (*backendSuite).RegistrationFails).
		Step(`^a backend extension "([^"]*)"$`, (*backendSuite).Extension).
		Step(`^its runtime returns an invalid manifest$`, (*backendSuite).Invalid).
		Step(`^a Python plugin with the same name$`, (*backendSuite).Python).
		Step(`^the optional runtime was not loaded$`, (*backendSuite).NotLoaded).
		Step(`^the optional runtime is closed once$`, (*backendSuite).Closed).
		Step(`^plugin discovery runs$`, (*backendSuite).Load).
		Step(`^(\d+) routes should be exposed$`, (*backendSuite).RouteCount).
		Step(`^discovery should fail$`, (*backendSuite).LoadFails).
		Step(`^a plugin "([^"]*)" request is sent to "([^"]*)" with range "([^"]*)"$`, (*backendSuite).Request).
		Step(`^the plugin response body should be "([^"]*)"$`, (*backendSuite).Body).
		Step(`^the manager is closed twice$`, (*backendSuite).CloseTwice).
		Run()
}
func (s *backendSuite) Before() {
	s.pluginSuite.TestingT = s.TestingT
	s.pluginSuite.Before()
	s.backend = &optionalBackend{extension: ".optional"}
	s.registrationErr = nil
	backendRegistry.Lock()
	s.original = backendRegistry.items
	backendRegistry.items = map[string]Backend{}
	backendRegistry.Unlock()
}
func (s *backendSuite) After() {
	s.pluginSuite.After()
	backendRegistry.Lock()
	backendRegistry.items = s.original
	backendRegistry.Unlock()
}
func (s *backendSuite) Plugin() { s.write("assetgoblin-custom.optional", "runtime input") }
func (s *backendSuite) Register() {
	s.Helper()
	if err := RegisterBackend(s.backend); err != nil {
		s.Fatal(err)
	}
}
func (s *backendSuite) RegisterAgain() { s.registrationErr = RegisterBackend(s.backend) }
func (s *backendSuite) RegistrationFails() {
	s.Helper()
	if s.registrationErr == nil {
		s.Fatal("invalid registration accepted")
	}
}
func (s *backendSuite) Extension(ext string) { s.backend.extension = ext }
func (s *backendSuite) Invalid()             { s.backend.invalid = true }
func (s *backendSuite) Python() {
	s.write("assetgoblin-custom.py", `plugin = {"api_version": 1, "routes": [{"path": "/python/", "directory": "."}]}`)
}
func (s *backendSuite) NotLoaded() {
	s.Helper()
	if s.backend.loads != 0 {
		s.Fatal("Python priority lost")
	}
}
func (s *backendSuite) Closed() {
	s.Helper()
	if s.backend.closes != 1 {
		s.Fatalf("closed %d times", s.backend.closes)
	}
}
