package config

import "testing"

// isolateConfigAndCacheEnv sets per-test config/cache environment directories.
func isolateConfigAndCacheEnv(t *testing.T) {
	t.Helper()

	tempHome := t.TempDir()
	tempConfig := t.TempDir()
	tempCache := t.TempDir()
	tempProgramData := t.TempDir()
	tempAppData := t.TempDir()
	tempLocalAppData := t.TempDir()

	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", tempConfig)
	t.Setenv("XDG_CACHE_HOME", tempCache)
	t.Setenv("ProgramData", tempProgramData)
	t.Setenv("APPDATA", tempAppData)
	t.Setenv("LOCALAPPDATA", tempLocalAppData)
}
