package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/mcputil"
	"adversarychef/mythic/internal/client"
)

func registerCallbacks(server *mcp.Server, c *client.Client) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "mythic_list_callbacks",
		Description: "List all active agent callbacks from the Mythic C2 server",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		res, err := c.GetCallbacks(ctx, &client.GetCallbacksInput{})
		if err != nil {
			return mcputil.TextResult("failed to list callbacks: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res.Callbacks, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "mythic_get_callback",
		Description: "Get details of a specific callback by display ID",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		CallbackID string `json:"callback_id" jsonschema:"Callback display ID (the number shown in Mythic UI)"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := c.GetCallbacks(ctx, &client.GetCallbacksInput{})
		if err != nil {
			return mcputil.TextResult("failed to get callback: " + err.Error()), nil, nil
		}
		for _, cb := range res.Callbacks {
			if cb.ID == params.CallbackID {
				b, _ := json.MarshalIndent(cb, "", "  ")
				return mcputil.TextResult(string(b)), nil, nil
			}
		}
		return mcputil.TextResult("callback not found: " + params.CallbackID), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "mythic_get_callback_commands",
		Description: "List all loaded commands for a callback",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		CallbackID string `json:"callback_id" jsonschema:"Callback display ID"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := c.GetCallbackCommands(ctx, &client.GetCallbackCommandsInput{CallbackID: params.CallbackID})
		if err != nil {
			return mcputil.TextResult("failed to get commands: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res.Commands, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})
}
