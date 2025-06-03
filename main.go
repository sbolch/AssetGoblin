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

var conf config.Config

func main() {
	serveFlag := flag.Bool("serve", false, "Run the server")
	versionFlag := flag.Bool("version", false, "Print version info")
	updateFlag := flag.Bool("update", false, "Update to latest version")
	flag.Parse()

	if *serveFlag {
		serve()
	} else if *versionFlag {
		fmt.Print(Version)
		latest, _ := getLatestVersion()
		if latest != Version {
			fmt.Print(" (Update available: ", latest, ")")
		}
		fmt.Print("\n")
		os.Exit(0)
	} else if *updateFlag {
		update()
	}

	fmt.Println(Logo, "\nVersion:", Version)

	fmt.Println("\nUsage:")
	flag.PrintDefaults()

	fmt.Println("\nHomepage: https://github.com/sbolch/AssetGoblin")

	os.Exit(0)
}

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
