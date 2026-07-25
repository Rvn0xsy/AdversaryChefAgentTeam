package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
		Description: "Get job details including status and output. For polling wait, prefer job_wait instead.",
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

func registerJobWait(server *mcp.Server, mgr *job.Manager) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name: "job_wait",
		Description: "Wait for a job to complete (blocks until done or timeout). Use this instead of polling get_job repeatedly — it consumes only 1 turn.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		JobID      string `json:"job_id" jsonschema:"Job ID to wait for"`
		TimeoutSec int    `json:"timeout_secs,omitempty" jsonschema:"Max wait time in seconds (default 300)"`
	}) (*mcp.CallToolResult, any, error) {
		timeout := 300 * time.Second
		if params.TimeoutSec > 0 {
			timeout = time.Duration(params.TimeoutSec) * time.Second
		}

		deadline := time.After(timeout)
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()

		for {
			select {
			case <-ctx.Done():
				return mcputil.TextResult("cancelled"), nil, nil
			case <-deadline:
				// Return partial result on timeout
				j, err := mgr.Get(params.JobID)
				if err != nil {
					return mcputil.TextResult(fmt.Sprintf("job not found after %v: %v", timeout, err)), nil, nil
				}
				b, _ := json.Marshal(j)
				return mcputil.TextResult("timeout waiting for job:\n" + string(b)), nil, nil
			case <-tick.C:
				j, err := mgr.Get(params.JobID)
				if err != nil {
					return mcputil.TextResult("job not found: " + err.Error()), nil, nil
				}
				switch j.Status {
				case "completed", "failed", "killed", "timed_out":
					b, _ := json.Marshal(j)
					return mcputil.TextResult(string(b)), nil, nil
				}
				// Still running — keep waiting
			}
		}
	})
}
