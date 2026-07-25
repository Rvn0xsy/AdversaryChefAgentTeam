package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/asset/internal/models"
	"adversarychef/asset/internal/store"
	"adversarychef/mcputil"
)

type createClueParams struct {
	ProjectID string `json:"project_id" jsonschema:"Project ID"`
	Title     string `json:"title" jsonschema:"Clue title"`
	Content   string `json:"content,omitempty" jsonschema:"Clue details"`
	Type      string `json:"type,omitempty" jsonschema:"Clue type, e.g. vulnerability/info_disclosure/misconfig"`
	Status    string `json:"status,omitempty" jsonschema:"Clue status, e.g. open/confirmed/false_positive/resolved"`
}

type updateClueParams struct {
	ID      string `json:"id" jsonschema:"Clue ID"`
	Title   string `json:"title,omitempty" jsonschema:"Clue title"`
	Content string `json:"content,omitempty" jsonschema:"Clue details"`
	Type    string `json:"type,omitempty" jsonschema:"Clue type"`
	Status  string `json:"status,omitempty" jsonschema:"Clue status"`
}

func registerClues(server *mcp.Server, s store.Store) {
	Lister(server, "list_clues", "List all clues/findings by project ID", "Clue", s.ListClues)
	Getter(server, "get_clue", "Get clue details", "Clue", s.GetClue)
	Deleter(server, "delete_clue", "Delete a clue", "Clue", s.DeleteClue)

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "create_clue",
		Description: "Create a new clue/finding record",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params createClueParams) (*mcp.CallToolResult, any, error) {
		c := &models.Clue{
			ProjectID: params.ProjectID,
			Title:     params.Title,
			Content:   params.Content,
			Type:      params.Type,
			Status:    params.Status,
		}
		if err := s.CreateClue(c); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(c)
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "update_clue",
		Description: "Update clue information",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params updateClueParams) (*mcp.CallToolResult, any, error) {
		existing, err := s.GetClue(params.ID)
		if err != nil {
			return mcputil.TextResult("Clue not found: " + err.Error()), nil, nil
		}
		if params.Title != "" {
			existing.Title = params.Title
		}
		if params.Content != "" {
			existing.Content = params.Content
		}
		if params.Type != "" {
			existing.Type = params.Type
		}
		if params.Status != "" {
			existing.Status = params.Status
		}
		if err := s.UpdateClue(existing); err != nil {
			return mcputil.TextResult("update failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(existing)
		return mcputil.TextResult(string(b)), nil, nil
	})
}
