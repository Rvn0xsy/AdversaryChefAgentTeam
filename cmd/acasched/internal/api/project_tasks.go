package api

import (
	"encoding/json"
	"net/http"

	"adversarychef/acasched/internal/store"
)

func handleProjectTasks(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := r.PathValue("id")
		if projectID == "" {
			http.Error(w, "missing project id", http.StatusBadRequest)
			return
		}
		tasks, err := s.ListTasksByProject(projectID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	}
}
