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
			alive := server.Ping() // Real ping check
			server.SetHealth(alive)

			if alive && server.Index == -1 {
				log.Printf("✅ %s recovered. Adding to pool.", server.Name)
				pool.AddServer(server)
			} else if !alive && server.Index != -1 {
				log.Printf("❌ %s failed health check. Removing from pool.", server.Name)
				pool.RemoveServer(server)
			}
		}
	})
	s.StartAsync()
}
