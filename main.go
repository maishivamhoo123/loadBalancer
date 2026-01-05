package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type Server struct {
	Name              string `json:"name"`
	URL               string `json:"url"`
	ReverseProxy      *httputil.ReverseProxy
	Health            bool
	ActiveConnections int
	mux               sync.Mutex
}

func newServer(name, urlstr string) *Server {
	u, _ := url.Parse(urlstr)
	rp := httputil.NewSingleHostReverseProxy(u)
	return &Server{
		Name:              name,
		URL:               urlstr,
		ReverseProxy:      rp,
		Health:            true,
		ActiveConnections: 0,
	}
}

func (s *Server) CheckHealth() bool {
	resp, err := http.Head(s.URL)
	if err != nil {
		s.Health = false
		return s.Health
	}
	if resp.StatusCode != http.StatusOK {
		s.Health = false
		return s.Health
	}
	s.Health = true
	return s.Health
}

// IncrementActive safely increases the connection count
func (s *Server) IncrementActive() {
	s.mux.Lock()
	s.ActiveConnections++
	s.mux.Unlock()
}

// DecrementActive safely decreases the connection count
func (s *Server) DecrementActive() {
	s.mux.Lock()
	s.ActiveConnections--
	s.mux.Unlock()
}

// GetActive safely reads the connection count
func (s *Server) GetActive() int {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.ActiveConnections
}
