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
// TEST 1: The "Smart" Logic (Min-Heap / Priority Queue)
// ==========================================
func TestLeastConnectionsHeap(t *testing.T) {
	// 1. Reset the global pool for testing
	pool = ServerPool{}

	// 2. Create mock servers
	// Server 1 starts with 10 connections
	s1 := newServer("server-1", "http://localhost:8081")
	s1.ActiveConnections = 10

	// Server 2 starts with 0 connections
	s2 := newServer("server-2", "http://localhost:8082")
	s2.ActiveConnections = 0

	// 3. Add them to the Heap
	pool.AddServer(s1)
	pool.AddServer(s2)

	// 4. Test: GetNextServer should return s2 (Min connections)
	best := pool.GetNextServer()
	if best == nil {
		t.Fatalf("Expected a server, got nil")
	}
	if best.Name != "server-2" {
		t.Errorf("Expected server-2 (0 conns) but got %s (%d conns)", best.Name, best.ActiveConnections)
	}

	// 5. Scenario: Traffic Spike!
	// We simulate s2 getting hammered with requests.
	// We use IncrementActive to ensure the Heap re-balances itself.
	for i := 0; i < 20; i++ {
		pool.IncrementActive(s2)
	}
	// Now: s1 (10), s2 (20). The Min-Heap should rotate s1 to the top.

	best = pool.GetNextServer()
	if best.Name != "server-1" {
		t.Errorf("Expected server-1 (10 conns) but got %s (%d conns)", best.Name, best.ActiveConnections)
	}
}

// ==========================================
// TEST 2: All Servers Down (Empty Heap)
// ==========================================
func TestEmptyHeap(t *testing.T) {
	// Reset pool to empty
	pool = ServerPool{}

	// Should return nil if no servers exist
	best := pool.GetNextServer()
	if best != nil {
		t.Error("Expected nil when pool is empty, but got a server")
	}
}

// ==========================================
// TEST 3: Concurrency / Race Conditions
// ==========================================
func TestConcurrency(t *testing.T) {
	// Ensure Mutex prevents race conditions on connection counters
	s := &Server{ActiveConnections: 0}
	var wg sync.WaitGroup

	// Simulate 100 concurrent requests incrementing at once
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Note: In the real app we use pool.IncrementActive,
			// but here we are testing the Server struct's safety internally if we were to access it.
			// However, since we moved the lock to pool in some places, let's test the pool lock.

			// Actually, let's test the 'pool' method because that's where the lock is now for Increment
			pool.IncrementActive(s)
		}()
	}
	wg.Wait()

	if s.ActiveConnections != 100 {
		t.Errorf("Expected 100 active connections, got %d. Possible Race Condition!", s.ActiveConnections)
	}

	// Simulate 100 concurrent requests decrementing
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.DecrementActive(s)
		}()
	}
	wg.Wait()

	if s.ActiveConnections != 0 {
		t.Errorf("Expected 0 active connections after decrement, got %d", s.ActiveConnections)
	}
}

// ==========================================
// TEST 4: Health Checks (Ping)
// ==========================================
func TestPing(t *testing.T) {
	// 1. Mock a healthy backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	u, _ := url.Parse(backend.URL)
	s := &Server{
		Name:         "test-server",
		URL:          backend.URL,
		ReverseProxy: httputil.NewSingleHostReverseProxy(u),
		Health:       true,
	}

	// Test Ping() - Should return true
	if !s.Ping() {
		t.Errorf("Expected Ping to return true for active server, got false")
	}

	// 2. Test Dead Server (Wrong Port/URL)
	s.URL = "http://localhost:99999" // Invalid port

	// Reduce timeout for test speed (optional, since Ping has 2s timeout)
	// But we can just run it.

	if s.Ping() {
		t.Errorf("Expected Ping to return false for dead server, got true")
	}
}

// ==========================================
// TEST 5: Config Loading
// ==========================================
func TestLoadConfig(t *testing.T) {
	// Create a temp config file
	content := `[{"name": "test-1", "url": "http://localhost:9000"}]`
	tmpfile, err := os.CreateTemp("", "config_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Reset globals
	allServers = []*Server{}
	pool = ServerPool{}

	// Load
	err = loadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check Backup List
	if len(allServers) != 1 {
		t.Errorf("Expected 1 server in allServers, got %d", len(allServers))
	}
	// Check Heap
	if pool.servers.Len() != 1 {
		t.Errorf("Expected 1 server in Pool Heap, got %d", pool.servers.Len())
	}
}

// ==========================================
// TEST 6: Stats Dashboard Handler
// ==========================================
func TestStatsHandler(t *testing.T) {
	// Setup dummy data in the backup list
	allServers = []*Server{
		{Name: "stats-server", Health: true, ActiveConnections: 5},
	}

	req, err := http.NewRequest("GET", "/stats", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(statsHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("handler returned invalid JSON: %v", err)
	}

	if response[0]["name"] != "stats-server" {
		t.Errorf("Expected JSON name 'stats-server', got %v", response[0]["name"])
	}
}
