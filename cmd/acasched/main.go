package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"adversarychef/acasched/internal/goose"
	"adversarychef/acasched/internal/scheduler"
	"adversarychef/acasched/internal/store"
)

func main() {
	dbPath := flag.String("db", "acasched.db", "sqlite database path")
	port := flag.Int("port", 9090, "HTTP API port")
	promptsDir := flag.String("prompts", "prompts", "prompts directory")
	nexusURL := flag.String("nexus-mcp", "http://127.0.0.1:8081", "nexus-mcp URL")
	kaliURL := flag.String("kali-mcp", "http://127.0.0.1:8080", "kali-mcp URL")
	mythicURL := flag.String("mythic-mcp", "http://127.0.0.1:8082", "mythic-mcp URL")
	flag.Parse()

	s, err := store.NewStore(*dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer s.Close()

	runner := &goose.Runner{
		PromptsDir: *promptsDir,
		NexusMCP:   *nexusURL,
		KaliMCP:    *kaliURL,
		MythicMCP:  *mythicURL,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	disp := scheduler.NewDispatcher(s, runner)
	go disp.Run(ctx)
	go runAPI(ctx, s, *port)
	go scheduler.RunReaper(s, 30)

	log.Printf("acasched started on :%d", *port)
	<-ctx.Done()
	log.Println("acasched shutting down")
}

func runAPI(ctx context.Context, s *store.Store, port int) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// MCP endpoint stub — to be expanded in a future task
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(w, `{"error":"mcp endpoint not yet implemented"}`)
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
