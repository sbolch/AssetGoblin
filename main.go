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

// main is the entry point of the application. It parses command-line flags
// and executes the appropriate action based on the provided flags.
func main() {
	serveFlag := flag.Bool("serve", false, "Run the server")
	clearGobFlag := flag.Bool("clear-gob", false, "Delete the cached gob config file")
	printConfigFlag := flag.Bool("config", false, "Print effective config and config source info")
	versionFlag := flag.Bool("version", false, "Print version info")
	flag.BoolVar(versionFlag, "v", false, "Print version info (shorthand)")
	updateFlag := flag.Bool("update", false, "Update to latest version")
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
		fmt.Printf("Deleted gob config cache (if it existed): %s\n", config.GobFilePath())
		os.Exit(0)
	} else if *versionFlag {
		fmt.Println(Version)
		fmt.Printf("Build: %s #%s\n", BuildTime, GitCommit)
		latest, _ := getLatestVersion()
		if latest != Version {
			fmt.Printf("\033[1;33mUpdate available: %s\033[0m\n", latest)
		}
		os.Exit(0)
	} else if *updateFlag {
		update()
	}

	fmt.Println(Logo, "\nServe static files or dynamically manipulated images with ease")

	latest, _ := getLatestVersion()
	if latest != Version {
		fmt.Println("\n\033[1;33m╔═════════════════════════════════╗\033[0m")
		fmt.Printf("\033[1;33m  Update available: %s\033[0m\n", latest)
		fmt.Println("\033[1;33m╚═════════════════════════════════╝\033[0m")
	}

	fmt.Println("\nUsage:")
	flag.PrintDefaults()

	fmt.Println("\nHomepage: https://github.com/sbolch/AssetGoblin")
	fmt.Println("Version: ", Version)
	fmt.Printf("Build: %s #%s\n", BuildTime, GitCommit)

	os.Exit(0)
}
