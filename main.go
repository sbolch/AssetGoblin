// AssetGoblin serves static files, transforms images, and loads runtime plugins.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"assetgoblin/config"
)

func main() {
	pluginsDir := flag.String("plugins-dir", "", "Load Python plugins from this directory (default: beside executable)")
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
		serve(*pluginsDir)
		return
	} else if *printConfigFlag {
		printConfig()
		return
	} else if *clearGobFlag {
		if err := config.RemoveGobFile(); err != nil {
			slog.Error("Failed to delete gob config cache", "error", err)
			os.Exit(1)
		}
		printSuccess("Deleted gob config cache (if it existed): " + config.GobFilePath())
		return
	} else if *versionFlag {
		printVersionInfo(os.Stdout)
		latest, err := getLatestRelease()
		if err == nil && updateAvailable(latest) {
			printUpdateNotice("v" + latest.Version())
		}
		return
	} else if *updateFlag {
		if err := update(); err != nil {
			slog.Error("Failed to update", "error", err)
			os.Exit(1)
		}
		return
	}

	printBranding()

	latest, err := getLatestRelease()
	if err == nil && updateAvailable(latest) {
		fmt.Println()
		printUpdateNotice("v" + latest.Version())
	}

	fmt.Println()
	printUsage(os.Stdout)
	fmt.Println()
	printFooter()
}
