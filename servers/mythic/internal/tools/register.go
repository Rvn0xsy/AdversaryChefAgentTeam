// Package tools provides MCP tool wrappers for the Mythic C2 client.
package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"adversarychef/mythic/internal/client"
)

// Register registers all Mythic MCP tools.
func Register(server *mcp.Server, c *client.Client) {
	registerCallbacks(server, c)
	registerTasks(server, c)
	registerFiles(server, c)
	registerPayloads(server, c)
}
