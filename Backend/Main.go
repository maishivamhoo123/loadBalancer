package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time" // <--- Import time to allow sleeping
)

func main() {
	// We will simulate 5 backend servers on ports 5001-5005
	ports := []string{"5001", "5002", "5003", "5004", "5005"}
	var wg sync.WaitGroup

	for _, port := range ports {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			mux := http.NewServeMux()

			// Handle the root path
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				// 1. GENERATE DELAY: Sleep for 2 to 4 seconds
				sleepTime := time.Duration(rand.Intn(2000)+2000) * time.Millisecond

				fmt.Printf("--> Server %s received request (processing for %v)...\n", p, sleepTime)

				// Simulate the work
				time.Sleep(sleepTime)

				// 2. Respond after the delay
				fmt.Fprintf(w, "Hello from Backend Server on Port %s!", p)
			})

			log.Printf("Starting backend server on port %s...", p)
			if err := http.ListenAndServe(":"+p, mux); err != nil {
				log.Fatal(err)
			}
		}(port)
	}

	// Keep the main function running
	wg.Wait()
}
