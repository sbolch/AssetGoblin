// Package middleware provides HTTP middleware components for the application.
// These middleware components can be used to add functionality like authentication,
// rate limiting, and other request processing to the HTTP server.
package middleware

import (
	"assetgoblin/config"
	"net/http"
	"sync"
	"time"
)

// RateLimit is a middleware that limits the number of requests from a single IP address.
// It uses a map to track requests and their timestamps, with a mutex for thread safety.
type RateLimit struct {
	sync.Mutex                            // Mutex for thread-safe access to the requests map
	Config     *config.RateLimit          // Configuration for rate limiting
	requests   map[string]*requestCounter // Map of IP addresses to request counters
}

// requestCounter tracks the number of requests and the timestamp of the last request
// for a specific IP address.
type requestCounter struct {
	count       int       // Number of requests within the time window
	lastRequest time.Time // Timestamp of the last request
}

// NewRateLimit creates a new RateLimit middleware with the given configuration.
// It initializes the requests map and starts a background goroutine to clean up
// expired entries from the map.
func NewRateLimit(config *config.RateLimit) *RateLimit {
	r := &RateLimit{
		Config:   config,
		requests: make(map[string]*requestCounter),
	}

	go r.cleanup(5 * time.Minute)

	return r
}

// Limit returns a middleware handler that limits the number of requests from a single IP address.
// It tracks requests by IP address and returns a 429 Too Many Requests response if the
// limit is exceeded within the configured time window.
func (r *RateLimit) Limit(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ip := req.RemoteAddr

		r.Lock()
		defer r.Unlock()

		if client, found := r.requests[ip]; found {
			if time.Since(client.lastRequest) < r.Config.Ttl {
				if client.count >= r.Config.Limit {
					http.Error(res, "Rate limit exceeded", http.StatusTooManyRequests)
					return
				}
				client.count++
			} else {
				client.count = 1
			}
			client.lastRequest = time.Now()
		} else {
			r.requests[ip] = &requestCounter{count: 1, lastRequest: time.Now()}
		}

		handler.ServeHTTP(res, req)
	})
}

// cleanup runs periodically to remove expired entries from the requests map.
// It is started as a goroutine by NewRateLimit and runs every 'interval' duration.
// This prevents the map from growing indefinitely with inactive IP addresses.
func (r *RateLimit) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		r.Lock()
		for ip, client := range r.requests {
			if time.Since(client.lastRequest) > r.Config.Ttl {
				delete(r.requests, ip)
			}
		}
		r.Unlock()
	}
}
