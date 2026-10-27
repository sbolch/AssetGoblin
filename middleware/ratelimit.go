// Package middleware provides HMAC authentication and per-IP rate limiting.
package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"assetgoblin/config"

	"github.com/sethvargo/go-limiter"
	"github.com/sethvargo/go-limiter/httplimit"
	"github.com/sethvargo/go-limiter/memorystore"
)

// RateLimit owns a fixed-window per-IP limiter and its expiration worker.
type RateLimit struct {
	store      limiter.Store
	middleware *httplimit.Middleware
}

func NewRateLimit(cfg *config.RateLimit) (*RateLimit, error) {
	if cfg == nil || cfg.Limit <= 0 || cfg.Ttl <= 0 {
		return nil, fmt.Errorf("rate limit and TTL must be positive")
	}
	store, err := memorystore.New(&memorystore.Config{
		Tokens: uint64(cfg.Limit), Interval: cfg.Ttl,
		SweepInterval: 5 * time.Minute, SweepMinTTL: cfg.Ttl,
	})
	if err != nil {
		return nil, err
	}
	middleware, err := httplimit.NewMiddleware(store, func(req *http.Request) (string, error) {
		return extractIP(req.RemoteAddr), nil
	})
	if err != nil {
		store.Close(context.Background())
		return nil, err
	}
	return &RateLimit{store: store, middleware: middleware}, nil
}

func (r *RateLimit) Limit(handler http.Handler) http.Handler { return r.middleware.Handle(handler) }

// Close releases client buckets and stops the expiration worker. It is safe to repeat.
func (r *RateLimit) Close() error { return r.store.Close(context.Background()) }

// extractIP accepts both socket addresses and bare IPv4 or IPv6 addresses.
func extractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host
	}
	return addr
}
