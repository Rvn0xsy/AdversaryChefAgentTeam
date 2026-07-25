package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/nexus/internal/models"
	"adversarychef/nexus/internal/store"
	"adversarychef/mcputil"
)

type searchAssetsParams struct {
	ProjectID string `json:"project_id" jsonschema:"Project ID"`
	Query     string `json:"query" jsonschema:"Search keyword (matches name, IPs, domains, tech stack, description)"`
}

type searchCluesParams struct {
	ProjectID string `json:"project_id" jsonschema:"Project ID"`
	Query     string `json:"query,omitempty" jsonschema:"Search keyword (matches title and content)"`
	Type      string `json:"type,omitempty" jsonschema:"Filter by clue type, e.g. vulnerability/info_disclosure/misconfig"`
	Status    string `json:"status,omitempty" jsonschema:"Filter by status, e.g. open/confirmed/false_positive/resolved"`
}

type projectSummaryParams struct {
	ProjectID string `json:"project_id" jsonschema:"Project ID"`
}

func registerSearch(server *mcp.Server, s store.Store) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "search_assets",
		Description: "Search assets by keyword across name, IPs, domains, tech stack, and description",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params searchAssetsParams) (*mcp.CallToolResult, any, error) {
		items, err := s.SearchAssets(params.ProjectID, params.Query)
		if err != nil {
			return mcputil.TextResult("Search failed: " + err.Error()), nil, nil
		}
		if items == nil {
			items = []models.Asset{}
		}
		b, _ := json.Marshal(items)
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "search_clues",
		Description: "Search clues/findings by keyword, with optional type and status filters",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params searchCluesParams) (*mcp.CallToolResult, any, error) {
		items, err := s.SearchClues(params.ProjectID, params.Query, params.Type, params.Status)
		if err != nil {
			return mcputil.TextResult("Search failed: " + err.Error()), nil, nil
		}
		if items == nil {
			items = []models.Clue{}
		}
		b, _ := json.Marshal(items)
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "project_summary",
		Description: "Get a summary of a project: asset count, clue count by type/status, credential count, worklog count",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params projectSummaryParams) (*mcp.CallToolResult, any, error) {
		ps, err := s.ProjectSummary(params.ProjectID)
		if err != nil {
			return mcputil.TextResult("Summary failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(ps)
		return mcputil.TextResult(string(b)), nil, nil
	})
}
