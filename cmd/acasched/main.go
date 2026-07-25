package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"adversarychef/acasched/internal/api"
	"adversarychef/acasched/internal/goose"
	"adversarychef/acasched/internal/scheduler"
	"adversarychef/acasched/internal/store"
)

func main() {
	dbPath := flag.String("db", "acasched.db", "sqlite database path")
	port := flag.Int("port", 9090, "HTTP API port")
	promptsDir := flag.String("prompts", "prompts", "prompts directory")
	skillsDir := flag.String("skills", "skills", "skills directory")
	logDir := flag.String("log-dir", "", "task log directory (default: ~/.aca/logs/tasks/)")
	registryPath := flag.String("registry", "prompts/_mcp-registry.yaml", "MCP registry path")
	flag.Parse()

	s, err := store.NewStore(*dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer s.Close()

	registry, err := goose.LoadRegistry(*registryPath)
	if err != nil {
		log.Fatalf("load MCP registry: %v", err)
	}

	squads, err := goose.LoadSquads(filepath.Join(*promptsDir, "_squads.yaml"))
	if err != nil {
		log.Fatalf("load squads: %v", err)
	}

	runner := &goose.Runner{
		PromptsDir: *promptsDir,
		SkillsDir:  *skillsDir,
		LogDir:     *logDir,
		Registry:   registry,
		Squads:     squads,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	disp := scheduler.NewDispatcher(s, runner)
	go disp.Run(ctx)
	go api.RunAPI(ctx, s, *logDir, *port)
	go scheduler.RunReaper(s, 30*time.Second)

	log.Printf("acasched started on :%d", *port)
	<-ctx.Done()
	log.Println("acasched shutting down")
}


