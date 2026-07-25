package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/kali/internal/job"
	"adversarychef/mcputil"
)

const defaultTimeout = 30 * time.Minute

// BashParams defines shell command execution parameters.
type BashParams struct {
	Command string `json:"command" jsonschema:"Shell command to execute"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"Timeout in seconds, default 1800 (30 min)"`
}

func registerExec(server *mcp.Server, mgr *job.Manager) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name: "exec",
		Description: "Execute a shell command asynchronously in the Kali Linux container. Returns a job_id. Use get_job to check progress/output, kill_job to stop. Supports nmap, sqlmap, metasploit, gobuster, hydra, and other pentesting tools.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params BashParams) (*mcp.CallToolResult, any, error) {
		timeout := defaultTimeout
		if params.Timeout > 0 {
			timeout = time.Duration(params.Timeout) * time.Second
		}
		jobID := mgr.Start(params.Command, timeout)
		return mcputil.TextResult(fmt.Sprintf(
			"job started: %s\ncommand: %s\ntimeout: %v\nuse get_job to check progress, kill_job to stop.",
			jobID, params.Command, timeout)), nil, nil
	})
}
