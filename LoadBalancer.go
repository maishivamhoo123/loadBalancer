package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// --- DSA UPGRADE ---
// 1. 'pool' manages the Min-Heap for fast selection (O(1))
// 2. 'allServers' is a simple list used for Health Checks and Stats
var pool ServerPool
var allServers []*Server

func main() {
	// Initialize the Heap Pool
	pool = ServerPool{}

	// 1. Load Configuration
	err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading configuration: %s", err)
	}
	log.Printf("Loaded %d servers from config", len(allServers))

	// 2. Register Routes
	http.HandleFunc("/", ForwardRequest)

	// API for JSON stats
	http.HandleFunc("/stats", statsHandler)

	// Visual Dashboard
	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, dashboardHTML)
	})

	// 3. Start Health Check (Background)
	go startHealthCheck()

	// 4. Start Server
	log.Printf("🚀 DSA Load Balancer starting on port :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func ForwardRequest(res http.ResponseWriter, rep *http.Request) {
	// --- DSA MAGIC START ---
	// Instead of looping (O(N)), we just peek at the top of the heap (O(1))
	target := pool.GetNextServer()

	if target == nil {
		http.Error(res, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	// Increment connection count & Re-balance the Heap (O(log N))
	pool.IncrementActive(target)

	// Log only the active connections to keep terminal clean
	log.Printf("Forwarding to %s (Active: %d)", target.Name, target.ActiveConnections)

	// Forward the request
	target.ReverseProxy.ServeHTTP(res, rep)

	// Decrement connection count & Re-balance the Heap (O(log N))
	pool.DecrementActive(target)
	// --- DSA MAGIC END ---
}

// statsHandler returns the current status of all servers as JSON
func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type ServerStats struct {
		Name   string `json:"name"`
		URL    string `json:"url"`
		Health bool   `json:"health"`
		Active int    `json:"active_connections"`
	}

	var stats []ServerStats
	// We read from 'allServers' to show stats even for dead servers
	for _, s := range allServers {
		stats = append(stats, ServerStats{
			Name:   s.Name,
			URL:    s.URL,
			Health: s.CheckHealth(),
			Active: s.GetActive(),
		})
	}

	json.NewEncoder(w).Encode(stats)
}

// loadConfig reads servers from a JSON file
func loadConfig(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var configs []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := json.Unmarshal(data, &configs); err != nil {
		return err
	}

	for _, c := range configs {
		s := newServer(c.Name, c.URL)

		// Add to the backup list (for stats)
		allServers = append(allServers, s)

		// Add to the DSA Heap (for traffic routing)
		pool.AddServer(s)
	}
	return nil
}

// ==========================================================
// DASHBOARD HTML CODE
// ==========================================================
const dashboardHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>DSA Load Balancer Dashboard</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; padding: 20px; background: #f4f7f6; color: #333; }
        h1 { color: #2c3e50; text-align: center; margin-bottom: 30px; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { padding: 15px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #3498db; color: white; text-transform: uppercase; font-size: 0.9em; letter-spacing: 1px; }
        tr:hover { background-color: #f1f1f1; }
        .status-badge { padding: 5px 10px; border-radius: 15px; font-weight: bold; font-size: 0.85em; }
        .up { background-color: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .down { background-color: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .conn-count { font-weight: bold; color: #2c3e50; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 DSA Load Balancer Status</h1>
        <table id="serverTable">
            <thead>
                <tr>
                    <th>Server Name</th>
                    <th>Address</th>
                    <th>Status</th>
                    <th>Active Connections</th>
                </tr>
            </thead>
            <tbody></tbody>
        </table>
    </div>

    <script>
        function updateStats() {
            fetch('/stats')
                .then(response => response.json())
                .then(data => {
                    const tbody = document.querySelector('#serverTable tbody');
                    tbody.innerHTML = '';
                    data.forEach(server => {
                        const row = document.createElement('tr');
                        const statusClass = server.health ? 'up' : 'down';
                        const statusText = server.health ? ' Online' : 'Offline';
                        
                        row.innerHTML = '<td><strong>' + server.name + '</strong></td>' +
                                        '<td>' + server.url + '</td>' +
                                        '<td><span class="status-badge ' + statusClass + '">' + statusText + '</span></td>' +
                                        '<td class="conn-count">' + server.active_connections + '</td>';
                        tbody.appendChild(row);
                    });
                })
                .catch(error => console.error('Error fetching stats:', error));
        }
        
        // Update every 1 second
        setInterval(updateStats, 1000); 
        updateStats(); 
    </script>
</body>
</html>
`
