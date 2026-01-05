package main

import (
	"log"
	"time"

	"github.com/go-co-op/gocron"
)

func startHealthCheck() {
	s := gocron.NewScheduler(time.Local)

	s.Every(2).Seconds().Do(func() {
		for _, server := range allServers {
			alive := server.CheckHealth()

			// Mock check: Try to reach the URL (Simplified for example)
			// In real life, use http.Head(server.URL)
			// For this demo, we assume the server.Health boolean is truth
			// (You'd implement actual ping logic here)

			// Let's assume you have a real ping function.
			// For now, we trust the 'alive' state or use the previous simple logic.
			// But for the Heap Logic:

			if alive && server.Index == -1 {
				// Server was dead, now alive -> ADD TO HEAP
				log.Printf("✅ %s is back! Adding to Heap.", server.Name)
				pool.AddServer(server)
			} else if !alive && server.Index != -1 {
				// Server was alive, now dead -> REMOVE FROM HEAP
				log.Printf("❌ %s is down! Removing from Heap.", server.Name)
				pool.RemoveServer(server)
			}
		}
	})

	s.StartAsync()
}
