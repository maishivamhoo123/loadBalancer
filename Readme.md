# Go Load Balancer

A high-performance, custom-built Load Balancer written in Go. This project distributes HTTP traffic across multiple backend servers using the **"Least Connections"** algorithm, ensuring optimal load distribution. It includes active health checking, a real-time status dashboard, and dynamic configuration.

## 🚀 Features

* **Least Connections Strategy:** Routes traffic to the server with the fewest active requests.
* **Active Health Checks:** Automatically detects and removes dead servers from the rotation.
* **Real-time Dashboard:** View server status and active connection counts at `/stats`.
* **Concurrency Safe:** Handles concurrent requests using Mutex locking.
* **Configurable:** Load backend servers dynamically from a JSON file.

## 📋 Prerequisites

* [Go (Golang)](https://go.dev/dl/) installed (version 1.18+ recommended).

## 🛠️ Installation

1.  **Clone or create the project folder:**
    ```bash
    mkdir my-loadbalancer
    cd my-loadbalancer
    ```

2.  **Initialize the module and install dependencies:**
    ```bash
    go mod init my-loadbalancer
    go get [github.com/go-co-op/gocron](https://github.com/go-co-op/gocron)
    ```

## 🏃‍♂️ Quick Start

To see the load balancer in action, you need to run the **Backends** (simulated servers) and the **Load Balancer** simultaneously.

### Step 1: Start Backend Servers
Open a new terminal. Run the simulation script (assumes it is located in a `backends/` subfolder):

```bash
go run backends/main.go