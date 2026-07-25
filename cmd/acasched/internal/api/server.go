package api

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"adversarychef/acasched/internal/store"
)

// RunAPI starts the HTTP API server and blocks until ctx is cancelled.
func RunAPI(ctx context.Context, s *store.Store, logDir string, port int) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/tasks", handleTasks(s))
	mux.HandleFunc("/api/tasks/", handleTaskByID(s))
	mux.HandleFunc("GET /api/tasks/{id}/logs", handleTaskLogs(logDir))
	mux.HandleFunc("/api/projects", handleProjects(s))
	mux.HandleFunc("/api/projects/", handleProjectByID(s))
	mux.HandleFunc("GET /api/projects/{id}/tasks", handleProjectTasks(s))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("api: listening on :%d", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("api: %v", err)
	}
}
