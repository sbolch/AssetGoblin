package middleware

import (
	"assetgoblin/config"
	"github.com/regen-network/gocuke"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestMiddlewareFeatures(t *testing.T) {
	gocuke.NewRunner(t, &clientSuite{}).Path("../features/middleware/*.feature").After((*clientSuite).After).Step(`^client address "([^"]*)" is normalized$`, (*clientSuite).Address).Step(`^its IP should be "([^"]*)"$`, (*clientSuite).IP).Step(`^a rate limiter allowing (\d+) requests per minute$`, (*clientSuite).Limiter).Step(`^two requests start slow downstream handlers$`, (*clientSuite).Start).Step(`^both handlers should run concurrently$`, (*clientSuite).Concurrent).Run()
}

type clientSuite struct {
	gocuke.TestingT
	ip               string
	limiter          *RateLimit
	entered, release chan struct{}
	wg               sync.WaitGroup
}

func (s *clientSuite) Address(address string) { s.ip = extractIP(address) }
func (s *clientSuite) IP(ip string) {
	s.Helper()
	if s.ip != ip {
		s.Fatalf("expected IP %q, got %q", ip, s.ip)
	}
}
func (s *clientSuite) Limiter(limit int64) {
	s.limiter = NewRateLimit(&config.RateLimit{Limit: int(limit), Ttl: time.Minute})
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
	if s.release != nil {
		close(s.release)
		s.wg.Wait()
		s.release = nil
	}
}
