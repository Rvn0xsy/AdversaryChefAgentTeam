package tools
import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"adversarychef/asset/internal/store"
)

// RegisterAll registers all CRUD tools.
func RegisterAll(server *mcp.Server, s store.Store) {
	registerProjects(server, s)
	registerAssets(server, s)
	registerClues(server, s)
	registerCredentials(server, s)
	registerWorkLogs(server, s)
}
