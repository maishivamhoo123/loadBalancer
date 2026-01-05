# 🚀 DSA-Powered High-Performance Load Balancer

A production-ready **Layer 7 HTTP Load Balancer** built in Go (Golang). This project optimizes traffic distribution using a **Min-Heap (Priority Queue)** data structure to achieve **O(1)** server selection, ensuring high throughput and low latency even under heavy loads.

It features concurrent request handling, active health checking, and a real-time monitoring dashboard.

---

## ⚡ Key Features

* **Optimal Routing (DSA):** Uses a **Min-Heap** to instantly select the backend server with the least active connections ($O(1)$ time complexity).
* **Concurrency Safe:** Implements **Mutex locks** (`sync.RWMutex`) to prevent race conditions during high-volume traffic.
* **Fault Tolerance:** Background **Health Check Daemon** automatically removes dead servers from the rotation and re-adds them when they recover.
* **Real-Time Observability:** Built-in HTML/JS **Status Dashboard** to visualize server health and traffic distribution.
* **Dynamic Configuration:** Load backend server pools via a simple `config.json` file.

---

## 🧠 The DSA Implementation (Why this matters)

Most simple load balancers use a linear loop to find the best server. This works for small scales but degrades as the server pool grows. I optimized this bottleneck using a **Binary Heap**.

| Operation | Standard Approach (Linear Search) | My Solution (Min-Heap) |
| :--- | :--- | :--- |
| **Find Best Server** | $O(N)$ (Scan all servers) | **$O(1)$ (Peek Root)** |
| **Update Connections** | $O(1)$ | **$O(\log N)$ (Re-balance)** |
| **Scalability** | Linear degradation | Logarithmic scaling |

**Core Logic:**
The `ServerPool` manages a Priority Queue where the "priority" is the number of active connections. The server with the fewest connections always "floats" to the top of the heap.

---

## 📂 Code Breakdown

Here is how the project is architected:

### 1. `server.go` (The Blueprint)
Defines the `Server` struct.
* **ReverseProxy:** Uses Go's `httputil` to forward actual traffic.
* **Mutex:** Protects the connection counter from race conditions.
* **Ping():** Sends a 2-second timeout `HEAD` request to check if a backend is alive.

### 2. `server_heap.go` (The DSA Core)
Implements Go's `heap.Interface`.
* **Len() / Swap() / Less():** Standard sort interface. `Less()` determines that servers with *fewer* connections come first.
* **Push() / Pop():** Handles adding/removing items from the stack.
* **ServerPool:** A wrapper that makes the Heap thread-safe.
    * `GetNextServer()`: Returns the top server (Index 0) in **O(1)**.
    * `IncrementActive()`: Adds connection +1, then calls `heap.Fix()` to sink the node down.
    * `DecrementActive()`: Subtracts connection -1, then calls `heap.Fix()` to float the node up.

### 3. `health.go` (The Doctor)
Runs a background Goroutine every 2 seconds.
* It iterates through all known servers and runs `Ping()`.
* **Self-Healing Logic:**
    * If a server is **Dead** but currently in the Heap -> **Remove it** (Stop sending traffic).
    * If a server is **Alive** but NOT in the Heap -> **Add it** (Recovered).

### 4. `main.go` (The Entry Point)
* Loads `config.json`.
* Initializes the Heap and Health Checkers.
* **ForwardRequest:** The main HTTP handler.
    1.  Get Best Server (Heap).
    2.  Increment Active Count.
    3.  Forward Request.
    4.  Decrement Active Count.

---

## 🛠️ Project Structure

```text
.
├── config.json        # List of backend servers
├── main.go            # Entry point & HTTP Handlers
├── main_test.go       # Unit tests for Heap & Logic
├── health.go          # Background health check worker
├── server.go          # Server struct & Networking logic
└── server_heap.go     # Min-Heap DSA implementation
🚀 Getting Started
Prerequisites
Go 1.18 or higher installed.

1. Clone the Repository
Bash

git clone [https://github.com/yourusername/dsa-load-balancer.git](https://github.com/yourusername/dsa-load-balancer.git)
cd dsa-load-balancer
2. Install Dependencies
We use gocron for the health check scheduler.

Bash

go get [github.com/go-co-op/gocron](https://github.com/go-co-op/gocron)
go mod tidy
3. Configure Servers
Create a file named config.json in the root directory:

JSON

[
  {
    "name": "Server 1",
    "url": "http://localhost:8081"
  },
  {
    "name": "Server 2",
    "url": "http://localhost:8082"
  },
  {
    "name": "Server 3",
    "url": "http://localhost:8083"
  }
]
4. Run the Load Balancer
Bash

go run .
You will see logs indicating the server has started:

🚀 DSA Load Balancer starting on port :8000

🧪 How to Test
Start Backend Services: You can use python to quickly spin up dummy servers to test:

Bash

# Terminal 1
python3 -m http.server 8081
# Terminal 2
python3 -m http.server 8082
Send Traffic: Open your browser and visit http://localhost:8000. The load balancer will forward your request to one of the active backends.

View Dashboard: Go to http://localhost:8000/dashboard to see the live status of your servers.

Simulate Failure: Kill one of the python servers. Watch the dashboard—the status will turn to Offline and traffic will stop flowing to that node. Restart it, and it will rejoin the pool.

🤝 Future Improvements
Weighted Round Robin: Support servers with different capacities (e.g., a powerful server gets 2x traffic).

Retries: Automatically retry a request on a different server if the chosen one fails.

Dockerization: Containerize the application for easy deployment.