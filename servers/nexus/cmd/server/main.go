package main

import (
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/nexus/internal/store"
	"adversarychef/nexus/internal/tools"
	"adversarychef/mcputil"
)

func main() {
	cfg := mcputil.ParseConfig("asset", "0.2.0", 8081)
	s, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer s.Close()
	mcputil.Run(cfg, func(server *mcp.Server) { tools.RegisterAll(server, s) })
}
