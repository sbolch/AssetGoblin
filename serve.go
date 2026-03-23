package main

import (
	"assetgoblin/image"
	"assetgoblin/middleware"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// serve starts the HTTP server with the configured handlers and middleware.
// It loads the configuration, sets up routes for serving images and static files,
// and applies middleware for security and rate limiting if configured.
func serve() {
	if err := conf.Load(); err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if conf.Port == "" {
		slog.Error("Invalid port", "port", conf.Port)
		os.Exit(1)
	}

	wd, _ := os.Getwd()

	mux := http.NewServeMux()

	if len(conf.Image.Formats) > 0 && len(conf.Image.Presets) > 0 {
		imageService := image.Service{Config: &conf.Image}
		mux.HandleFunc(conf.Image.Path, imageService.Serve)
	} else {
		slog.Warn("Images are served as static files due to missing config")
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

	srv := &http.Server{
		Addr:         ":" + conf.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("Starting server", "port", conf.Port)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
