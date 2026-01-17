package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

var pool ServerPool
var allServers []*Server

func main() {
	pool = ServerPool{}

	// 1. Load Configuration
	err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading configuration: %s", err)
	}
	log.Printf("Loaded %d servers from config", len(allServers))

	// 2. Register Routes
	http.HandleFunc("/", ForwardRequest)
	http.HandleFunc("/stats", statsHandler)
	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, dashboardHTML)
	})

	// 3. Start Health Check (Background)
	go startHealthCheck()

	log.Printf("🚀 Weighted DSA Load Balancer starting on port :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func ForwardRequest(res http.ResponseWriter, rep *http.Request) {
	target := pool.GetNextServer()

	if target == nil {
		http.Error(res, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	pool.IncrementActive(target)
	log.Printf("Forwarding to %s (Load Ratio: %.2f)", target.Name, float64(target.ActiveConnections)/float64(target.Weight))

	target.ReverseProxy.ServeHTTP(res, rep)

	pool.DecrementActive(target)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type ServerStats struct {
		Name   string `json:"name"`
		URL    string `json:"url"`
		Weight int    `json:"weight"`
		Health bool   `json:"health"`
		Active int    `json:"active_connections"`
	}

	var stats []ServerStats
	for _, s := range allServers {
		stats = append(stats, ServerStats{
			Name:   s.Name,
			URL:    s.URL,
			Weight: s.Weight,
			Health: s.CheckHealth(),
			Active: s.GetActive(),
		})
	}
	json.NewEncoder(w).Encode(stats)
}

func loadConfig(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var configs []struct {
		Name   string `json:"name"`
		URL    string `json:"url"`
		Weight int    `json:"weight"`
	}
	if err := json.Unmarshal(data, &configs); err != nil {
		return err
	}

	for _, c := range configs {
		s := newServer(c.Name, c.URL)
		s.Weight = c.Weight
		if s.Weight <= 0 {
			s.Weight = 1
		}
		allServers = append(allServers, s)
		pool.AddServer(s)
	}
	return nil
}

const dashboardHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>DSA Load Balancer Dashboard</title>
    <style>
        body { font-family: 'Segoe UI', sans-serif; padding: 20px; background: #f4f7f6; }
        .container { max-width: 900px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { padding: 15px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #3498db; color: white; }
        .status-badge { padding: 5px 10px; border-radius: 15px; font-weight: bold; }
        .up { background-color: #d4edda; color: #155724; }
        .down { background-color: #f8d7da; color: #721c24; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📊 DSA Weighted Load Balancer</h1>
        <table id="serverTable">
            <thead>
                <tr>
                    <th>Server Name</th>
                    <th>Address</th>
                    <th>Weight (Capacity)</th>
                    <th>Status</th>
                    <th>Active Connections</th>
                </tr>
            </thead>
            <tbody></tbody>
        </table>
    </div>
    <script>
        function updateStats() {
            fetch('/stats').then(res => res.json()).then(data => {
                const tbody = document.querySelector('#serverTable tbody');
                tbody.innerHTML = '';
                data.forEach(s => {
                    const row = document.createElement('tr');
                    const statusClass = s.health ? 'up' : 'down';
                    row.innerHTML = '<td>' + s.name + '</td>' +
                                    '<td>' + s.url + '</td>' +
                                    '<td>' + s.weight + '</td>' +
                                    '<td><span class="status-badge ' + statusClass + '">' + (s.health ? 'Online' : 'Offline') + '</span></td>' +
                                    '<td>' + s.active_connections + '</td>';
                    tbody.appendChild(row);
                });
            });
        }
        setInterval(updateStats, 1000);
        updateStats();
    </script>
</body>
</html>`
