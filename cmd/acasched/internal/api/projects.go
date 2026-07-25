package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"adversarychef/acasched/internal/store"
)

func handleProjects(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			projects, err := s.ListProjects()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(projects)

		case "POST":
			var p store.Project
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if p.Status == "" {
				p.Status = "active"
			}
			if err := s.CreateProject(&p); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(p)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleProjectByID(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		// Strip trailing segments like /tasks
		if idx := strings.Index(id, "/"); idx > 0 {
			id = id[:idx]
		}
		if id == "" {
			http.Error(w, "missing project id", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case "GET":
			p, err := s.GetProject(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(p)

		case "PATCH":
			var updates map[string]string
			if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Get current to preserve unchanged fields
			cur, err := s.GetProject(id)
			if err != nil {
				http.Error(w, "project not found", http.StatusNotFound)
				return
			}
			name := cur.Name
			desc := cur.Description
			status := cur.Status
			if v, ok := updates["name"]; ok {
				name = v
			}
			if v, ok := updates["description"]; ok {
				desc = v
			}
			if v, ok := updates["status"]; ok {
				status = v
			}
			if err := s.UpdateProject(id, name, desc, status); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
