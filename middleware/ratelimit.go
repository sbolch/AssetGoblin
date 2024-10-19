package middleware

import (
	"assetgoblin/config"
	"net/http"
	"sync"
	"time"
)

type RateLimit struct {
	sync.Mutex
	Config    *config.RateLimit
	requests  map[string]int
	timestamp map[string]time.Time
}

func NewRateLimit(config *config.RateLimit) *RateLimit {
	r := &RateLimit{
		Config:    config,
		requests:  make(map[string]int),
		timestamp: make(map[string]time.Time),
	}

	go r.cleanup(5 * time.Minute)

	return r
}

func (r *RateLimit) Limit(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ip := req.RemoteAddr

		r.Lock()
		defer r.Unlock()

		if count, ok := r.requests[ip]; ok {
			if time.Since(r.timestamp[ip]) < r.Config.Ttl {
				if count >= r.Config.Limit {
					http.Error(res, "Rate limit exceeded", http.StatusTooManyRequests)
					return
				}
				r.requests[ip]++
			} else {
				r.requests[ip] = 1
				r.timestamp[ip] = time.Now()
			}
		} else {
			r.requests[ip] = 1
			r.timestamp[ip] = time.Now()
		}

		handler.ServeHTTP(res, req)
	})
}

func (r *RateLimit) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		r.Lock()
		for ip, timestamp := range r.timestamp {
			if time.Since(timestamp) > r.Config.Ttl {
				delete(r.requests, ip)
				delete(r.timestamp, ip)
			}
		}
		r.Unlock()
	}
}
