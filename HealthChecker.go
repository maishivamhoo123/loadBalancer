package main

import (
	"log"
	"time"

	"github.com/go-co-op/gocron"
)

func startHealthCheck() {
	s := gocron.NewScheduler(time.Local)

	// Run every 2 seconds
	s.Every(2).Seconds().Do(func() {
		for _, server := range serverList {
			// 1. Remember the old status
			oldStatus := server.Health

			// 2. Check the new status
			newStatus := server.CheckHealth()

			// 3. ONLY log if the status CHANGED
			if oldStatus != newStatus {
				if newStatus {
					log.Printf("✅ STATUS CHANGE: %s is back ONLINE!", server.Name)
				} else {
					log.Printf("❌ STATUS CHANGE: %s is DOWN!", server.Name)
				}
			}
			// If status is the same, do nothing (Stay Silent)
		}
	})

	s.StartAsync()
}
