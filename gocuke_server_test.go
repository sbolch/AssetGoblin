package main

import (
	"assetgoblin/config"
	"context"
	"github.com/go-fuego/fuego"
	"net"
	"time"
)

func (s *assetgoblinSuite) ServerRateLimit(limit int64) {
	s.config.RateLimit = config.RateLimit{Limit: int(limit), Ttl: time.Minute}
}
func (s *assetgoblinSuite) StartProductionServer() {
	s.Helper()
	server, err := s.featureServer()
	if err != nil {
		s.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.Fatal(err)
	}
	fuego.WithListener(listener)(server)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.RunContext(ctx) }()
	s.liveURL = "http://" + listener.Addr().String()
	s.liveStop = func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				s.Error(err)
			}
		case <-time.After(10 * time.Second):
			s.Error("server did not stop")
		}
	}
}
func (s *assetgoblinSuite) StopProductionServer() {
	s.Helper()
	if s.liveStop == nil {
		s.Fatal("no running production server")
	}
	s.liveStop()
	s.liveStop = nil
	s.liveURL = ""
}
