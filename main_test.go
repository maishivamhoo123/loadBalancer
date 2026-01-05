package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"testing"
)

// ==========================================
// TEST 1: The "Smart" Logic (Least Connections)
// ==========================================
func TestLeastConnections(t *testing.T) {
	// Setup: Server 1 is busy (10 conn), Server 2 is free (0 conn)
	s1 := &Server{Name: "server-1", ActiveConnections: 10, Health: true}
	s2 := &Server{Name: "server-2", ActiveConnections: 0, Health: true}

	// Reset global list
	serverList = []*Server{s1, s2}

	// Test: Should pick Server 2
	best, err := getLeastConnectedServer()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if best.Name != "server-2" {
		t.Errorf("Expected server-2 (0 conns) but got %s (%d conns)", best.Name, best.ActiveConnections)
	}

	// Scenario: Server 2 gets busy, but Server 1 clears up
	s2.ActiveConnections = 20
	s1.ActiveConnections = 5

	best, _ = getLeastConnectedServer()
	if best.Name != "server-1" {
		t.Errorf("Expected server-1 (5 conns) but got %s", best.Name)
	}
}

// ==========================================
// TEST 2: All Servers Down (Edge Case)
// ==========================================
func TestAllServersDown(t *testing.T) {
	s1 := &Server{Name: "server-1", Health: false}
	s2 := &Server{Name: "server-2", Health: false}
	serverList = []*Server{s1, s2}

	_, err := getLeastConnectedServer()
	if err == nil {
		t.Error("Expected error 'No healthy hosts', but got nil")
	}
}

// ==========================================
// TEST 3: Concurrency / Race Conditions
// ==========================================
func TestConcurrency(t *testing.T) {
	// This ensures your Mutex is working.
	// If you didn't have a Mutex, this test would likely panic or fail.
	s := &Server{ActiveConnections: 0}
	var wg sync.WaitGroup

	// Simulate 100 concurrent requests incrementing the counter at once
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.IncrementActive()
		}()
	}
	wg.Wait()

	if s.GetActive() != 100 {
		t.Errorf("Expected 100 active connections, got %d. Possible Race Condition!", s.GetActive())
	}

	// Simulate 100 concurrent requests decrementing
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.DecrementActive()
		}()
	}
	wg.Wait()

	if s.GetActive() != 0 {
		t.Errorf("Expected 0 active connections after decrement, got %d", s.GetActive())
	}
}

// ==========================================
// TEST 4: Health Checks (Mock HTTP)
// ==========================================
func TestHealthCheck(t *testing.T) {
	// Mock a healthy server (returns 200 OK)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	u, _ := url.Parse(mockServer.URL)
	lbServer := &Server{
		Name:         "test-server",
		URL:          mockServer.URL,
		ReverseProxy: httputil.NewSingleHostReverseProxy(u),
		Health:       true,
	}

	if !lbServer.CheckHealth() {
		t.Errorf("Expected server to be healthy, but it was not")
	}

	// Mock a dead server (returns 500 Error)
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badServer.Close()

	lbServer.URL = badServer.URL
	if lbServer.CheckHealth() {
		t.Errorf("Expected server to be unhealthy (500), but it passed")
	}
}

// ==========================================
// TEST 5: Config Loading (File I/O)
// ==========================================
func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	content := `[{"name": "test-1", "url": "http://localhost:9000"}]`
	tmpfile, err := os.CreateTemp("", "config_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // Clean up file after test

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Reset global list and load from temp file
	serverList = []*Server{}
	err = loadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(serverList) != 1 {
		t.Errorf("Expected 1 server loaded, got %d", len(serverList))
	}
	if serverList[0].Name != "test-1" {
		t.Errorf("Expected server name 'test-1', got %s", serverList[0].Name)
	}
}

// ==========================================
// TEST 6: Stats Dashboard Handler
// ==========================================
func TestStatsHandler(t *testing.T) {
	// Setup dummy data
	serverList = []*Server{
		{Name: "stats-server", Health: true, ActiveConnections: 5},
	}

	// Create a request to /stats
	req, err := http.NewRequest("GET", "/stats", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(statsHandler)

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check body contains JSON
	var response []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("handler returned invalid JSON: %v", err)
	}

	if response[0]["name"] != "stats-server" {
		t.Errorf("Expected JSON name 'stats-server', got %v", response[0]["name"])
	}
}
