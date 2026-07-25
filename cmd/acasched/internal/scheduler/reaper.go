package scheduler

import (
	"log"
	"time"

	"adversarychef/acasched/internal/store"
)

func RunReaper(s *store.Store, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	_ = s
	log.Println("reaper: started")
	for range ticker.C {
		// Find running tasks past their timeout
		// For now: simple approach — dispatcher's context.WithTimeout handles it
		// Future: scan for orphaned "running" tasks and terminate them
	}
}
