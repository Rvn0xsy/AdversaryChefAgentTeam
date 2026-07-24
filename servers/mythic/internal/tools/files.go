package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/mcputil"
	"adversarychef/mythic/internal/client"
)

func registerFiles(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mythic_list_files",
		Description: "List files stored in Mythic",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		Limit int `json:"limit,omitempty" jsonschema:"Max files to return, default 100"`
	}) (*mcp.CallToolResult, any, error) {
		limit := params.Limit
		if limit <= 0 {
			limit = 100
		}
		res, err := c.GetFiles(ctx, &client.GetFilesInput{Limit: limit})
		if err != nil {
			return mcputil.TextResult("failed to list files: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res.Files, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mythic_upload_file",
		Description: "Upload a local file to Mythic",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		FilePath    string `json:"file_path" jsonschema:"Local file path to upload"`
		Comment     string `json:"comment,omitempty" jsonschema:"Optional comment for the file"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := c.UploadFile(ctx, &client.UploadFileInput{
			FilePath: params.FilePath,
			Comment:  params.Comment,
		})
		if err != nil {
			return mcputil.TextResult("upload failed: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mythic_download_file",
		Description: "Download a file from Mythic by file ID and save locally",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		FileID   string `json:"file_id" jsonschema:"Mythic file ID (agent_file_id)"`
		SavePath string `json:"save_path" jsonschema:"Local path to save the downloaded file"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := c.DownloadFile(ctx, &client.DownloadFileInput{
			FileID:   params.FileID,
			SavePath: params.SavePath,
		})
		if err != nil {
			return mcputil.TextResult("download failed: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mythic_delete_file",
		Description: "Delete a file from Mythic",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		FileID string `json:"file_id" jsonschema:"Mythic file ID to delete"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := c.DeleteFile(ctx, &client.DeleteFileInput{FileID: params.FileID})
		if err != nil {
			return mcputil.TextResult("delete failed: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})
}
