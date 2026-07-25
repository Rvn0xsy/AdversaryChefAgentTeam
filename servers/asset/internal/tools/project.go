package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/asset/internal/models"
	"adversarychef/asset/internal/store"
	"adversarychef/mcputil"
)

type createProjectParams struct {
	Name        string `json:"name" jsonschema:"Project name"`
	Description string `json:"description,omitempty" jsonschema:"Project description"`
	Status      string `json:"status" jsonschema:"Project status, e.g. active/completed/archived"`
}

type updateProjectParams struct {
	ID          string `json:"id" jsonschema:"Project ID"`
	Name        string `json:"name,omitempty" jsonschema:"Project name"`
	Description string `json:"description,omitempty" jsonschema:"Project description"`
	Status      string `json:"status,omitempty" jsonschema:"Project status"`
}

func registerProjects(server *mcp.Server, s store.Store) {
	ListerAll(server, "list_projects", "List all penetration testing projects", s.ListProjects)
	Getter(server, "get_project", "Get project details", "Project", s.GetProject)
	Deleter(server, "delete_project", "Delete a project", "Project", s.DeleteProject)

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "create_project",
		Description: "Create a new penetration testing project",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params createProjectParams) (*mcp.CallToolResult, any, error) {
		p := &models.Project{
			Name:        params.Name,
			Description: params.Description,
			Status:      params.Status,
		}
		if err := s.CreateProject(p); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(p)
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "update_project",
		Description: "Update project information",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params updateProjectParams) (*mcp.CallToolResult, any, error) {
		existing, err := s.GetProject(params.ID)
		if err != nil {
			return mcputil.TextResult("Project not found: " + err.Error()), nil, nil
		}
		if params.Name != "" {
			existing.Name = params.Name
		}
		if params.Description != "" {
			existing.Description = params.Description
		}
		if params.Status != "" {
			existing.Status = params.Status
		}
		if err := s.UpdateProject(existing); err != nil {
			return mcputil.TextResult("update failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(existing)
		return mcputil.TextResult(string(b)), nil, nil
	})
}
