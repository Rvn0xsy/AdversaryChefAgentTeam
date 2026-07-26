// cmd/acactl/commands/mcp_registry.go
package commands

// mcpDisplayNames maps goose extension keys to human-readable MCP names.
// Goose normalizes URLs like "http://host.docker.internal:8080" to "127_0_0_1_8080".
var mcpDisplayNames = map[string]string{
	"127_0_0_1_8081": "nexus-mcp",
	"127_0_0_1_8080": "kali-mcp",
	"127_0_0_1_8082": "mythic-mcp",
}
