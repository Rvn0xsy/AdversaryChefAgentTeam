package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/nexus/internal/models"
	"adversarychef/nexus/internal/store"
	"adversarychef/mcputil"
)

// registerGraphEdges registers tools for graph edge CRUD.
func registerGraphEdges(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// edge_create
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "edge_create",
		Description: "Create a graph edge between two nodes",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID    string   `json:"project_id" jsonschema:"Project ID"`
		FromID       string   `json:"from_id" jsonschema:"Source node ID"`
		ToID         string   `json:"to_id" jsonschema:"Target node ID"`
		EdgeType     string   `json:"edge_type" jsonschema:"Edge type (e.g. supports, refutes, derived_from, exploited_by)"`
		EvidenceRefs []string `json:"evidence_refs,omitempty" jsonschema:"Evidence node IDs supporting this edge"`
	}) (*mcp.CallToolResult, any, error) {
		e := &models.GraphEdge{
			ProjectID:    params.ProjectID,
			FromID:       params.FromID,
			ToID:         params.ToID,
			EdgeType:     params.EdgeType,
			EvidenceRefs: params.EvidenceRefs,
		}
		if err := s.CreateEdge(e); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(e)
		return mcputil.TextResult(string(b)), nil, nil
	})

	// edge_list
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "edge_list",
		Description: "List graph edges by project ID (optional from/to filter)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID string `json:"project_id" jsonschema:"Project ID"`
		FromID    string `json:"from_id,omitempty" jsonschema:"Filter by source node ID"`
		ToID      string `json:"to_id,omitempty" jsonschema:"Filter by target node ID"`
	}) (*mcp.CallToolResult, any, error) {
		items, err := s.ListEdges(params.ProjectID, params.FromID, params.ToID)
		if err != nil {
			return mcputil.TextResult("query failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(items)
		return mcputil.TextResult(string(b)), nil, nil
	})

	// edge_delete
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "edge_delete",
		Description: "Delete a graph edge",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ID string `json:"id" jsonschema:"Edge ID"`
	}) (*mcp.CallToolResult, any, error) {
		if err := s.DeleteEdge(params.ID); err != nil {
			return mcputil.TextResult("delete failed: " + err.Error()), nil, nil
		}
		return mcputil.TextResult("deleted"), nil, nil
	})
}
