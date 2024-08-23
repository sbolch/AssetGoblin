package main

import (
	"asset-manager/config"
	"asset-manager/image"
	"log"
	"net/http"
	"os"
)

var conf config.Config

func init() {
	var err error
	conf, err = config.Load()
	switch {
	case err != nil:
		log.Fatalf("Failed to load config: %v", err)
	case conf.Port == "":
		log.Fatalf("Invalid port: %v", conf.Port)
	}
}

func main() {
	wd, _ := os.Getwd()

	if len(conf.Image.Formats) > 0 && len(conf.Image.Presets) > 0 {
		http.HandleFunc("/img/", image.Serve)
	} else {
		log.Println("Warning: images are served as static files due to missing config.")
	}

	http.Handle("/", http.FileServer(http.Dir(wd+"/public")))

	if err := http.ListenAndServe(":"+conf.Port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
