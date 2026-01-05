package main

import (
	"net/http" // <--- ADDED: Needed for http.Client
	"net/http/httputil"
	"net/url"
	"sync"
	"time" // <--- ADDED: Needed for time.Second
)

type Server struct {
	Name              string
	URL               string
	ReverseProxy      *httputil.ReverseProxy
	Health            bool
	ActiveConnections int
	mux               sync.RWMutex

	// The Index is required by the Heap to update priority in O(log n) time
	Index int
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
		Index:             -1,
	}
}

// CheckHealth just reads the current status (fast)
func (s *Server) CheckHealth() bool {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.Health
}

// SetHealth updates the status safely
func (s *Server) SetHealth(alive bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.Health = alive
}

// GetActive reads the connection count safely
func (s *Server) GetActive() int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.ActiveConnections
}

// Ping sends a HEAD request to check if the backend is actually alive
func (s *Server) Ping() bool {
	// 2 second timeout so we don't get stuck
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Head(s.URL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
