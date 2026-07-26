package api

import (
	"encoding/json"
	"log"
	"net/http"

	"adversarychef/acasched/internal/scheduler"
	"adversarychef/acasched/internal/store"
)

type GraphEvent struct {
	ProjectID string `json:"project_id"`
	Action    string `json:"action"`
	Entity    string `json:"entity"`
	NodeID    string `json:"node_id"`
	ParentID  string `json:"parent_id"`
	Summary   string `json:"summary"`
	Timestamp string `json:"timestamp"`
}

func handleEvents(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var evt GraphEvent
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if evt.ProjectID == "" {
			http.Error(w, "missing project_id", http.StatusBadRequest)
			return
		}
		log.Printf("events: received %s.%s for project %s", evt.Entity, evt.Action, evt.ProjectID)
		scheduler.EnqueueEvent(evt.ProjectID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}
