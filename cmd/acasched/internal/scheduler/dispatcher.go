package scheduler

import (
	"context"
	"log"
	"time"

	"adversarychef/acasched/internal/goose"
	"adversarychef/acasched/internal/store"
)

type Dispatcher struct {
	store   *store.Store
	runner  *goose.Runner
	running map[string]context.CancelFunc
}

func NewDispatcher(s *store.Store, r *goose.Runner) *Dispatcher {
	return &Dispatcher{store: s, runner: r, running: map[string]context.CancelFunc{}}
}

func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.tick()
		}
	}
}

func (d *Dispatcher) tick() {
	tasks, err := d.store.ListPending("")
	if err != nil {
		log.Printf("dispatcher: list pending: %v", err)
		return
	}
	log.Printf("dispatcher: tick: %d pending tasks", len(tasks))
	for _, t := range tasks {
		if t.ParentID != "" && !d.parentReady(t.ParentID) {
			continue
		}
		log.Printf("dispatcher: dispatching %s (agent=%s)", t.ID, t.Agent)
		go d.dispatchOne(t)
	}
}

func (d *Dispatcher) dispatchOne(task store.Task) {
	log.Printf("dispatcher: dispatching task %s (agent=%s)", task.ID, task.Agent)
	TransitionToDispatched(d.store, task.ID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(task.TimeoutSecs)*time.Second)
	d.running[task.ID] = cancel
	defer func() {
		cancel()
		delete(d.running, task.ID)
	}()

	TransitionToRunning(d.store, task.ID)
	log.Printf("dispatcher: running goose for %s", task.ID)
	result, err := d.runner.Execute(ctx, &task)
	if err != nil {
		log.Printf("dispatcher: goose error for %s: %v", task.ID, err)
		if ctx.Err() == context.DeadlineExceeded {
			TransitionToTimeout(d.store, task.ID)
		} else if task.Attempt < task.RetryCount {
			task.Attempt++
			task.Description += "\n[RETRY] " + err.Error()
			d.store.UpdateStatus(task.ID, "pending", "", "")
			return
		} else {
			TransitionToFailed(d.store, task.ID, err.Error())
		}
		return
	}
	TransitionToDone(d.store, task.ID, result.Summary)
	TriggerParent(d.store, task.ParentID)
}

func (d *Dispatcher) parentReady(parentID string) bool {
	p, err := d.store.GetTask(parentID)
	if err != nil {
		return false
	}
	return p.Status == "done" || p.Status == "failed" || p.Status == "timeout"
}
