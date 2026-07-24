package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/asset/internal/models"
	"adversarychef/asset/internal/store"
	"adversarychef/mcputil"
)

type createAssetParams struct {
	ProjectID   string   `json:"project_id" jsonschema:"Project ID"`
	Name        string   `json:"name" jsonschema:"Asset name"`
	IPs         []string `json:"ips,omitempty" jsonschema:"IP address list"`
	Domains     []string `json:"domains,omitempty" jsonschema:"Domain list"`
	TechStack   []string `json:"tech_stack,omitempty" jsonschema:"Tech stack"`
	Scope       string   `json:"scope,omitempty" jsonschema:"Scope of authorization"`
	Description string   `json:"description,omitempty" jsonschema:"Asset description"`
}

type updateAssetParams struct {
	ID          string   `json:"id" jsonschema:"Asset ID"`
	Name        string   `json:"name,omitempty" jsonschema:"Asset name"`
	IPs         []string `json:"ips,omitempty" jsonschema:"IP address list"`
	Domains     []string `json:"domains,omitempty" jsonschema:"Domain list"`
	TechStack   []string `json:"tech_stack,omitempty" jsonschema:"Tech stack"`
	Scope       string   `json:"scope,omitempty" jsonschema:"Scope of authorization"`
	Description string   `json:"description,omitempty" jsonschema:"Asset description"`
}

func registerAssets(server *mcp.Server, s store.Store) {
	Lister(server, "list_assets", "List all target assets by project ID", "Asset", s.ListAssets)
	Getter(server, "get_asset", "Get asset details", "Asset", s.GetAsset)
	Deleter(server, "delete_asset", "Delete an asset", "Asset", s.DeleteAsset)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_asset",
		Description: "Create a new asset record",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params createAssetParams) (*mcp.CallToolResult, any, error) {
		a := &models.Asset{
			ProjectID:   params.ProjectID,
			Name:        params.Name,
			IPs:         params.IPs,
			Domains:     params.Domains,
			TechStack:   params.TechStack,
			Scope:       params.Scope,
			Description: params.Description,
		}
		if err := s.CreateAsset(a); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(a)
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_asset",
		Description: "Update asset information",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params updateAssetParams) (*mcp.CallToolResult, any, error) {
		existing, err := s.GetAsset(params.ID)
		if err != nil {
			return mcputil.TextResult("Asset not found: " + err.Error()), nil, nil
		}
		if params.Name != "" {
			existing.Name = params.Name
		}
		if len(params.IPs) > 0 {
			existing.IPs = params.IPs
		}
		if len(params.Domains) > 0 {
			existing.Domains = params.Domains
		}
		if len(params.TechStack) > 0 {
			existing.TechStack = params.TechStack
		}
		if params.Scope != "" {
			existing.Scope = params.Scope
		}
		if params.Description != "" {
			existing.Description = params.Description
		}
		if err := s.UpdateAsset(existing); err != nil {
			return mcputil.TextResult("update failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(existing)
		return mcputil.TextResult(string(b)), nil, nil
	})
}
