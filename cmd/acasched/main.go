package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"adversarychef/acasched/internal/api"
	"adversarychef/acasched/internal/goose"
	"adversarychef/acasched/internal/scheduler"
	"adversarychef/acasched/internal/store"
)

func main() {
	dbPath := flag.String("db", "acasched.db", "sqlite database path")
	port := flag.Int("port", 9090, "HTTP API port")
	promptsDir := flag.String("prompts", "prompts", "prompts directory")
	logDir := flag.String("log-dir", "", "task log directory (default: ~/.aca/logs/tasks/)")
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
		LogDir:     *logDir,
		NexusMCP:   *nexusURL,
		KaliMCP:    *kaliURL,
		MythicMCP:  *mythicURL,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	disp := scheduler.NewDispatcher(s, runner)
	go disp.Run(ctx)
	go api.RunAPI(ctx, s, *logDir, *port)
	go scheduler.RunReaper(s, 30)

	log.Printf("acasched started on :%d", *port)
	<-ctx.Done()
	log.Println("acasched shutting down")
}


