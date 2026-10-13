// Package main implements a web server for serving and transforming static assets,
// particularly images, with support for different formats and presets.
package main

import (
	"assetgoblin/config"
	"flag"
	"fmt"
	"log/slog"
	"os"
)

// conf holds the application configuration loaded from config files.
var conf config.Config

var pluginsDir string

// main is the entry point of the application. It parses command-line flags
// and executes the appropriate action based on the provided flags.
func main() {
	flag.StringVar(&pluginsDir, "plugins-dir", "", "Load Lua plugins from this directory (default: beside executable)")
	serveFlag := flag.Bool("serve", false, "Run the server")
	flag.BoolVar(serveFlag, "S", false, "Run the server (shorthand)")
	clearGobFlag := flag.Bool("clear-gob", false, "Delete the cached gob config file")
	printConfigFlag := flag.Bool("config", false, "Print effective config and config source info")
	versionFlag := flag.Bool("version", false, "Print version info")
	flag.BoolVar(versionFlag, "v", false, "Print version info (shorthand)")
	updateFlag := flag.Bool("update", false, "Update to latest version")
	flag.CommandLine.Usage = func() { printUsage(flag.CommandLine.Output()) }
	flag.Parse()

	if *serveFlag {
		serve()
	} else if *printConfigFlag {
		printConfig()
		os.Exit(0)
	} else if *clearGobFlag {
		if err := config.RemoveGobFile(); err != nil {
			slog.Error("Failed to delete gob config cache", "error", err)
			os.Exit(1)
		}
		printSuccess("Deleted gob config cache (if it existed): " + config.GobFilePath())
		os.Exit(0)
	} else if *versionFlag {
		printVersionInfo(os.Stdout)
		latest, err := getLatestVersion()
		if err == nil && latest != Version {
			printUpdateNotice(latest)
		}
		os.Exit(0)
	} else if *updateFlag {
		update()
	}

	printBranding()

	latest, err := getLatestVersion()
	if err == nil && latest != Version {
		fmt.Println()
		printUpdateNotice(latest)
	}

	fmt.Println()
	printUsage(os.Stdout)
	fmt.Println()
	printFooter()

	os.Exit(0)
}
