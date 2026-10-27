package middleware

import (
	"assetgoblin/config"
	"context"
	"github.com/regen-network/gocuke"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestMiddlewareFeatures(t *testing.T) {
	gocuke.NewRunner(t, &clientSuite{}).Path("../features/middleware/*.feature").After((*clientSuite).After).Step(`^client address "([^"]*)" is normalized$`, (*clientSuite).Address).Step(`^its IP should be "([^"]*)"$`, (*clientSuite).IP).Step(`^a rate limiter allowing (\d+) requests per minute$`, (*clientSuite).Limiter).Step(`^two requests start slow downstream handlers$`, (*clientSuite).Start).Step(`^both handlers should run concurrently$`, (*clientSuite).Concurrent).Step(`^a request from "([^"]*)" returns (\d+) with (\d+) remaining$`, (*clientSuite).Request).
		Step(`^the response includes rate limit and retry headers$`, (*clientSuite).Headers).
		Step(`^accepted requests keep the original window reset$`, (*clientSuite).FixedWindow).
		Step(`^the limiter is closed twice$`, (*clientSuite).Close).
		Step(`^the closed limiter rejects new requests$`, (*clientSuite).Stopped).
		Step(`^a limiter with limit (-?\d+) and TTL "([^"]*)" is rejected$`, (*clientSuite).Invalid).Run()
}

type clientSuite struct {
	gocuke.TestingT
	ip               string
	limiter          *RateLimit
	entered, release chan struct{}
	wg               sync.WaitGroup
	response         *httptest.ResponseRecorder
}

func (s *clientSuite) Address(address string) { s.ip = extractIP(address) }
func (s *clientSuite) IP(ip string) {
	s.Helper()
	if s.ip != ip {
		s.Fatalf("expected IP %q, got %q", ip, s.ip)
	}
}
func (s *clientSuite) Limiter(limit int64) {
	var err error
	s.limiter, err = NewRateLimit(&config.RateLimit{Limit: int(limit), Ttl: time.Minute})
	if err != nil {
		s.Fatal(err)
	}
}
func (s *clientSuite) Start() {
	s.entered = make(chan struct{}, 2)
	s.release = make(chan struct{})
	handler := s.limiter.Limit(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { s.entered <- struct{}{}; <-s.release }))
	for range 2 {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
		}()
	}
}
func (s *clientSuite) Concurrent() {
	s.Helper()
	for range 2 {
		select {
		case <-s.entered:
		case <-time.After(time.Second):
			s.Fatal("rate limiter serialized downstream handlers")
		}
	}
}
func (s *clientSuite) After() {
	defer func() {
		if s.limiter != nil {
			s.limiter.Close()
			s.limiter = nil
		}
	}()
	if s.release != nil {
		close(s.release)
		s.wg.Wait()
		s.release = nil
	}
}

func (s *clientSuite) Request(address string, status, remaining int64) {
	s.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = address
	// Untrusted forwarding headers must not select a separate bucket.
	req.Header.Set("X-Forwarded-For", "203.0.113."+strconv.FormatInt(remaining+1, 10))
	s.response = httptest.NewRecorder()
	s.limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })).ServeHTTP(s.response, req)
	if s.response.Code != int(status) {
		s.Fatalf("expected %d, got %d", status, s.response.Code)
	}
	if got := s.response.Header().Get("X-RateLimit-Remaining"); got != strconv.FormatInt(remaining, 10) {
		s.Fatalf("remaining %q, expected %d", got, remaining)
	}
}
func (s *clientSuite) Headers() {
	s.Helper()
	headers := s.response.Header()
	if headers.Get("X-RateLimit-Limit") != "2" || headers.Get("X-RateLimit-Reset") == "" || headers.Get("Retry-After") != headers.Get("X-RateLimit-Reset") {
		s.Fatalf("unexpected headers: %v", headers)
	}
	if _, err := http.ParseTime(headers.Get("Retry-After")); err != nil {
		s.Fatal(err)
	}
}
func (s *clientSuite) FixedWindow() {
	s.Helper()
	if s.limiter != nil {
		s.limiter.Close()
	}
	var err error
	s.limiter, err = NewRateLimit(&config.RateLimit{Limit: 2, Ttl: time.Second})
	if err != nil {
		s.Fatal(err)
	}
	_, _, reset, ok, err := s.limiter.store.Take(context.Background(), "192.0.2.1")
	if err != nil || !ok {
		s.Fatalf("initial request: %v", err)
	}
	boundary := time.Unix(0, int64(reset))
	time.Sleep(time.Until(boundary.Add(-500 * time.Millisecond)))
	s.Request("192.0.2.1:123", 200, 0)
	time.Sleep(time.Until(boundary.Add(20 * time.Millisecond)))
	s.Request("192.0.2.1:456", 200, 1)
}
func (s *clientSuite) Close() {
	s.Helper()
	for range 2 {
		if err := s.limiter.Close(); err != nil {
			s.Fatal(err)
		}
	}
}
func (s *clientSuite) Stopped() {
	s.Helper()
	_, _, _, _, err := s.limiter.store.Take(context.Background(), "192.0.2.1")
	if err == nil {
		s.Fatal("closed limiter accepted a request")
	}
}
func (s *clientSuite) Invalid(limit int64, ttl string) {
	s.Helper()
	duration, err := time.ParseDuration(ttl)
	if err != nil {
		s.Fatal(err)
	}
	limiter, err := NewRateLimit(&config.RateLimit{Limit: int(limit), Ttl: duration})
	if limiter != nil {
		limiter.Close()
	}
	if err == nil {
		s.Fatal("invalid rate limiter configuration accepted")
	}
}
