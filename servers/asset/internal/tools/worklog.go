package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/asset/internal/models"
	"adversarychef/asset/internal/store"
	"adversarychef/mcputil"
)

type createWorkLogParams struct {
	ProjectID string `json:"project_id" jsonschema:"Project ID"`
	Title     string `json:"title" jsonschema:"Log title"`
	Content   string `json:"content,omitempty" jsonschema:"Log content"`
}

type updateWorkLogParams struct {
	ID      string `json:"id" jsonschema:"Log ID"`
	Title   string `json:"title,omitempty" jsonschema:"Log title"`
	Content string `json:"content,omitempty" jsonschema:"Log content"`
}

func registerWorkLogs(server *mcp.Server, s store.Store) {
	Lister(server, "list_work_logs", "List all work logs by project ID", "Work log", s.ListWorkLogs)
	Getter(server, "get_work_log", "Get work log details", "Work log", s.GetWorkLog)
	Deleter(server, "delete_work_log", "Delete a work log", "Work log", s.DeleteWorkLog)

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "create_work_log",
		Description: "Create a new work log record",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params createWorkLogParams) (*mcp.CallToolResult, any, error) {
		w := &models.WorkLog{
			ProjectID: params.ProjectID,
			Title:     params.Title,
			Content:   params.Content,
		}
		if err := s.CreateWorkLog(w); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(w)
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "update_work_log",
		Description: "Update work log",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params updateWorkLogParams) (*mcp.CallToolResult, any, error) {
		existing, err := s.GetWorkLog(params.ID)
		if err != nil {
			return mcputil.TextResult("Work log not found: " + err.Error()), nil, nil
		}
		if params.Title != "" {
			existing.Title = params.Title
		}
		if params.Content != "" {
			existing.Content = params.Content
		}
		if err := s.UpdateWorkLog(existing); err != nil {
			return mcputil.TextResult("update failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(existing)
		return mcputil.TextResult(string(b)), nil, nil
	})
}
