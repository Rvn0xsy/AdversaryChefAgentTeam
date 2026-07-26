package scheduler

import (
	"fmt"
	"strings"

	"adversarychef/acasched/internal/store"
)

func TriggerParent(s *store.Store, parentID string) {
	if parentID == "" {
		return
	}
	children, err := s.FindChildren(parentID)
	if err != nil {
		return
	}
	allTerminal := true
	for _, c := range children {
		if c.Status != "done" && c.Status != "failed" && c.Status != "timeout" && c.Status != "skipped" {
			allTerminal = false
			break
		}
	}
	if !allTerminal {
		return
	}

	parent, err := s.GetTask(parentID)
	if err != nil {
		return
	}

	// If the parent is a supervisor, enqueue an event instead of setting pending
	if strings.Contains(parent.Agent, "supervisor") {
		if len(children) > 0 {
			EnqueueEvent(children[0].ProjectID)
		}
		return
	}

	// Inject child results into parent description
	summary := "\n\n## Child Task Results\n"
	for _, c := range children {
		ct, _ := s.GetTask(c.ID)
		summary += fmt.Sprintf("- %s (%s): %s\n", ct.Title, ct.Status, truncate(ct.Result, 200))
	}
	parent.Description += summary
	parent.Status = "pending"
	s.UpdateStatus(parent.ID, "pending", "", "")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
