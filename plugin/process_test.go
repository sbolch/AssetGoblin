package plugin

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This helper acts as an arbitrary external tool, independent of media formats.
func TestPluginProcessHelper(t *testing.T) {
	if os.Getenv("ASSETGOBLIN_TEST_PROCESS_HELPER") != "1" {
		return
	}
	args := flag.Args()
	if len(args) != 3 {
		os.Exit(2)
	}
	switch args[0] {
	case "wait":
		time.Sleep(10 * time.Second)
	case "fail":
		os.WriteFile(args[2], []byte("partial"), 0600)
		os.Exit(3)
	case "upper":
		body, err := os.ReadFile(args[1])
		if err != nil {
			os.Exit(4)
		}
		if err := os.WriteFile(args[2], []byte(strings.ToUpper(string(body))), 0600); err != nil {
			os.Exit(5)
		}
	}
	os.Exit(0)
}

func processHelper(t *testing.T) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(executable)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ASSETGOBLIN_TEST_PROCESS_HELPER", "1")
	t.Setenv("GORACE", "atexit_sleep_ms=0")
	return filepath.Base(executable)
}

func helperPlan(tool, mode string) processPlan {
	return processPlan{Tool: tool, Extension: "txt", Args: []string{"-test.run=^TestPluginProcessHelper$", "--", mode, "{input}", "{output}"}}
}
