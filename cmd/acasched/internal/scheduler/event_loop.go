package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"adversarychef/acasched/internal/goose"
	"adversarychef/acasched/internal/store"
)

var (
	eventQueue    = make(chan string, 100)
	runningSupers = make(map[string]bool)
	lastEventTime = make(map[string]time.Time)
	evtMu         sync.Mutex
)

func EnqueueEvent(projectID string) {
	select {
	case eventQueue <- projectID:
	default:
		log.Printf("event_loop: queue full, dropping event for %s", projectID)
	}
}

func SupervisorDone(projectID string) {
	evtMu.Lock()
	delete(runningSupers, projectID)
	evtMu.Unlock()
}

func RunEventLoop(ctx context.Context, s *store.Store) {
	debounceTimers := make(map[string]*time.Timer)

	for {
		select {
		case <-ctx.Done():
			return
		case projectID := <-eventQueue:
			evtMu.Lock()
			if t, ok := debounceTimers[projectID]; ok {
				t.Stop()
			}
			debounceTimers[projectID] = time.AfterFunc(5*time.Second, func() {
				evtMu.Lock()
				if runningSupers[projectID] {
					evtMu.Unlock()
					return
				}
				proj, err := s.GetProject(projectID)
				if err != nil {
					log.Printf("event_loop: get project %s: %v", projectID, err)
					evtMu.Unlock()
					return
				}
				taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
				task := &store.Task{
					ID:          taskID,
					ProjectID:   projectID,
					Agent:       "red-team/supervisor",
					Status:      "pending",
					Title:       "Supervisor Evaluation",
					Description: fmt.Sprintf("Evaluate project state and dispatch agents.\nProject: %s\nGoal: %s", proj.Name, proj.Description),
					CreatedBy:   "event_loop",
					MaxTurns:    goose.GetMaxTurns("red-team/supervisor"),
					TimeoutSecs: 600,
					RetryCount:  0,
				}
				if err := s.CreateTask(task); err != nil {
					log.Printf("event_loop: create supervisor task: %v", err)
					evtMu.Unlock()
					return
				}
				runningSupers[projectID] = true
				lastEventTime[projectID] = time.Now()
				evtMu.Unlock()
				log.Printf("event_loop: dispatched supervisor for project %s", projectID)
			})
			evtMu.Unlock()
		}
	}
}
