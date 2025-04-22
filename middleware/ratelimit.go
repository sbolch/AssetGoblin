package middleware

import (
	"assetgoblin/config"
	"net/http"
	"sync"
	"time"
)

type RateLimit struct {
	sync.Mutex
	Config   *config.RateLimit
	requests map[string]*requestCounter
}

type requestCounter struct {
	count       int
	lastRequest time.Time
}

func NewRateLimit(config *config.RateLimit) *RateLimit {
	r := &RateLimit{
		Config:   config,
		requests: make(map[string]*requestCounter),
	}

	go r.cleanup(5 * time.Minute)

	return r
}

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
