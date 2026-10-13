package main

import (
	"assetgoblin/config"
	"assetgoblin/image"
	"assetgoblin/middleware"
	"assetgoblin/plugin"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-fuego/fuego"
)

// newServer builds the production server without opening a listener.
func newServer(cfg *config.Config, pluginHandlers ...plugin.RouteHandler) (*fuego.Server, error) {
	return newServerWithCodecs(cfg, nil, pluginHandlers...)
}

func newServerWithCodecs(cfg *config.Config, codecs map[string]image.Codec, pluginHandlers ...plugin.RouteHandler) (*fuego.Server, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("working directory: %w", err)
	}
	srv := fuego.NewServer(
		fuego.WithAddr(":"+cfg.Port),
		fuego.WithoutLogger(),
		fuego.WithoutStartupMessages(),
		fuego.WithEngineOptions(fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{Disabled: true, DisableLocalSave: true})),
	)
	srv.ReadTimeout = 15 * time.Second
	srv.ReadHeaderTimeout = 5 * time.Second
	srv.WriteTimeout = 15 * time.Second
	srv.IdleTimeout = 60 * time.Second
	paths := map[string]bool{"/": true}
	register := func(path string, handler http.HandlerFunc) error {
		decoded, err := url.PathUnescape(path)
		if err != nil || decoded != path || pathpkg.Clean(path)+"/" != path || !strings.HasPrefix(path, "/") || !strings.HasSuffix(path, "/") || strings.ContainsAny(path, "{} \t\n") || paths[path] {
			return fmt.Errorf("invalid or duplicate route path %q", path)
		}
		paths[path] = true
		fuego.GetStd(srv, path, handler, fuego.OptionHide())
		return nil
	}
	if len(cfg.Image.Formats) > 0 {
		service := &image.Service{Config: &cfg.Image, Codecs: codecs}
		if err := register(cfg.Image.Path, service.Serve); err != nil {
			return nil, err
		}
	}
	for _, plugin := range pluginHandlers {
		if err := register(plugin.Path(), plugin.Handler()); err != nil {
			return nil, err
		}
	}
	publicDir := cfg.PublicDir
	if !filepath.IsAbs(publicDir) {
		publicDir = filepath.Join(wd, publicDir)
	}
	fuego.GetStd(srv, "/", http.FileServer(http.Dir(publicDir)).ServeHTTP, fuego.OptionHide())
	var middlewares []func(http.Handler) http.Handler
	if cfg.Secret != "" {
		middlewares = append(middlewares, (&middleware.Signkey{Secret: cfg.Secret}).Verify)
	}
	if cfg.RateLimit.Limit > 0 {
		middlewares = append(middlewares, middleware.NewRateLimit(&cfg.RateLimit).Limit)
	}
	fuego.WithGlobalMiddlewares(middlewares...)(srv)
	// Apply Fuego's global middleware order to the handler used by tests too.
	srv.Handler = srv.Mux
	for _, wrap := range middlewares {
		srv.Handler = wrap(srv.Handler)
	}

	return srv, nil
}

// serve loads configuration and starts the Fuego HTTP server.
func serve() {
	if err := conf.Load(); err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	if conf.Port == "" {
		slog.Error("Invalid port", "port", conf.Port)
		os.Exit(1)
	}
	manager, err := loadPlugins(pluginsDir)
	if err != nil {
		slog.Error("Failed to load plugins", "error", err)
		os.Exit(1)
	}
	defer manager.Close()
	srv, err := newServerWithCodecs(&conf, manager.ImageCodecs(), manager.Handlers()...)
	if err != nil {
		slog.Error("Failed to configure server", "error", err)
		os.Exit(1)
	}
	routes := make([]string, 0, len(manager.Handlers()))
	for _, handler := range manager.Handlers() {
		routes = append(routes, handler.Path())
	}
	codecs := make([]string, 0, len(manager.ImageCodecs()))
	for name := range manager.ImageCodecs() {
		codecs = append(codecs, name)
	}
	printServerSummary(os.Stdout, &conf, srv.Addr, routes, codecs)
	slog.Info("Starting server", "port", conf.Port)
	// Fuego applies the same middleware chain when starting the listener.
	if err := srv.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
