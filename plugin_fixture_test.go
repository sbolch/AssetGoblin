package main

import (
	"github.com/regen-network/gocuke"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func copyExample(t gocuke.TestingT, dir, name, sourceDir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("plugins", name, "assetgoblin-"+name+".py"))
	if err != nil {
		t.Fatal(err)
	}
	if sourceDir != "" {
		data = []byte(strings.Replace(string(data), `"directory": "assets/`+name+`"`, `"directory": `+strconv.Quote(filepath.ToSlash(sourceDir)), 1))
	}
	if err := os.WriteFile(filepath.Join(dir, "assetgoblin-"+name+".py"), data, 0600); err != nil {
		t.Fatal(err)
	}
}
