package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/nexus/internal/store"
	"adversarychef/mcputil"
)

// registerReasoningNodes registers tools for Reasoning Graph nodes
// (evidence, hypothesis, vulnerability). Implementation deferred to Task 6 continuing work.
func registerReasoningNodes(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// TODO: Task 6 — Reasoning Graph tools
}

// registerGraphEdges registers tools for graph edge CRUD.
// Implementation deferred to Task 6 continuing work.
func registerGraphEdges(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// TODO: Task 6 — Graph Edge tools
}

// registerGraphQueries registers tools for graph traversal queries.
// Implementation deferred to Task 6 continuing work.
func registerGraphQueries(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// TODO: Task 6 — Graph Query tools
}

// registerSchedulerBridge registers tools for the scheduler integration bridge.
// Implementation deferred to Task 7.
func registerSchedulerBridge(server *mcp.Server, sm *mcputil.SessionMap) {
	// TODO: Task 7 — Scheduler Bridge tools
}
