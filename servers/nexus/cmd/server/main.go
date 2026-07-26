package main

import (
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/nexus/internal/store"
	"adversarychef/nexus/internal/tools"
	"adversarychef/mcputil"
)

func main() {
	cfg := mcputil.ParseConfig("nexus", "0.3.0", 8081)
	s, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer s.Close()

	sessionMap := mcputil.NewSessionMap()

	schedulerURL := os.Getenv("SCHEDULER_URL")
	if schedulerURL == "" {
		schedulerURL = "http://127.0.0.1:9090"
	}

	webhookURL := os.Getenv("ACASCHED_WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "http://127.0.0.1:9090/api/events"
	}
	eventedStore := store.NewEventedStore(s, webhookURL)

	mcputil.Run(cfg, func(server *mcp.Server) {
		tools.RegisterAllV3(server, eventedStore, sessionMap, schedulerURL)
	}, sessionMap)
}
