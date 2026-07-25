package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/kali/internal/job"
	"adversarychef/mcputil"
)

func registerListJobs(server *mcp.Server, mgr *job.Manager) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name: "list_jobs",
		Description: "List all jobs. Optionally filter by status: running/completed/failed/killed/timed_out.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		Status string `json:"status,omitempty" jsonschema:"Filter by status: running/completed/failed/killed/timed_out, omit for all"`
	}) (*mcp.CallToolResult, any, error) {
		jobs := mgr.List(job.Status(params.Status))
		b, _ := json.Marshal(jobs)
		return mcputil.TextResult(string(b)), nil, nil
	})
}

func registerGetJob(server *mcp.Server, mgr *job.Manager) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name: "get_job",
		Description: "Get job details including status and output. Partial output is available while the job is running.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		JobID string `json:"job_id" jsonschema:"Job ID"`
	}) (*mcp.CallToolResult, any, error) {
		j, err := mgr.Get(params.JobID)
		if err != nil {
			return mcputil.TextResult("not found: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(j)
		return mcputil.TextResult(string(b)), nil, nil
	})
}
