// Mythic MCP Server — exposes Mythic C2 operations as MCP tools.
package main

import (
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/mcputil"
	"adversarychef/mythic/internal/client"
	"adversarychef/mythic/internal/tools"
)

func main() {
	cfg := mcputil.ParseConfig("mythic", "0.1.0", 8082)

	c := client.NewClientWithConfig(cfg.MythicServer, cfg.MythicAPIKey)
	if err := c.ValidateConfig(); err != nil {
		log.Printf("[mythic] warning: %v (server will start but tools may fail until configured)", err)
	}

	mcputil.Run(cfg, func(server *mcp.Server) {
		tools.Register(server, c)
	})
}
