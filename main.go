package main

import (
	"assetgoblin/config"
	"assetgoblin/image"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

var conf config.Config

func main() {
	versionFlag := flag.Bool("version", false, "Print version info")
	updateFlag := flag.Bool("update", false, "Update to latest version")
	flag.Parse()

	if *versionFlag {
		fmt.Println("Version:", Version)
		latest, _ := getLatestVersion()
		if latest != Version {
			fmt.Println("Update available:", latest)
		}
		os.Exit(0)
	} else if *updateFlag {
		update()
	}

	run()
}

func run() {
	if err := conf.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if conf.Port == "" {
		log.Fatalf("Invalid port: %v", conf.Port)
	}

	wd, _ := os.Getwd()

	if len(conf.Image.Formats) > 0 && len(conf.Image.Presets) > 0 {
		imageService := image.Service{Config: conf.Image}
		http.HandleFunc(conf.Image.Path, imageService.Serve)
	} else {
		log.Println("Warning: images are served as static files due to missing config.")
	}

	http.Handle("/", http.FileServer(http.Dir(wd+"/"+conf.PublicDir)))

	if err := http.ListenAndServe(":"+conf.Port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
