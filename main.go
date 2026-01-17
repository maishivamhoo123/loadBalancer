package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type Server struct {
	Name              string
	URL               string
	Weight            int
	ReverseProxy      *httputil.ReverseProxy
	Health            bool
	ActiveConnections int
	mux               sync.RWMutex
	Index             int
}

func newServer(name, urlstr string) *Server {
	u, _ := url.Parse(urlstr)
	rp := httputil.NewSingleHostReverseProxy(u)
	return &Server{
		Name:         name,
		URL:          urlstr,
		ReverseProxy: rp,
		Health:       true,
		Index:        -1,
	}
}

func (s *Server) CheckHealth() bool {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.Health
}

func (s *Server) SetHealth(alive bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.Health = alive
}

func (s *Server) GetActive() int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.ActiveConnections
}

func (s *Server) Ping() bool {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Head(s.URL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
