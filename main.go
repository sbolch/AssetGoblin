// Package main implements a web server for serving and transforming static assets,
// particularly images, with support for different formats and presets.
package main

import (
	"assetgoblin/config"
	"assetgoblin/image"
	"assetgoblin/middleware"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// conf holds the application configuration loaded from config files.
var conf config.Config

// main is the entry point of the application. It parses command-line flags
// and executes the appropriate action based on the provided flags.
func main() {
	serveFlag := flag.Bool("serve", false, "Run the server")
	versionFlag := flag.Bool("version", false, "Print version info")
	flag.BoolVar(versionFlag, "v", false, "Print version info (shorthand)")
	updateFlag := flag.Bool("update", false, "Update to latest version")
	flag.Parse()

	if *serveFlag {
		serve()
	} else if *versionFlag {
		fmt.Println(Version)
		fmt.Printf("Build: %s #%s\n", BuildTime, GitCommit)
		latest, _ := getLatestVersion()
		if latest != Version {
			fmt.Printf("\033[1;33mUpdate available: %s \033[0m\n", latest)
		}
		os.Exit(0)
	} else if *updateFlag {
		update()
	}

	fmt.Println(Logo, "\nServe static files or dynamically manipulated images with ease")

	latest, _ := getLatestVersion()
	if latest != Version {
		fmt.Println("\n\033[1;33m╔═════════════════════════════════╗\033[0m")
		fmt.Printf("\033[1;33m  Update available: %s \033[0m\n", latest)
		fmt.Println("\033[1;33m╚═════════════════════════════════╝\033[0m")
	}

	fmt.Println("\nUsage:")
	flag.PrintDefaults()

	fmt.Println("\nHomepage: https://github.com/sbolch/AssetGoblin")
	fmt.Println("Version: ", Version)
	fmt.Printf("Build: %s #%s\n", BuildTime, GitCommit)

	os.Exit(0)
}

// serve starts the HTTP server with the configured handlers and middleware.
// It loads the configuration, sets up routes for serving images and static files,
// and applies middleware for security and rate limiting if configured.
func serve() {
	if err := conf.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if conf.Port == "" {
		log.Fatalf("Invalid port: %v", conf.Port)
	}

	wd, _ := os.Getwd()

	mux := http.NewServeMux()

	if len(conf.Image.Formats) > 0 && len(conf.Image.Presets) > 0 {
		imageService := image.Service{Config: &conf.Image}
		mux.HandleFunc(conf.Image.Path, imageService.Serve)
	} else {
		log.Println("Warning: images are served as static files due to missing config.")
	}

	mux.Handle("/", http.FileServer(http.Dir(filepath.Join(wd, conf.PublicDir))))

	var handler http.Handler = mux

	if conf.Secret != "" {
		signkeyMiddleware := middleware.Signkey{Secret: conf.Secret}
		handler = signkeyMiddleware.Verify(handler)
	}

	if conf.RateLimit.Limit > 0 {
		ratelimitMiddleware := middleware.NewRateLimit(&conf.RateLimit)
		handler = ratelimitMiddleware.Limit(handler)
	}

	if err := http.ListenAndServe(":"+conf.Port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
