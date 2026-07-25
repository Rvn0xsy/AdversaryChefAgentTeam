package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/nexus/internal/store"
	"adversarychef/mcputil"
)

// registerGraphQueries registers tools for graph traversal queries.
func registerGraphQueries(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// graph_query — BFS traverse from a node
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "graph_query",
		Description: "BFS traverse from a node, up to max_hops (default 2)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID   string `json:"project_id" jsonschema:"Project ID"`
		StartNodeID string `json:"start_node_id" jsonschema:"Node ID to start traversal from"`
		MaxHops     int    `json:"max_hops,omitempty" jsonschema:"Maximum number of hops (default 2)"`
	}) (*mcp.CallToolResult, any, error) {
		if params.MaxHops == 0 {
			params.MaxHops = 2
		}
		sub, err := s.GraphQuery(params.ProjectID, params.StartNodeID, params.MaxHops)
		if err != nil {
			return mcputil.TextResult("query failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(sub)
		return mcputil.TextResult(string(b)), nil, nil
	})

	// graph_trace — backtrace from a node to its evidence sources
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "graph_trace",
		Description: "Backtrace from a node to its evidence sources",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID string `json:"project_id" jsonschema:"Project ID"`
		NodeID    string `json:"node_id" jsonschema:"Node ID to trace from"`
	}) (*mcp.CallToolResult, any, error) {
		trace, err := s.GraphTrace(params.ProjectID, params.NodeID)
		if err != nil {
			return mcputil.TextResult("trace failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(trace)
		return mcputil.TextResult(string(b)), nil, nil
	})
}
