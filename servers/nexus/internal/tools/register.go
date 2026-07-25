package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/nexus/internal/store"
	"adversarychef/mcputil"
)

// RegisterAll registers all CRUD tools + search + stats.
func RegisterAll(server *mcp.Server, s store.Store) {
	registerProjects(server, s)
	registerAssets(server, s)
	registerClues(server, s)
	registerCredentials(server, s)
	registerWorkLogs(server, s)
	registerSearch(server, s)
}

// RegisterAllV2 registers all V1 tools plus Operation Graph, Reasoning Graph,
// Graph Edges, Graph Queries, and Scheduler Bridge tools.
func RegisterAllV2(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	RegisterAll(server, s)
	registerGraphNodes(server, s, sm)      // Operation Graph — Task 6
	registerReasoningNodes(server, s, sm)   // Reasoning Graph — Task 6
	registerGraphEdges(server, s, sm)       // Graph Edges — Task 6
	registerGraphQueries(server, s, sm)     // Graph Queries — Task 6
	registerSchedulerBridge(server, sm)      // Scheduler Bridge — Task 7
}
