package main

import (
	"assetgoblin/image"
	"assetgoblin/middleware"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

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
