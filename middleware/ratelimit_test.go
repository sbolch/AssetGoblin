package middleware

import (
	"assetgoblin/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimit_Limit(t *testing.T) {
	tests := []struct {
		name         string
		ip           string
		intervals    []time.Duration
		expectedCode int
	}{
		{name: "within limit", ip: "192.168.0.1", intervals: []time.Duration{0, 50 * time.Millisecond, 100 * time.Millisecond}, expectedCode: http.StatusOK},
		{name: "exceed limit", ip: "192.168.0.2", intervals: []time.Duration{0, 50 * time.Millisecond, 50 * time.Millisecond}, expectedCode: http.StatusTooManyRequests},
		{name: "reset limit", ip: "192.168.0.3", intervals: []time.Duration{0, 100 * time.Millisecond, 50 * time.Millisecond}, expectedCode: http.StatusOK},
	}

	ratelimiter := NewRateLimit(&config.RateLimit{Limit: 2, Ttl: 100 * time.Millisecond})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratelimiter.requests = make(map[string]*requestCounter)
			handler := ratelimiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.ip

			var res *httptest.ResponseRecorder
			for i, interval := range tt.intervals {
				time.Sleep(interval)
				res = httptest.NewRecorder()
				handler.ServeHTTP(res, req)
				if i == len(tt.intervals)-1 && res.Code != tt.expectedCode {
					t.Errorf("expected status %v, got %v", tt.expectedCode, res.Code)
				}
			}
		})
	}
}
