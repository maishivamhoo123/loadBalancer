package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

// ==========================================
// TEST 1: The "Weighted" Logic (Load Ratio)
// ==========================================
func TestWeightedLeastConnectionsHeap(t *testing.T) {
	// Reset the global pool
	pool = ServerPool{}

	// Scenario:
	// Server A: Weight 10, Conns 20 -> Ratio = 2.0
	// Server B: Weight 2,  Conns 10 -> Ratio = 5.0
	// Even though Server A has MORE connections, it is "emptier" relative to its capacity.

	s1 := newServer("power-server", "http://localhost:8081")
	s1.Weight = 10
	s1.ActiveConnections = 20

	s2 := newServer("weak-server", "http://localhost:8082")
	s2.Weight = 2
	s2.ActiveConnections = 10

	pool.AddServer(s1)
	pool.AddServer(s2)

	// Test: Should pick s1 (Ratio 2.0 is better than 5.0)
	best := pool.GetNextServer()
	if best == nil {
		t.Fatalf("Expected a server, got nil")
	}
	if best.Name != "power-server" {
		t.Errorf("Expected power-server (Ratio 2.0) but got %s", best.Name)
	}

	// Scenario: Hammer Server A until it's busier than Server B
	// Increase s1 by 40 -> Ratio becomes 60/10 = 6.0
	for i := 0; i < 40; i++ {
		pool.IncrementActive(s1)
	}

	// Test: Should now pick s2 (Ratio 5.0 is now better than 6.0)
	best = pool.GetNextServer()
	if best.Name != "weak-server" {
		t.Errorf("Expected weak-server to be picked after traffic spike on s1")
	}
}

// ==========================================
// TEST 2: Concurrency & Mutex Safety
// ==========================================
func TestConcurrency(t *testing.T) {
	pool = ServerPool{}
	s := newServer("concurrent-server", "http://localhost:8083")
	s.Weight = 1
	pool.AddServer(s)

	var wg sync.WaitGroup
	numRequests := 200

	// Simulate 200 concurrent users hitting the load balancer
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.IncrementActive(s)
		}()
	}
	wg.Wait()

	if s.ActiveConnections != numRequests {
		t.Errorf("Race Condition! Expected %d conns, got %d", numRequests, s.ActiveConnections)
	}

	// Simulate 200 concurrent completions
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.DecrementActive(s)
		}()
	}
	wg.Wait()

	if s.ActiveConnections != 0 {
		t.Errorf("Expected 0 conns after cleanup, got %d", s.ActiveConnections)
	}
}

// ==========================================
// TEST 3: Health Checks (Network Ping)
// ==========================================
func TestPing(t *testing.T) {
	// 1. Mock a Healthy Server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	s := newServer("test-server", backend.URL)
	if !s.Ping() {
		t.Error("Ping failed on a healthy server")
	}

	// 2. Mock a Dead Server (invalid address)
	s.URL = "http://localhost:9999"
	if s.Ping() {
		t.Error("Ping succeeded on a dead server")
	}
}

// ==========================================
// TEST 4: Config Load with Weights
// ==========================================
func TestLoadConfig(t *testing.T) {
	content := `[{"name": "heavy", "url": "http://loc:5001", "weight": 10}]`
	tmpfile, _ := os.CreateTemp("", "conf*.json")
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte(content))
	tmpfile.Close()

	allServers = []*Server{}
	pool = ServerPool{}

	err := loadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if allServers[0].Weight != 10 {
		t.Errorf("Weight not loaded. Got %d", allServers[0].Weight)
	}
}

// ==========================================
// TEST 5: Stats Handler API
// ==========================================
func TestStatsHandler(t *testing.T) {
	allServers = []*Server{
		{Name: "api-test", Health: true, ActiveConnections: 3, Weight: 5},
	}

	req, _ := http.NewRequest("GET", "/stats", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(statsHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Stats API returned status %d", rr.Code)
	}

	var stats []map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &stats)

	if stats[0]["weight"].(float64) != 5 {
		t.Errorf("Stats JSON weight mismatch")
	}
}
