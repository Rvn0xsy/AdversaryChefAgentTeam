package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/asset/internal/models"
	"adversarychef/asset/internal/store"
	"adversarychef/mcputil"
)

type createCredentialParams struct {
	ProjectID      string `json:"project_id" jsonschema:"Project ID"`
	AssetID        string `json:"asset_id,omitempty" jsonschema:"Associated asset ID"`
	CredentialType string `json:"credential_type" jsonschema:"Credential type, e.g. ssh_key/password/api_key/token"`
	Label          string `json:"label" jsonschema:"Credential label/identifier"`
	Value          string `json:"value" jsonschema:"Credential value (sensitive, not logged)"`
	ExpiresAt      string `json:"expires_at,omitempty" jsonschema:"Expiration time"`
	Notes          string `json:"notes,omitempty" jsonschema:"Notes"`
}

type updateCredentialParams struct {
	ID        string `json:"id" jsonschema:"Credential ID"`
	AssetID   string `json:"asset_id,omitempty" jsonschema:"Associated asset ID"`
	Label     string `json:"label,omitempty" jsonschema:"Credential label"`
	Value     string `json:"value,omitempty" jsonschema:"Credential value"`
	ExpiresAt string `json:"expires_at,omitempty" jsonschema:"Expiration time"`
	Notes     string `json:"notes,omitempty" jsonschema:"Notes"`
}

func registerCredentials(server *mcp.Server, s store.Store) {
	Lister(server, "list_credentials", "List all credentials by project ID", "Credential", s.ListCredentials)
	Getter(server, "get_credential", "Get credential details", "Credential", s.GetCredential)
	Deleter(server, "delete_credential", "Delete a credential", "Credential", s.DeleteCredential)

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "create_credential",
		Description: "Create a new credential record",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params createCredentialParams) (*mcp.CallToolResult, any, error) {
		c := &models.Credential{
			ProjectID:      params.ProjectID,
			AssetID:        params.AssetID,
			CredentialType: params.CredentialType,
			Label:          params.Label,
			Value:          params.Value,
			ExpiresAt:      params.ExpiresAt,
			Notes:          params.Notes,
		}
		if err := s.CreateCredential(c); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(c)
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "update_credential",
		Description: "Update credential information",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params updateCredentialParams) (*mcp.CallToolResult, any, error) {
		existing, err := s.GetCredential(params.ID)
		if err != nil {
			return mcputil.TextResult("Credential not found: " + err.Error()), nil, nil
		}
		if params.AssetID != "" {
			existing.AssetID = params.AssetID
		}
		if params.Label != "" {
			existing.Label = params.Label
		}
		if params.Value != "" {
			existing.Value = params.Value
		}
		if params.ExpiresAt != "" {
			existing.ExpiresAt = params.ExpiresAt
		}
		if params.Notes != "" {
			existing.Notes = params.Notes
		}
		if err := s.UpdateCredential(existing); err != nil {
			return mcputil.TextResult("update failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(existing)
		return mcputil.TextResult(string(b)), nil, nil
	})
}
