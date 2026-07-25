package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/kali/internal/job"
	"adversarychef/mcputil"
)

func registerKillJob(server *mcp.Server, mgr *job.Manager) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name: "kill_job",
		Description: "Terminate a running job. Force-kills the entire process group to ensure all child processes are cleaned up.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		JobID string `json:"job_id" jsonschema:"Job ID"`
	}) (*mcp.CallToolResult, any, error) {
		if err := mgr.Kill(params.JobID); err != nil {
			return mcputil.TextResult("kill failed: " + err.Error()), nil, nil
		}
		return mcputil.TextResult("job killed: " + params.JobID), nil, nil
	})
}
