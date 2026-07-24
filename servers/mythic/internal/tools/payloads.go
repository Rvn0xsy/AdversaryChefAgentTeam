package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/mcputil"
	"adversarychef/mythic/internal/client"
)

func registerPayloads(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mythic_create_payload",
		Description: "Create a new payload/agent in Mythic",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		PayloadType string `json:"payload_type" jsonschema:"Payload type (e.g. apollo, hermes, poseidon)"`
		Format      string `json:"format,omitempty" jsonschema:"Output format (exe, dll, shellcode)"`
		Description string `json:"description,omitempty" jsonschema:"Payload description"`
		Filename    string `json:"filename,omitempty" jsonschema:"Output filename"`
		SelectedOS  string `json:"selected_os,omitempty" jsonschema:"Target OS (Linux, Windows, macOS)"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := c.CreatePayload(ctx, &client.CreatePayloadInput{
			PayloadType: params.PayloadType,
			Format:      params.Format,
			Description: params.Description,
			Filename:    params.Filename,
			SelectedOS:  params.SelectedOS,
		})
		if err != nil {
			return mcputil.TextResult("create payload failed: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mythic_get_payload",
		Description: "Get payload details by UUID",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		PayloadUUID string `json:"payload_uuid" jsonschema:"Payload UUID"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := c.GetPayloadByUUID(ctx, params.PayloadUUID)
		if err != nil {
			return mcputil.TextResult("get payload failed: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})
}
