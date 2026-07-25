package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/mcputil"
)

// Lister registers a list tool that filters by project ID.
func Lister[T any](server *mcp.Server, name, desc, entityName string,
	fn func(string) ([]T, error)) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        name,
		Description: desc,
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID string `json:"project_id" jsonschema:"Project ID"`
	}) (*mcp.CallToolResult, any, error) {
		items, err := fn(params.ProjectID)
		if err != nil {
			return mcputil.TextResult("query failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(items)
		return mcputil.TextResult(string(b)), nil, nil
	})
}

// ListerAll registers a list tool with no project ID filter.
func ListerAll[T any](server *mcp.Server, name, desc string,
	fn func() ([]T, error)) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        name,
		Description: desc,
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		items, err := fn()
		if err != nil {
			return mcputil.TextResult("query failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(items)
		return mcputil.TextResult(string(b)), nil, nil
	})
}

// Getter registers a get-by-ID tool.
func Getter[T any](server *mcp.Server, name, desc, entityName string,
	fn func(string) (*T, error)) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        name,
		Description: desc,
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ID string `json:"id" jsonschema:"Record ID"`
	}) (*mcp.CallToolResult, any, error) {
		item, err := fn(params.ID)
		if err != nil {
			return mcputil.TextResult(entityName + " not found: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(item)
		return mcputil.TextResult(string(b)), nil, nil
	})
}

// Deleter registers a delete-by-ID tool.
func Deleter(server *mcp.Server, name, desc, entityName string,
	fn func(string) error) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        name,
		Description: desc,
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ID string `json:"id" jsonschema:"Record ID"`
	}) (*mcp.CallToolResult, any, error) {
		if err := fn(params.ID); err != nil {
			return mcputil.TextResult("delete failed: " + err.Error()), nil, nil
		}
		return mcputil.TextResult("deleted"), nil, nil
	})
}
