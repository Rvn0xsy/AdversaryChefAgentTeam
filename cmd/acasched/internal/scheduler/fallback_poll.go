package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"adversarychef/acasched/internal/goose"
	"adversarychef/acasched/internal/store"
)

func RunFallbackPoll(ctx context.Context, s *store.Store) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			projects, err := s.ListProjects()
			if err != nil {
				log.Printf("fallback_poll: list projects: %v", err)
				continue
			}
			for _, p := range projects {
				if p.Status != "active" {
					continue
				}
				evtMu.Lock()
				lastT, ok := lastEventTime[p.ID]
				isRunning := runningSupers[p.ID]

				if isRunning {
					evtMu.Unlock()
					continue
				}
				if ok && time.Since(lastT) < 120*time.Second {
					evtMu.Unlock()
					continue
				}

				taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
				task := &store.Task{
					ID:          taskID,
					ProjectID:   p.ID,
					Agent:       "red-team/supervisor",
					Status:      "pending",
					Title:       "Supervisor Evaluation (fallback)",
					Description: fmt.Sprintf("Fallback evaluation.\nProject: %s\nGoal: %s", p.Name, p.Description),
					CreatedBy:   "fallback_poll",
					MaxTurns:    goose.GetMaxTurns("red-team/supervisor"),
					TimeoutSecs: 600,
					RetryCount:  0,
				}
				if err := s.CreateTask(task); err != nil {
					log.Printf("fallback_poll: create supervisor: %v", err)
					evtMu.Unlock()
					continue
				}
				runningSupers[p.ID] = true
				lastEventTime[p.ID] = time.Now()
				evtMu.Unlock()
				log.Printf("fallback_poll: triggered supervisor for project %s", p.ID)
			}
		}
	}
}
