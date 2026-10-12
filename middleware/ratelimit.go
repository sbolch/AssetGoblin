// Package middleware provides HTTP middleware components for the application.
// These middleware components can be used to add functionality like authentication,
// rate limiting, and other request processing to the HTTP server.
package middleware

import (
	"assetgoblin/config"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimit is a middleware that limits the number of requests from a single IP address.
// It uses a map to track requests and their timestamps, with a mutex for thread safety.
type RateLimit struct {
	sync.Mutex
	Config      *config.RateLimit
	requests    map[string]*requestCounter
	nextCleanup time.Time
}

// requestCounter tracks the number of requests and the timestamp of the last request
// for a specific IP address.
type requestCounter struct {
	count       int
	lastRequest time.Time
}

// NewRateLimit creates a new RateLimit middleware with the given configuration.
// Expired entries are pruned periodically on requests, without a background goroutine.
func NewRateLimit(cfg *config.RateLimit) *RateLimit {
	r := &RateLimit{
		Config:   cfg,
		requests: make(map[string]*requestCounter),
	}

	return r
}

// Limit returns a middleware handler that limits the number of requests from a single IP address.
// It tracks requests by IP address and returns a 429 Too Many Requests response if the
// limit is exceeded within the configured time window.
func (r *RateLimit) Limit(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ip := extractIP(req.RemoteAddr)

		r.Lock()
		now := time.Now()
		if !now.Before(r.nextCleanup) {
			for ip, client := range r.requests {
				if now.Sub(client.lastRequest) > r.Config.Ttl {
					delete(r.requests, ip)
				}
			}
			r.nextCleanup = now.Add(5 * time.Minute)
		}

		if client, found := r.requests[ip]; found {
			if now.Sub(client.lastRequest) < r.Config.Ttl {
				if client.count >= r.Config.Limit {
					r.Unlock()
					http.Error(res, "Rate limit exceeded", http.StatusTooManyRequests)
					return
				}
				client.count++
			} else {
				client.count = 1
			}
			client.lastRequest = now
		} else {
			r.requests[ip] = &requestCounter{count: 1, lastRequest: now}
		}

		r.Unlock()
		handler.ServeHTTP(res, req)
	})
}

// extractIP extracts the IP address from a RemoteAddr string.
// It handles both IPv4 and IPv6 formats with port numbers.
func extractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host
	}
	return addr
}
